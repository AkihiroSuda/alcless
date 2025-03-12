package create

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/AkihiroSuda/alcless/pkg/brew"
	"github.com/AkihiroSuda/alcless/pkg/cmdutil"
	"github.com/AkihiroSuda/alcless/pkg/store"
	"github.com/AkihiroSuda/alcless/pkg/userutil"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "create [INSTANCE]",
		Short:                 "Create an instance",
		Args:                  cobra.MaximumNArgs(1),
		RunE:                  action,
		DisableFlagsInUseLine: true,
	}
	return cmd
}

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
	instName := "default"
	if len(args) != 0 {
		instName = args[0]
	}
	if err = store.ValidateName(instName); err != nil {
		return err
	}
	instUser := userutil.UserFromInstance(instName)
	instUserExists, err := userutil.Exists(instUser)
	if err != nil {
		return err
	}
	if instUserExists {
		slog.InfoContext(ctx, "Already exists", "instance", instName, "instUser", instUser)
	} else {
		slog.InfoContext(ctx, "Creating an instance", "instance", instName, "instUser", instUser)
		cmds, err := userutil.AddUserCmds(ctx, instUser, flagTty)
		if err != nil {
			return err
		}
		if err := cmdutil.RunWithCobra(ctx, cmds, cmd); err != nil {
			return err
		}
	}
	if !flagPlain {
		if err = brew.Installed(ctx, instUser); err == nil {
			slog.InfoContext(ctx, "Homebrew is already installed", "instance", instName, "instUser", instUser)
		} else {
			slog.DebugContext(ctx, "Homebrew is not installed", "instance", instName, "instUser", instUser, "error", err)
			slog.InfoContext(ctx, "Installing Homebrew (If you are seeing an error, do NOT report it to the upstream Homebrew)", "instance", instName, "instUser", instUser)
			cmds := brew.InstallCmds(ctx, instUser)
			if err = cmdutil.RunWithCobra(ctx, cmds, cmd); err != nil {
				return err
			}
			if err = brew.Installed(ctx, instUser); err != nil {
				return fmt.Errorf("failed to detect Homebrew: %w", err)
			}
		}
	}
	return nil
}
