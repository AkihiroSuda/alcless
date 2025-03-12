package delete

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/AkihiroSuda/alcless/pkg/cmdutil"
	"github.com/AkihiroSuda/alcless/pkg/store"
	"github.com/AkihiroSuda/alcless/pkg/userutil"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "delete INSTANCE",
		Aliases:               []string{"remove", "rm"},
		Short:                 "Delete an instance",
		Args:                  cobra.ExactArgs(1),
		RunE:                  action,
		DisableFlagsInUseLine: true,
	}
	return cmd
}

func action(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	instName := args[0]
	if err := store.ValidateName(instName); err != nil {
		return err
	}
	instUser := userutil.UserFromInstance(instName)
	instUserExists, err := userutil.Exists(instUser)
	if err != nil {
		return err
	}
	if !instUserExists {
		slog.WarnContext(ctx, "No such instance", "instance", instName, "instUser", instUser)
		return nil
	}
	cmds, err := userutil.DeleteUserCmds(ctx, instUser)
	return cmdutil.RunWithCobra(ctx, cmds, cmd)
}
