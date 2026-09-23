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
	"path/filepath"
	"strings"

	"github.com/sethvargo/go-password/password"

	"github.com/AkihiroSuda/alcless/pkg/sudo"
)

func Users(ctx context.Context) ([]string, error) {
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "dscl", ".", "list", "/Users")
	cmd.Stderr = &stderr
	slog.DebugContext(ctx, "Running command", "cmd", cmd.Args)
	b, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run %v: %w (stderr=%q)", cmd.Args, err, stderr.String())
	}
	var res []string
	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		res = append(res, scanner.Text())
	}
	return res, scanner.Err()
}

func GroupUsers(ctx context.Context, group string) ([]string, error) {
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "dscl", ".", "-read", "/Groups/"+group, "GroupMembership")
	cmd.Stderr = &stderr
	slog.DebugContext(ctx, "Running command", "cmd", cmd.Args)
	b, err := cmd.Output()
	if err != nil {
		if strings.Contains(stderr.String(), "eDSRecordNotFound") {
			// The group does not exist yet: no members.
			return nil, nil
		}
		return nil, fmt.Errorf("failed to run %v: %w (stderr=%q)", cmd.Args, err, stderr.String())
	}
	return parseGroupMembership(b), nil
}

// parseGroupMembership parses `dscl . -read /Groups/<g> GroupMembership` output.
//
// dscl(1) prints "No such key: GroupMembership" (exit 0) when the group exists
// but has no members attribute. Otherwise it prints "GroupMembership:" followed
// by a space-separated value list on the same line, or, per dscl(1), one value
// per indented line when a value contains an embedded space.
func parseGroupMembership(out []byte) []string {
	lines := strings.Split(string(out), "\n")
	var res []string
	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || strings.HasPrefix(trimmed, "No such key:") {
			continue
		}
		rest, ok := strings.CutPrefix(trimmed, "GroupMembership:")
		if !ok {
			continue
		}
		if rest = strings.TrimSpace(rest); rest != "" {
			res = append(res, strings.Fields(rest)...)
			continue
		}
		for i+1 < len(lines) && strings.HasPrefix(lines[i+1], " ") {
			i++
			if v := strings.TrimSpace(lines[i]); v != "" {
				res = append(res, v)
			}
		}
	}
	return res
}

// groupExists reports whether /Groups/<groupName> exists in the local
// directory service.
func groupExists(ctx context.Context, groupName string) (bool, error) {
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "dscl", ".", "-read", "/Groups/"+groupName)
	cmd.Stderr = &stderr
	slog.DebugContext(ctx, "Running command", "cmd", cmd.Args)
	if err := cmd.Run(); err != nil {
		if strings.Contains(stderr.String(), "eDSRecordNotFound") {
			return false, nil
		}
		return false, fmt.Errorf("failed to run %v: %w (stderr=%q)", cmd.Args, err, stderr.String())
	}
	return true, nil
}

func ReadAttribute(ctx context.Context, username string, k Attribute) (string, error) {
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "dscl", ".", "-read", "/Users/"+username, string(k))
	cmd.Stderr = &stderr
	slog.DebugContext(ctx, "Running command", "cmd", cmd.Args)
	b, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run %v: %w (stderr=%q)", cmd.Args, err, stderr.String())
	}
	s := string(b)
	s = strings.TrimPrefix(s, string(k)+":")
	s = strings.TrimSpace(s)
	return s, nil
}

func AddUserCmds(ctx context.Context, instUser string, tty bool) ([]*exec.Cmd, error) {
	sudoersContent, err := sudo.Sudoers(instUser)
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
	return []*exec.Cmd{
		exec.CommandContext(ctx, "sudo", "sysadminctl", "-addUser", instUser, "-password", pw),
		exec.CommandContext(ctx, "sudo", "chmod", "go-rx", filepath.Join("/Users", instUser)),
		exec.CommandContext(ctx, "sudo", "sh", "-c", sudoersCmd),
	}, nil
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
		exec.CommandContext(ctx, "sudo", append([]string{"sysadminctl"}, sysadminctlArgs...)...),
		exec.CommandContext(ctx, "sudo", "rm", "-f", sudoersPath),
	}
	return cmds, nil
}

func GroupSetupCmds(ctx context.Context, instUser, groupName string) ([]*exec.Cmd, error) {
	var exists bool
	if groupName != "" {
		var err error
		if exists, err = groupExists(ctx, groupName); err != nil {
			return nil, err
		}
	}
	return groupSetupCmds(ctx, instUser, groupName, exists)
}

// groupSetupCmds builds the command list for GroupSetupCmds.
// groupAlreadyExists says whether /Groups/<groupName> already exists, so the
// -create command is included only when it is needed. Every command that
// carries groupName passes it as a direct argument, never inside a shell
// string.
func groupSetupCmds(ctx context.Context, instUser, groupName string, groupAlreadyExists bool) ([]*exec.Cmd, error) {
	sudoersContent, err := sudo.Sudoers(instUser)
	if err != nil {
		return nil, err
	}
	sudoersPath, err := sudo.SudoersPath(instUser)
	if err != nil {
		return nil, err
	}

	cmds := []*exec.Cmd{
		exec.CommandContext(ctx, "sudo", "chmod", "go-rx", filepath.Join("/Users", instUser)),
		exec.CommandContext(ctx, "sudo", "sh", "-c", fmt.Sprintf("mkdir -p /etc/sudoers.d && echo '%s' >'%s' && chmod 440 '%s'", sudoersContent, sudoersPath, sudoersPath)),
	}

	if groupName != "" {
		if !groupAlreadyExists {
			cmds = append(cmds, exec.CommandContext(ctx, "sudo", "dscl", ".", "-create", "/Groups/"+groupName))
		}
		// -merge only adds the value if it is not already present, so a repeat call is a no-op.
		cmds = append(cmds, exec.CommandContext(ctx, "sudo", "dscl", ".", "-merge", "/Groups/"+groupName, "GroupMembership", instUser))
	}

	return cmds, nil
}
