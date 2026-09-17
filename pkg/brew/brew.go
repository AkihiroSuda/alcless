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

package brew

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/AkihiroSuda/alcless/pkg/sudo"
)

const (
	// Dir is the Homebrew prefix, relative to the home directory of the instance user.
	//
	// Deliberately short (not "homebrew"), so that the prefix fits in [MaxPrefixLen].
	Dir = "h"
	// LegacyDir is the Homebrew prefix that was used by the older versions of Alcoholless.
	LegacyDir = "homebrew"
)

// Prefix returns the Homebrew prefix (HOMEBREW_PREFIX), e.g., "/Users/u502/h".
func Prefix(homeDir string) string {
	return filepath.Join(homeDir, Dir)
}

// LegacyPrefix returns the Homebrew prefix used by the older versions of Alcoholless.
func LegacyPrefix(homeDir string) string {
	return filepath.Join(homeDir, LegacyDir)
}

// MaxPrefixLen returns the maximum length of HOMEBREW_PREFIX that still allows
// pouring the official bottles. A longer prefix makes Homebrew build every
// formula from source.
//
// A bottle can only be relocated to a prefix that is not longer than the prefix
// it was built for, as the prefix strings are patched in place:
// https://github.com/Homebrew/brew/blob/HEAD/Library/Homebrew/bottle_specification.rb
//
// Returns 0 when the platform is unknown.
func MaxPrefixLen() int {
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return len("/opt/homebrew")
		}
		return len("/usr/local")
	case "linux":
		return len("/home/linuxbrew/.linuxbrew")
	}
	return 0
}

func InstalledCmd(ctx context.Context, instUser, prefix string) *exec.Cmd {
	return sudo.Cmd(ctx, instUser, "", filepath.Join(prefix, "bin/brew"), []string{"--version"})
}

// Installed returns the Homebrew prefix that is already installed for the instance user.
//
// [LegacyPrefix] is probed too, so that the instances created by the older
// versions of Alcoholless keep working.
func Installed(ctx context.Context, instUser, homeDir string) (string, error) {
	var errs []error
	for _, prefix := range []string{Prefix(homeDir), LegacyPrefix(homeDir)} {
		var stderr bytes.Buffer
		cmd := InstalledCmd(ctx, instUser, prefix)
		cmd.Stderr = &stderr
		slog.DebugContext(ctx, "Running command", "cmd", cmd.Args)
		b, err := cmd.Output()
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to run %v: %w (stderr=%q)", cmd.Args, err, stderr.String()))
			continue
		}
		slog.DebugContext(ctx, "Homebrew has been already installed", "user", instUser, "prefix", prefix, "version", string(b))
		return prefix, nil
	}
	return "", fmt.Errorf("failed to detect Homebrew for the user %q: %w", instUser, errors.Join(errs...))
}

func InstallCmds(ctx context.Context, instUser string) []*exec.Cmd {
	systemHomebrewPrefix := "/opt/homebrew"
	if runtime.GOOS == "linux" {
		systemHomebrewPrefix = "/home/linuxbrew/.linuxbrew"
	}
	cmds := []*exec.Cmd{
		// Remove system-wide Homebrew (/opt/homebrew/bin) from the PATH
		// Needed since Homebrew 4.5.9 (July 8, 2025)
		// https://github.com/AkihiroSuda/alcless/issues/23
		sudo.Cmd(ctx, instUser, "", "sh", []string{"-c", `echo 'PATH="$(echo "$PATH" | sed -e s@` + systemHomebrewPrefix + `/bin:@@g)"; export PATH' | tee -a "${HOME}/.bash_profile" | tee -a "${HOME}/.bashrc" | tee -a "${HOME}/.zprofile" >> "${HOME}/.zshenv"`}),

		sudo.Cmd(ctx, instUser, "", "git", []string{"clone", "https://github.com/Homebrew/brew", Dir}),
		sudo.Cmd(ctx, instUser, "", "sh", []string{"-c", `echo 'eval "$("${HOME}/` + Dir + `/bin/brew" shellenv)"' | tee -a "${HOME}/.bash_profile" >> "${HOME}/.zshenv"`}),
	}
	return cmds
}

func Supported() bool {
	switch runtime.GOOS {
	case "darwin":
		return true
	case "linux":
		switch runtime.GOARCH {
		case "amd64", "arm64":
			return true
		default:
			return false
		}
	default:
		return false
	}
}
