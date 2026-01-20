package create

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

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
	flags := cmd.Flags()
	flags.SetInterspersed(false)
	flags.String("name", "", "Override the instance name")

	return cmd
}

func resolveInstName(args0, flagName string) (string, error) {
	instName := "default"
	if flagName != "" {
		if strings.Contains(flagName, "/") {
			return "", errors.New("value of --name=... must not contain a slash")
		}
		instName = flagName
	}
	if args0 != "" {
		if strings.HasPrefix(args0, "template://") {
			switch args0 {
			case "template://default":
				return instName, nil
			default:
				return "", fmt.Errorf("unknown template: %q (currently, only template://default is available)", args0)
			}
		}
		if args0 != "" && flagName != "" && args0 != flagName {
			return "", fmt.Errorf("instance name %q and CLI flag --name=%q cannot be specified together",
				args0, flagName)
		}
		instName = args0
	}
	return instName, nil
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
	flagName, err := flags.GetString("name")
	if err != nil {
		return err
	}
	var args0 string
	if len(args) > 0 {
		args0 = args[0]
	}
	instName, err := resolveInstName(args0, flagName)
	if err != nil {
		return err
	}
	if err = store.ValidateName(instName); err != nil {
		return err
	}
	instUser := userutil.UserFromInstance(instName)
	instUserExists, err := userutil.Exists(instUser)
	if err != nil {
		return err
	}

	isGroupMode := userutil.Mode == "group"

	if instUserExists {
		if isGroupMode {
			slog.InfoContext(ctx, "Setting up existing user for alcless", "instance", instName, "instUser", instUser)
			cmds, err := userutil.GroupSetupCmds(ctx, instUser, userutil.GroupName(), instName)
			if err != nil {
				return err
			}
			if err := cmdutil.RunWithCobra(ctx, cmds, cmd); err != nil {
				return err
			}
		} else {
			slog.InfoContext(ctx, "Already exists", "instance", instName, "instUser", instUser)
		}
	} else {
		slog.InfoContext(ctx, "Creating an instance", "instance", instName, "instUser", instUser)
		cmds, err := userutil.AddUserCmds(ctx, instUser, flagTty)
		if err != nil {
			return err
		}
		if isGroupMode {
			setupCmds, err := userutil.GroupSetupCmds(ctx, instUser, userutil.GroupName())
			if err != nil {
				return err
			}
			cmds = append(cmds, setupCmds...)
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
