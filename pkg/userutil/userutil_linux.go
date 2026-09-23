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

	"github.com/AkihiroSuda/alcless/pkg/sudo"
)

func Users(ctx context.Context) ([]string, error) {
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "getent", "passwd")
	cmd.Stderr = &stderr
	slog.DebugContext(ctx, "Running command", "cmd", cmd.Args)
	b, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run %v: %w (stderr=%q)", cmd.Args, err, stderr.String())
	}
	var res []string
	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		line := scanner.Text()
		if i := strings.IndexByte(line, ':'); i > 0 {
			res = append(res, line[:i])
		}
	}
	return res, scanner.Err()
}

// getentKeyNotFoundExitCode is the exit code getent(1) uses when the requested
// key does not exist in the database (as opposed to a usage error or an
// enumeration-not-supported error).
const getentKeyNotFoundExitCode = 2

func GroupUsers(ctx context.Context, group string) ([]string, error) {
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "getent", "group", group)
	cmd.Stderr = &stderr
	slog.DebugContext(ctx, "Running command", "cmd", cmd.Args)
	b, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == getentKeyNotFoundExitCode {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to run %v: %w (stderr=%q)", cmd.Args, err, stderr.String())
	}
	gid, members, err := parseGetentGroup(strings.TrimRight(string(b), "\n"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse getent group output for %q: %w", group, err)
	}

	var passwdStderr bytes.Buffer
	passwdCmd := exec.CommandContext(ctx, "getent", "passwd")
	passwdCmd.Stderr = &passwdStderr
	slog.DebugContext(ctx, "Running command", "cmd", passwdCmd.Args)
	passwdOut, err := passwdCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run %v: %w (stderr=%q)", passwdCmd.Args, err, passwdStderr.String())
	}
	primary := usersWithPrimaryGID(passwdOut, gid)

	return dedup(append(members, primary...)), nil
}

// parseGetentGroup parses one line of `getent group <name>` output
// (`name:password:gid:member1,member2,...`) into the group's gid and its
// comma-separated (supplementary) member list.
func parseGetentGroup(line string) (gid string, members []string, err error) {
	fields := strings.Split(line, ":")
	if len(fields) < 4 {
		return "", nil, fmt.Errorf("unexpected getent group entry: %q", line)
	}
	gid = fields[2]
	if fields[3] != "" {
		members = strings.Split(fields[3], ",")
	}
	return gid, members, nil
}

// usersWithPrimaryGID scans `getent passwd` output
// (`name:password:uid:gid:gecos:home:shell`) and returns the usernames whose
// primary group (4th field) equals gid.
func usersWithPrimaryGID(passwd []byte, gid string) []string {
	var res []string
	scanner := bufio.NewScanner(bytes.NewReader(passwd))
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), ":")
		if len(fields) < 4 {
			continue
		}
		if fields[3] == gid {
			res = append(res, fields[0])
		}
	}
	return res
}

// dedup returns ss with duplicate elements removed, preserving the order of
// first occurrence.
func dedup(ss []string) []string {
	if ss == nil {
		return nil
	}
	seen := make(map[string]struct{}, len(ss))
	res := make([]string, 0, len(ss))
	for _, s := range ss {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		res = append(res, s)
	}
	return res
}

func ReadAttribute(_ context.Context, username string, k Attribute) (string, error) {
	switch k {
	case AttributeUserShell:
		// os/user does not expose the shell, so parse /etc/passwd via getent.
		var stderr bytes.Buffer
		cmd := exec.Command("getent", "passwd", username)
		cmd.Stderr = &stderr
		b, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to run %v: %w (stderr=%q)", cmd.Args, err, stderr.String())
		}
		line := strings.TrimRight(string(b), "\n")
		fields := strings.Split(line, ":")
		if len(fields) < 7 {
			return "", fmt.Errorf("unexpected passwd entry for %q: %q", username, line)
		}
		return fields[6], nil
	}
	return "", fmt.Errorf("unsupported attribute %q", k)
}

func AddUserCmds(ctx context.Context, instUser string, _ bool) ([]*exec.Cmd, error) {
	sudoersContent, err := sudo.Sudoers(instUser)
	if err != nil {
		return nil, err
	}
	sudoersPath, err := sudo.SudoersPath(instUser)
	if err != nil {
		return nil, err
	}
	sudoersCmd := fmt.Sprintf("echo '%s' >'%s'", sudoersContent, sudoersPath)
	// The user is accessed via `sudo /usr/bin/su -` (NOPASSWD), so no password is set.
	// useradd leaves the password locked by default, which is what we want.
	home := "/home/" + instUser
	return []*exec.Cmd{
		exec.CommandContext(ctx, "sudo", "useradd", "-s", "/bin/bash", "--create-home", "--home-dir", home, instUser),
		exec.CommandContext(ctx, "sudo", "chmod", "go-rx", home),
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
	if opts.Secure {
		slog.WarnContext(ctx, "The --secure flag is not implemented on Linux; falling back to a normal deletion", "user", instUser)
	}
	userdelArgs := []string{"userdel"}
	if !opts.KeepHome {
		userdelArgs = append(userdelArgs, "--remove")
	}
	userdelArgs = append(userdelArgs, instUser)
	return []*exec.Cmd{
		exec.CommandContext(ctx, "sudo", userdelArgs...),
		exec.CommandContext(ctx, "sudo", "rm", "-f", sudoersPath),
	}, nil
}

func GroupSetupCmds(ctx context.Context, instUser, groupName string) ([]*exec.Cmd, error) {
	sudoersContent, err := sudo.Sudoers(instUser)
	if err != nil {
		return nil, err
	}
	sudoersPath, err := sudo.SudoersPath(instUser)
	if err != nil {
		return nil, err
	}

	cmds := []*exec.Cmd{
		exec.CommandContext(ctx, "sudo", "chmod", "go-rx", filepath.Join("/home", instUser)),
		exec.CommandContext(ctx, "sudo", "sh", "-c", fmt.Sprintf("mkdir -p /etc/sudoers.d && echo '%s' >'%s' && chmod 440 '%s'", sudoersContent, sudoersPath, sudoersPath)),
	}

	if groupName != "" {
		cmds = append(cmds,
			// -f: exit success if the group already exists.
			exec.CommandContext(ctx, "sudo", "groupadd", "-f", groupName),
			// -aG: append to the supplementary group list; a no-op if the user is already a member.
			exec.CommandContext(ctx, "sudo", "usermod", "-aG", groupName, instUser),
		)
	}

	return cmds, nil
}
