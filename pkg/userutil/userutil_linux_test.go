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
	"slices"
	"testing"

	"gotest.tools/v3/assert"
)

const (
	testInstUser = "u1002"
	testLabel    = "alcless_exampleuser_default"
	testHome     = "/home/u1002"
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
	cmds, err := AddUserCmds(t.Context(), testInstUser, testLabel, 1002, testHome, true)
	assert.NilError(t, err)

	// -f so that a second instance does not fail on the existing group
	groupadd := findCmd(t, cmds, "groupadd")
	assert.DeepEqual(t, []string{"sudo", "groupadd", "-f", GroupName}, groupadd.Args)

	// -G (supplementary), not -g: the private primary group is kept, so the
	// instances stay isolated from each other.
	useradd := findCmd(t, cmds, "useradd")
	assert.DeepEqual(t, []string{"sudo", "useradd", "-s", "/bin/bash", "--create-home",
		"--home-dir", testHome, "--uid", "1002", "-c", testLabel, "-G", GroupName, testInstUser}, useradd.Args)

	chmod := findCmd(t, cmds, "chmod")
	assert.DeepEqual(t, []string{"sudo", "chmod", "go-rx", testHome}, chmod.Args)
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
			expectedArgs: []string{"sudo", "userdel", "--remove", testInstUser},
		},
		{
			// --secure is not implemented on Linux, and falls back to a normal deletion
			name:         "secure",
			opts:         DeleteOpts{Secure: true},
			expectedArgs: []string{"sudo", "userdel", "--remove", testInstUser},
		},
		{
			name:         "keep-home",
			opts:         DeleteOpts{KeepHome: true},
			expectedArgs: []string{"sudo", "userdel", testInstUser},
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
			assert.Assert(t, len(cmds) > 0)
			assert.DeepEqual(t, tt.expectedArgs, cmds[0].Args)
		})
	}
}

func TestPasswdEntries(t *testing.T) {
	const b = `root:x:0:0:root:/root:/bin/bash
u1002:x:1002:1002:alcless_exampleuser_default:/home/u1002:/bin/bash
truncated:x:1003
`
	got := passwdEntries([]byte(b))
	assert.Equal(t, 2, len(got))
	assert.Equal(t, "u1002", got[1][0])
	assert.Equal(t, "1002", got[1][2])
	assert.Equal(t, "alcless_exampleuser_default", got[1][4])
	assert.Equal(t, "/home/u1002", got[1][5])
}
