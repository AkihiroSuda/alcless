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

package userutil

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"os/user"
	"strconv"
	"strings"

	"github.com/sethvargo/go-password/password"

	"github.com/AkihiroSuda/alcless/pkg/sudo"
)

// homeDirBase is the parent directory of the home directories.
const homeDirBase = "/Users"

// minUID is the lowest UID to allocate.
//
// macOS reserves the UIDs below 500 for the system accounts, and 450-499 for
// the role accounts (see `sysadminctl` usage).
const minUID = 501

// dsclOutput parses the two-column output of `dscl . -list <path> <key>`.
// The first field is the record name; the remainder is the value, which may
// contain spaces (e.g., a RealName such as "World Wide Web Server").
func dsclOutput(b []byte) map[string]string {
	res := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		name, value, found := strings.Cut(scanner.Text(), " ")
		if name == "" {
			continue
		}
		if !found {
			res[name] = ""
			continue
		}
		res[name] = strings.TrimSpace(value)
	}
	return res
}

func dscl(ctx context.Context, args ...string) ([]byte, error) {
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "dscl", append([]string{"."}, args...)...)
	cmd.Stderr = &stderr
	slog.DebugContext(ctx, "Running command", "cmd", cmd.Args)
	b, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run %v: %w (stderr=%q)", cmd.Args, err, stderr.String())
	}
	return b, nil
}

func Users(ctx context.Context) ([]Entry, error) {
	b, err := dscl(ctx, "-list", "/Users", "RealName")
	if err != nil {
		return nil, err
	}
	var res []Entry
	for name, realName := range dsclOutput(b) {
		res = append(res, Entry{Name: name, Label: realName})
	}
	return res, nil
}

func UIDs(ctx context.Context) (map[int]struct{}, error) {
	b, err := dscl(ctx, "-list", "/Users", "UniqueID")
	if err != nil {
		return nil, err
	}
	res := make(map[int]struct{})
	for _, v := range dsclOutput(b) {
		uid, err := strconv.Atoi(v)
		if err != nil {
			slog.DebugContext(ctx, "Ignoring unparsable UniqueID", "value", v, "error", err)
			continue
		}
		res[uid] = struct{}{}
	}
	return res, nil
}

// GroupMembers returns the members of the [GroupName] group.
//
// os/user has no API for listing the members of a group, so this shells out to
// dscl, like [Users] does.
func GroupMembers(ctx context.Context) (map[string]struct{}, error) {
	res := make(map[string]struct{})
	if _, err := user.LookupGroup(GroupName); err != nil {
		var uge user.UnknownGroupError
		if errors.As(err, &uge) {
			// No instance has been created yet
			return res, nil
		}
		return nil, err
	}
	b, err := dscl(ctx, "-read", "/Groups/"+GroupName, "GroupMembership")
	if err != nil {
		return nil, err
	}
	// "GroupMembership: u502 u503", or "No such key: GroupMembership" for an empty group
	s := strings.TrimSpace(string(b))
	s, found := strings.CutPrefix(s, "GroupMembership:")
	if !found {
		return res, nil
	}
	for _, f := range strings.Fields(s) {
		res[f] = struct{}{}
	}
	return res, nil
}

func ReadAttribute(ctx context.Context, username string, k Attribute) (string, error) {
	b, err := dscl(ctx, "-read", "/Users/"+username, string(k))
	if err != nil {
		return "", err
	}
	s := string(b)
	s = strings.TrimPrefix(s, string(k)+":")
	s = strings.TrimSpace(s)
	return s, nil
}

// deleteGroupCmds returns the commands to remove the [GroupName] group.
func deleteGroupCmds(ctx context.Context) []*exec.Cmd {
	return []*exec.Cmd{
		exec.CommandContext(ctx, "sudo", "dseditgroup", "-o", "delete", GroupName),
	}
}

func AddUserCmds(ctx context.Context, instUser, label string, uid int, home string, tty bool) ([]*exec.Cmd, error) {
	sudoersContent, err := sudo.Sudoers(instUser, label)
	if err != nil {
		return nil, err
	}
	sudoersPath, err := sudo.SudoersPath(instUser)
	if err != nil {
		return nil, err
	}
	sudoersCmd := fmt.Sprintf("echo '%s' >'%s'", sudoersContent, sudoersPath)
	pw := "-"
	if !tty {
		pw, err = password.Generate(64, 10, 10, false, false)
		if err != nil {
			return nil, err
		}
		slog.WarnContext(ctx, "Generated a random password, as tty is not available. THE PASSWORD IS SHOWN IN THIS SCREEN.", "user", instUser, "password", pw)
	}
	var cmds []*exec.Cmd
	hasGroup, err := groupExists()
	if err != nil {
		return nil, err
	}
	if !hasGroup {
		cmds = append(cmds, exec.CommandContext(ctx, "sudo", "dseditgroup", "-o", "create", GroupName))
	}
	cmds = append(cmds,
		exec.CommandContext(ctx, "sudo", "sysadminctl", "-addUser", instUser,
			"-UID", strconv.Itoa(uid), "-fullName", label, "-home", home, "-password", pw),
		// When -home is specified, sysadminctl only *assigns* the home directory
		// ("Home directory is assigned (not created!)"); it does not create it.
		// createhomedir populates it from the macOS User Template.
		exec.CommandContext(ctx, "sudo", "createhomedir", "-c", "-u", instUser),
		exec.CommandContext(ctx, "sudo", "dseditgroup", "-o", "edit", "-a", instUser, "-t", "user", GroupName),
		exec.CommandContext(ctx, "sudo", "chmod", "go-rx", home),
		exec.CommandContext(ctx, "sudo", "sh", "-c", sudoersCmd),
	)
	return cmds, nil
}

func DeleteUserCmds(ctx context.Context, instUser string, opts DeleteOpts) ([]*exec.Cmd, error) {
	if opts.Secure && opts.KeepHome {
		return nil, errors.New("the Secure option conflicts with the KeepHome option")
	}
	sudoersPath, err := sudo.SudoersPath(instUser)
	if err != nil {
		return nil, err
	}
	// `sysadminctl -deleteUser <user name> [-secure || -keepHome]`
	sysadminctlArgs := []string{"-deleteUser", instUser}
	switch {
	case opts.Secure:
		sysadminctlArgs = append(sysadminctlArgs, "-secure")
	case opts.KeepHome:
		sysadminctlArgs = append(sysadminctlArgs, "-keepHome")
	}
	cmds := []*exec.Cmd{
		// Drop the group membership while the user record still exists.
		// `|| true`, as the user may not be a member of the group, e.g. when a
		// previous `create` failed halfway.
		exec.CommandContext(ctx, "sudo", "sh", "-c",
			fmt.Sprintf("dseditgroup -o edit -d '%s' -t user '%s' || true", instUser, GroupName)),
		exec.CommandContext(ctx, "sudo", append([]string{"sysadminctl"}, sysadminctlArgs...)...),
		exec.CommandContext(ctx, "sudo", "rm", "-f", sudoersPath),
	}
	return cmds, nil
}
