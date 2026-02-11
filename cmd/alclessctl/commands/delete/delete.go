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
