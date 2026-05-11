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

// Package sudo provides sudo utilities.
//
// su is wrapped inside sudo, so as to create a launchd session, which is necessary to isolate `open(1)`.
// sudo cannot create a session because `/etc/pam.d/sudo` lacks the config for `pam_launchd.so`.
package sudo

import (
	"context"
	"fmt"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"al.essio.dev/pkg/shellescape"
)

func SudoersPath(instUser string) (string, error) {
	return filepath.Join("/etc/sudoers.d/", instUser), nil
}

// suArgs returns the arguments passed to /usr/bin/su, excluding `-c COMMAND`.
//
// On Linux, `-P` allocates a pseudo-terminal so that the spawned shell has a
// controlling terminal and job control works. macOS `su` (BSD) does not
// support `-P`, and the macOS-style invocation already retains the caller's
// tty, so no workaround is needed there.
func suArgs(instUser string, pty bool) []string {
	if pty && runtime.GOOS == "linux" {
		return []string{"-P", "-", instUser}
	}
	return []string{"-", instUser}
}

func Sudoers(instUser string) (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", err
	}
	// Allow both the plain and the `-P` (Linux pty) forms. Both invocations
	// may be issued by alclessctl depending on whether an interactive
	// pseudo-terminal is needed.
	patterns := []string{
		strings.Join(append([]string{"/usr/bin/su"}, suArgs(instUser, false)...), " ") + " -c *",
	}
	if runtime.GOOS == "linux" {
		patterns = append(patterns,
			strings.Join(append([]string{"/usr/bin/su"}, suArgs(instUser, true)...), " ")+" -c *",
		)
	}
	return fmt.Sprintf("%s ALL=(root) NOPASSWD: %s", currentUser.Username, strings.Join(patterns, ", ")), nil
}

type cmdOpts struct {
	pty bool
}

// CmdOpt is an option for Cmd.
type CmdOpt func(*cmdOpts)

// WithPTY requests that the command be wrapped in a pseudo-terminal,
// so that job control works in interactive shells. Only honored on Linux.
func WithPTY() CmdOpt {
	return func(o *cmdOpts) {
		o.pty = true
	}
}

func Cmd(ctx context.Context, instUser, wd, cmdExe string, cmdArgs []string, opts ...CmdOpt) *exec.Cmd {
	var o cmdOpts
	for _, f := range opts {
		f(&o)
	}
	quotedArgs := make([]string, len(cmdArgs))
	for i, f := range cmdArgs {
		quotedArgs[i] = shellescape.Quote(f)
	}
	execPart := fmt.Sprintf("exec %s %s",
		shellescape.Quote(cmdExe),
		strings.Join(quotedArgs, " "))
	snippet := execPart
	if wd != "" {
		snippet = fmt.Sprintf("cd %s ; %s", shellescape.Quote(wd), execPart)
	}
	args := append([]string{"-n", "/usr/bin/su"}, suArgs(instUser, o.pty)...)
	args = append(args, "-c", snippet)
	cmd := exec.CommandContext(ctx, "sudo", args...)
	return cmd
}
