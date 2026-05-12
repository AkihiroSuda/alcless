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
	"fmt"
	"log/slog"
	"os/exec"
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

func DeleteUserCmds(ctx context.Context, instUser string, secure bool) ([]*exec.Cmd, error) {
	sudoersPath, err := sudo.SudoersPath(instUser)
	if err != nil {
		return nil, err
	}
	if secure {
		slog.WarnContext(ctx, "The --secure flag is not implemented on Linux; falling back to a normal deletion", "user", instUser)
	}
	return []*exec.Cmd{
		exec.CommandContext(ctx, "sudo", "userdel", "--remove", instUser),
		exec.CommandContext(ctx, "sudo", "rm", "-f", sudoersPath),
	}, nil
}
