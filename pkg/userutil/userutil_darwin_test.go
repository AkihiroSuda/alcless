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
	"os/exec"
	"os/user"
	"slices"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

const (
	testInstUser = "u502"
	testLabel    = "alcless_exampleuser_default"
	testHome     = "/Users/u502"
)

// findCmd returns the first command whose arguments contain want as a contiguous run.
func findCmd(t *testing.T, cmds []*exec.Cmd, want ...string) *exec.Cmd {
	t.Helper()
	for _, c := range cmds {
		for i := 0; i+len(want) <= len(c.Args); i++ {
			if slices.Equal(c.Args[i:i+len(want)], want) {
				return c
			}
		}
	}
	t.Fatalf("no %v command in %v", want, cmds)
	return nil
}

func TestAddUserCmds(t *testing.T) {
	t.Run("tty", func(t *testing.T) {
		cmds, err := AddUserCmds(t.Context(), testInstUser, testLabel, 502, testHome, true)
		assert.NilError(t, err)

		// "-" makes sysadminctl prompt for the password interactively
		sysadminctl := findCmd(t, cmds, "sysadminctl")
		assert.DeepEqual(t, []string{"sudo", "sysadminctl", "-addUser", testInstUser,
			"-UID", "502", "-fullName", testLabel, "-home", testHome, "-password", "-"}, sysadminctl.Args)

		// The user has to join the group, which is the trustable marker for
		// "created by Alcoholless".
		dseditgroup := findCmd(t, cmds, "dseditgroup", "-o", "edit")
		assert.DeepEqual(t, []string{"sudo", "dseditgroup", "-o", "edit", "-a", testInstUser,
			"-t", "user", GroupName}, dseditgroup.Args)

		// sysadminctl only assigns the home directory when -home is given, so it has
		// to be created explicitly, before the chmod that locks it down.
		createhomedir := findCmd(t, cmds, "createhomedir")
		assert.DeepEqual(t, []string{"sudo", "createhomedir", "-c", "-u", testInstUser}, createhomedir.Args)

		chmod := findCmd(t, cmds, "chmod")
		assert.DeepEqual(t, []string{"sudo", "chmod", "go-rx", testHome}, chmod.Args)
		assert.Assert(t, slices.Index(cmds, createhomedir) < slices.Index(cmds, chmod))
	})

	t.Run("no-tty", func(t *testing.T) {
		// Without a tty there is nothing to prompt, so the generated password
		// has to actually reach sysadminctl
		cmds, err := AddUserCmds(t.Context(), testInstUser, testLabel, 502, testHome, false)
		assert.NilError(t, err)

		args := findCmd(t, cmds, "sysadminctl").Args
		assert.DeepEqual(t, []string{"sudo", "sysadminctl", "-addUser", testInstUser,
			"-UID", "502", "-fullName", testLabel, "-home", testHome, "-password"}, args[:len(args)-1])
		pw := args[len(args)-1]
		assert.Assert(t, pw != "-", "expected a generated password, got the interactive prompt sentinel")
		assert.Equal(t, 64, len(pw))
	})
}

func TestDeleteGroupCmds(t *testing.T) {
	cmds := deleteGroupCmds(t.Context())
	assert.DeepEqual(t, []string{"sudo", "dseditgroup", "-o", "delete", GroupName},
		findCmd(t, cmds, "dseditgroup").Args)
}

func TestDeleteUserCmds(t *testing.T) {
	tests := []struct {
		name         string
		opts         DeleteOpts
		expectedArgs []string
		expectedErr  string
	}{
		{
			name:         "default",
			expectedArgs: []string{"sudo", "sysadminctl", "-deleteUser", testInstUser},
		},
		{
			name:         "secure",
			opts:         DeleteOpts{Secure: true},
			expectedArgs: []string{"sudo", "sysadminctl", "-deleteUser", testInstUser, "-secure"},
		},
		{
			name:         "keep-home",
			opts:         DeleteOpts{KeepHome: true},
			expectedArgs: []string{"sudo", "sysadminctl", "-deleteUser", testInstUser, "-keepHome"},
		},
		{
			name:        "secure-and-keep-home",
			opts:        DeleteOpts{Secure: true, KeepHome: true},
			expectedErr: "conflicts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmds, err := DeleteUserCmds(t.Context(), testInstUser, tt.opts)
			if tt.expectedErr != "" {
				assert.ErrorContains(t, err, tt.expectedErr)
				return
			}
			assert.NilError(t, err)
			assert.DeepEqual(t, tt.expectedArgs, findCmd(t, cmds, "sysadminctl").Args)
			// The group membership is dropped while the user record still exists
			sh := findCmd(t, cmds, "sh", "-c")
			assert.Assert(t, strings.Contains(sh.Args[3], "dseditgroup -o edit -d"))
			assert.Assert(t, slices.Index(cmds, sh) < slices.Index(cmds, findCmd(t, cmds, "sysadminctl")))
		})
	}
}

func TestDsclOutput(t *testing.T) {
	const b = `_www                     World Wide Web Server
u502                     alcless_exampleuser_default
nobody                   Unprivileged User
emptyvalue
`
	got := dsclOutput([]byte(b))
	assert.Equal(t, "alcless_exampleuser_default", got["u502"])
	// A value containing spaces is preserved
	assert.Equal(t, "World Wide Web Server", got["_www"])
	assert.Equal(t, "", got["emptyvalue"])
}

// TestExistsRequiresExactUserName guards against macOS getpwnam(3) resolving a
// user by its RealName: looking up the label of an instance must not report the
// account that carries that label as existing under that name.
func TestExistsRequiresExactUserName(t *testing.T) {
	// The current user always exists under its own name
	u, err := user.Current()
	assert.NilError(t, err)
	got, err := Exists(u.Username)
	assert.NilError(t, err)
	assert.Assert(t, got)

	if u.Name != "" && u.Name != u.Username {
		// The RealName resolves via getpwnam on macOS, but is not a user name
		got, err = Exists(u.Name)
		assert.NilError(t, err)
		assert.Assert(t, !got, "Exists(%q) must not match the account whose RealName it is", u.Name)
	}

	got, err = Exists("alcless-no-such-user-xyz")
	assert.NilError(t, err)
	assert.Assert(t, !got)
}
