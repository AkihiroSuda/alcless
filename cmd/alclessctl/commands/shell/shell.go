package shell

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/AkihiroSuda/alcless/pkg/cmdutil"
	"github.com/AkihiroSuda/alcless/pkg/rsync"
	"github.com/AkihiroSuda/alcless/pkg/store"
	"github.com/AkihiroSuda/alcless/pkg/sudo"
	"github.com/AkihiroSuda/alcless/pkg/userutil"
)

const example = `
  Run commands (long form):
  $ cd ~/SOME_DIRECTORY
  $ alclessctl shell default brew install xz
  $ alclessctl shell default xz SOME_FILE

  Run commands (short form):
  $ cd ~/SOME_DIRECTORY
  $ alcless brew install xz
  $ alcless xz SOME_FILE`

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "shell INSTANCE COMMAND [ARGS]...",
		Short:                 "Run a command in an instance",
		Example:               example,
		Args:                  cobra.MinimumNArgs(1),
		RunE:                  action,
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.SetInterspersed(false)
	flags.String("workdir", "", "specify working directory")
	flags.Bool("read-only", false, "disable syncing back modified files")

	return cmd
}

// Depth of "/Users/USER" is 3.
const rsyncMinimumSrcDirDepth = 4

func action(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	flags := cmd.Flags()
	flagTty, err := flags.GetBool("tty")
	if err != nil {
		return err
	}
	flagPlain, err := flags.GetBool("plain")
	if err != nil {
		return err
	}
	flagReadOnly, err := flags.GetBool("read-only")
	if err != nil {
		return err
	}
	instName := args[0]
	if err = store.ValidateName(instName); err != nil {
		return err
	}
	instUser := userutil.UserFromInstance(instName)
	instUserInfo, err := user.Lookup(instUser)
	if err != nil {
		var uee user.UnknownUserError
		if errors.As(err, &uee) {
			// TODO: run the `alclessctl create` command automatically
			slog.DebugContext(ctx, "user does not exist", "user", instUser, "error", err)
			return fmt.Errorf("instance %q does not exist (Hint: run `alclessctl create %s` first)", instName, instName)
		}
		return fmt.Errorf("failed to get user %q: %w", instUser, err)
	}

	var (
		cmdExe  string
		cmdArgs []string
	)
	if len(args) > 1 {
		cmdExe = args[1]
		cmdArgs = args[2:]
	} else {
		cmdExe, err = userutil.ReadAttribute(ctx, instUser, userutil.AttributeUserShell)
		if err != nil {
			return err
		}
		if cmdExe == "" {
			slog.WarnContext(ctx, "no shell was found, falling back to /bin/sh", "instUser", instUser)
			cmdExe = "/bin/sh"
		}
	}

	instUserHome := instUserInfo.HomeDir
	if instUserHome == "" {
		return fmt.Errorf("failed to detect the home directory of the user %q", instUser)
	}

	hostWD, err := os.Getwd()
	if err != nil {
		return err
	}
	guestWD := filepath.Join(instUserHome, hostWD)
	flagWorkdir, err := flags.GetString("workdir")
	if err != nil {
		return err
	}
	if flagWorkdir != "" {
		guestWD = flagWorkdir
	}

	if !flagPlain {
		const hint = "cd to a deeper directory, or run `alclessctl shell` with `--plain`"
		hostHome, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		if hostWD == hostHome {
			return fmt.Errorf("the host working directory must not be $HOME, as this directory is being rsynced to the instance (Hint: %s)", hint)
		} else {
			srcWdDepth := len(strings.Split(hostWD, string(os.PathSeparator)))
			// Depth of "/Users/USER" is 3
			slog.DebugContext(ctx, "Working directory depth", "wd", hostWD, "depth", srcWdDepth)
			if srcWdDepth < rsyncMinimumSrcDirDepth {
				return fmt.Errorf("expected the depth of the host working directory (%q) to be more than %d, only got %d (Hint: %s)",
					hostWD, rsyncMinimumSrcDirDepth, srcWdDepth, hint)
			}
		}
		rsyncSrc := hostWD + string(os.PathSeparator)
		rsyncDst := instName + ":" + guestWD
		slog.InfoContext(ctx, "➡️Syncing the files", "src", rsyncSrc, "dst", rsyncDst)
		rsyncCmd, err := rsync.Cmd(ctx, instName, rsyncSrc, rsyncDst)
		if err != nil {
			return err
		}
		rsyncCmds := []*exec.Cmd{
			sudo.Cmd(ctx, instUser, "", "mkdir", []string{"-p", "-m", "700", guestWD}),
			rsyncCmd,
		}
		rsyncCmdOpts, err := cmdutil.RunOptsFromCobraNoStdin(cmd)
		if err != nil {
			return err
		}
		if err = cmdutil.Run(ctx, rsyncCmds, rsyncCmdOpts); err != nil {
			return fmt.Errorf("%w (Hint: run with `alclessctl shell --plain` as a workaround)", err)
		}
	}

	sudoCmd := sudo.Cmd(ctx, instUser, guestWD, cmdExe, cmdArgs)
	sudoCmdOpts, err := cmdutil.RunOptsFromCobra(cmd) // Propagate stdin
	if err != nil {
		return err
	}
	sudoCmdOpts.Confirm = false // Not a privileged operation
	sudoCmdErr := cmdutil.Run(ctx, []*exec.Cmd{sudoCmd}, sudoCmdOpts)
	if sudoCmdErr != nil {
		slog.ErrorContext(ctx, sudoCmdErr.Error())
	}

	if !flagPlain && !flagReadOnly {
		rsyncSrc := instName + ":" + guestWD + string(os.PathSeparator)
		rsyncDst := hostWD
		var dryRunResultWasEmpty bool
		if flagTty {
			slog.InfoContext(ctx, "⬅️Syncing the files back (dry run)", "src", rsyncSrc, "dst", rsyncDst)
			rsyncCmd, err := rsync.Cmd(ctx, instName, rsyncSrc, rsyncDst, rsync.WithDryRun())
			if err != nil {
				return err
			}
			// dry run does not need confirmation input
			rsyncCmdOpts, err := cmdutil.RunOptsFromCobraNoStdin(cmd)
			if err != nil {
				return err
			}
			var dryRunStdout bytes.Buffer
			rsyncCmdOpts.Stdout = io.MultiWriter(rsyncCmdOpts.Stdout, &dryRunStdout)
			if err = cmdutil.Run(ctx, []*exec.Cmd{rsyncCmd}, rsyncCmdOpts); err != nil {
				return err
			}
			dryRunResultWasEmpty = strings.TrimSpace(dryRunStdout.String()) == ""
			// Confirmation prompt will be shown for the non-dry run
			// TODO: print a warning if rsyncSrc is newer than rsyncDst
		}
		if dryRunResultWasEmpty {
			slog.InfoContext(ctx, "⬅️Nothing to sync back", "src", rsyncSrc, "dst", rsyncDst)
		} else {
			slog.InfoContext(ctx, "⬅️Syncing the files back", "src", rsyncSrc, "dst", rsyncDst)
			rsyncCmd, err := rsync.Cmd(ctx, instName, rsyncSrc, rsyncDst)
			if err != nil {
				return err
			}
			if err = cmdutil.RunWithCobra(ctx, []*exec.Cmd{rsyncCmd}, cmd); err != nil {
				return err
			}
		}
		// TODO: create Homebrew wrappers (~alcless_USER_default/brew/bin/foo -> ~/.alcless/default/bin/foo)
	}

	return sudoCmdErr
}
