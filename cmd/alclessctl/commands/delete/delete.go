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
	"fmt"
	"log/slog"
	"os/user"
	"slices"

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
	instUser := userutil.UserFromInstance(instName)
	instUserExists, err := userutil.Exists(instUser)
	if err != nil {
		return err
	}
	if !instUserExists {
		slog.WarnContext(ctx, "No such instance", "instance", instName, "instUser", instUser)
		return nil
	}
	if userutil.Mode == "group" {
		members, err := userutil.GroupUsers(ctx, userutil.GroupName())
		if err != nil {
			return err
		}
		if !slices.Contains(members, instUser) {
			return fmt.Errorf("refusing to delete %q: not a member of group %q", instName, userutil.GroupName())
		}
	}
	var instUserHome string
	if flagKeepHome {
		u, err := user.Lookup(instUser)
		if err != nil {
			return err
		}
		instUserHome = u.HomeDir
	}
	cmds, err := userutil.DeleteUserCmds(ctx, instUser, userutil.DeleteOpts{
		Secure:   flagSecure,
		KeepHome: flagKeepHome,
	})
	if err != nil {
		return err
	}
	if err := cmdutil.RunWithCobra(ctx, cmds, cmd); err != nil {
		return err
	}
	if flagKeepHome {
		slog.InfoContext(ctx, "The home directory was kept, and still consumes the disk space. Remove it manually if it is no longer needed.",
			"instance", instName, "instUser", instUser, "home", instUserHome)
	}
	return nil
}
