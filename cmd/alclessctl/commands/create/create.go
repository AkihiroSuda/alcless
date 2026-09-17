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

package create

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/user"
	"strconv"
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
	inst, err := store.Inspect(ctx, instName)
	if err != nil {
		return err
	}
	if inst != nil {
		slog.InfoContext(ctx, "Already exists", "instance", instName, "instUser", inst.User, "home", inst.Home)
		inst.WarnIfLegacy(ctx)
	} else {
		uid, instUser, home, err := userutil.Allocate(ctx)
		if err != nil {
			return err
		}
		label := userutil.LabelFromInstance(instName)
		slog.InfoContext(ctx, "Creating an instance", "instance", instName, "instUser", instUser, "home", home)
		cmds, err := userutil.AddUserCmds(ctx, instUser, label, uid, home, flagTty)
		if err != nil {
			return err
		}
		if err := cmdutil.RunWithCobra(ctx, cmds, cmd); err != nil {
			return err
		}
		// Record what was actually created, not what was requested: the UID and
		// the home directory are only *hints* to useradd/sysadminctl, and a
		// mismatch would later make the instance unresolvable.
		created, err := user.Lookup(instUser)
		if err != nil {
			return fmt.Errorf("failed to look up the just-created user %q: %w", instUser, err)
		}
		actualUID, err := strconv.Atoi(created.Uid)
		if err != nil {
			return fmt.Errorf("failed to parse the UID %q of the user %q: %w", created.Uid, instUser, err)
		}
		if actualUID != uid || created.HomeDir != home {
			slog.WarnContext(ctx, "The created user does not match what was requested",
				"instUser", instUser, "requestedUID", uid, "actualUID", actualUID,
				"requestedHome", home, "actualHome", created.HomeDir)
			home = created.HomeDir
		}
		inst = &store.Instance{
			Version: store.Version,
			Name:    instName,
			User:    instUser,
			UID:     actualUID,
			Home:    home,
		}
		if err := store.Save(inst); err != nil {
			return err
		}
	}
	if !flagPlain {
		instUser := inst.User
		switch {
		case !brew.Supported():
			slog.WarnContext(ctx, "Homebrew is not supported on this host", "instance", instName, "instUser", instUser)
		default:
			prefix, err := brew.Installed(ctx, instUser, inst.Home)
			if err == nil {
				slog.InfoContext(ctx, "Homebrew is already installed", "instance", instName, "instUser", instUser, "prefix", prefix)
				break
			}
			slog.DebugContext(ctx, "Homebrew is not installed", "instance", instName, "instUser", instUser, "error", err)
			warnLongPrefix(ctx, brew.Prefix(inst.Home))
			slog.InfoContext(ctx, "Installing Homebrew (If you are seeing an error, do NOT report it to the upstream Homebrew)", "instance", instName, "instUser", instUser)
			cmds := brew.InstallCmds(ctx, instUser)
			if err = cmdutil.RunWithCobra(ctx, cmds, cmd); err != nil {
				return err
			}
			if _, err = brew.Installed(ctx, instUser, inst.Home); err != nil {
				return fmt.Errorf("failed to detect Homebrew: %w", err)
			}
		}
	}
	return nil
}

// warnLongPrefix warns that Homebrew will build the formulae from source,
// as the prefix is too long for relocating the official bottles.
func warnLongPrefix(ctx context.Context, prefix string) {
	max := brew.MaxPrefixLen()
	if max <= 0 || len(prefix) <= max {
		return
	}
	slog.WarnContext(ctx, "The Homebrew prefix is too long to pour the official bottles, so the formulae will be built from source (slow).",
		"prefix", prefix, "length", len(prefix), "max", max)
}
