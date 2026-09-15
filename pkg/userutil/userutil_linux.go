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
	"strconv"
	"strings"

	"github.com/AkihiroSuda/alcless/pkg/sudo"
)

// homeDirBase is the parent directory of the home directories.
const homeDirBase = "/home"

// minUID is the lowest UID to allocate.
// The UIDs below 1000 are typically reserved for the system accounts.
const minUID = 1000

// passwdEntries parses the output of `getent passwd`.
// The fields are: name:password:UID:GID:GECOS:home:shell
func passwdEntries(b []byte) [][]string {
	var res [][]string
	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), ":")
		if len(fields) < 7 || fields[0] == "" {
			continue
		}
		res = append(res, fields)
	}
	return res
}

func getent(ctx context.Context, args ...string) ([]byte, error) {
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "getent", args...)
	cmd.Stderr = &stderr
	slog.DebugContext(ctx, "Running command", "cmd", cmd.Args)
	b, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run %v: %w (stderr=%q)", cmd.Args, err, stderr.String())
	}
	return b, nil
}

func Users(ctx context.Context) ([]Entry, error) {
	b, err := getent(ctx, "passwd")
	if err != nil {
		return nil, err
	}
	var res []Entry
	for _, fields := range passwdEntries(b) {
		res = append(res, Entry{Name: fields[0], Label: fields[4]})
	}
	return res, nil
}

func UIDs(ctx context.Context) (map[int]struct{}, error) {
	b, err := getent(ctx, "passwd")
	if err != nil {
		return nil, err
	}
	res := make(map[int]struct{})
	for _, fields := range passwdEntries(b) {
		uid, err := strconv.Atoi(fields[2])
		if err != nil {
			slog.DebugContext(ctx, "Ignoring unparsable UID", "value", fields[2], "error", err)
			continue
		}
		res[uid] = struct{}{}
	}
	return res, nil
}

func GroupMembers(ctx context.Context) (map[string]struct{}, error) {
	res := make(map[string]struct{})
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "getent", "group", GroupName)
	cmd.Stderr = &stderr
	slog.DebugContext(ctx, "Running command", "cmd", cmd.Args)
	b, err := cmd.Output()
	if err != nil {
		// getent exits with 2 when the key is not found, i.e., no instance has
		// been created yet.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			slog.DebugContext(ctx, "No such group", "group", GroupName, "error", err)
			return res, nil
		}
		return nil, fmt.Errorf("failed to run %v: %w (stderr=%q)", cmd.Args, err, stderr.String())
	}
	// "alcless:x:1001:u1002,u1003"
	line := strings.TrimRight(string(b), "\n")
	fields := strings.Split(line, ":")
	if len(fields) < 4 {
		return res, nil
	}
	for _, f := range strings.Split(fields[3], ",") {
		if f != "" {
			res[f] = struct{}{}
		}
	}
	return res, nil
}

func ReadAttribute(ctx context.Context, username string, k Attribute) (string, error) {
	switch k {
	case AttributeUserShell:
		// os/user does not expose the shell, so parse /etc/passwd via getent.
		b, err := getent(ctx, "passwd", username)
		if err != nil {
			return "", err
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

func AddUserCmds(ctx context.Context, instUser, label string, uid int, home string, _ bool) ([]*exec.Cmd, error) {
	sudoersContent, err := sudo.Sudoers(instUser, label)
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
	return []*exec.Cmd{
		// -f: exit successfully if the group already exists
		exec.CommandContext(ctx, "sudo", "groupadd", "-f", GroupName),
		exec.CommandContext(ctx, "sudo", "useradd", "-s", "/bin/bash", "--create-home",
			"--home-dir", home, "--uid", strconv.Itoa(uid), "-c", label, "-G", GroupName, instUser),
		exec.CommandContext(ctx, "sudo", "chmod", "go-rx", home),
		exec.CommandContext(ctx, "sudo", "sh", "-c", sudoersCmd),
	}, nil
}

// deleteGroupCmds returns the commands to remove the [GroupName] group.
func deleteGroupCmds(ctx context.Context) []*exec.Cmd {
	return []*exec.Cmd{
		exec.CommandContext(ctx, "sudo", "groupdel", GroupName),
	}
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
	// userdel removes the supplementary group memberships too.
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
