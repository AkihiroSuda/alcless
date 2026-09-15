// Copyright The Alcoholless Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package delete

import (
	"errors"
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
	flags := cmd.Flags()
	flags.Bool("secure", false, "securely delete instance data (slow)")
	flags.Bool("keep-home", false, "keep the home directory of the instance user")
	return cmd
}

func action(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	flags := cmd.Flags()
	flagSecure, err := flags.GetBool("secure")
	if err != nil {
		return err
	}
	flagKeepHome, err := flags.GetBool("keep-home")
	if err != nil {
		return err
	}
	if flagSecure && flagKeepHome {
		return errors.New("option --secure conflicts with option --keep-home")
	}
	instName := args[0]
	if err := store.ValidateName(instName); err != nil {
		return err
	}
	inst, err := store.Inspect(ctx, instName)
	if err != nil {
		return err
	}
	if inst == nil {
		slog.WarnContext(ctx, "No such instance", "instance", instName)
		return nil
	}
	cmds, err := userutil.DeleteUserCmds(ctx, inst.User, userutil.DeleteOpts{
		Secure:   flagSecure,
		KeepHome: flagKeepHome,
	})
	if err != nil {
		return err
	}
	if err := cmdutil.RunWithCobra(ctx, cmds, cmd); err != nil {
		return err
	}
	if err := store.Remove(instName); err != nil {
		return err
	}
	// The group is shared by every instance, so it is removed only after the
	// last one is gone.
	groupCmds, err := userutil.DeleteGroupIfEmptyCmds(ctx)
	if err != nil {
		return err
	}
	if len(groupCmds) > 0 {
		if err := cmdutil.RunWithCobra(ctx, groupCmds, cmd); err != nil {
			return err
		}
	}
	if flagKeepHome {
		slog.InfoContext(ctx, "The home directory was kept, and still consumes the disk space. Remove it manually if it is no longer needed.",
			"instance", instName, "instUser", inst.User, "home", inst.Home)
	}
	return nil
}
