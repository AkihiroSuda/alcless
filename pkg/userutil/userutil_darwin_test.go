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
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestParseGroupMembership(t *testing.T) {
	tests := []struct {
		name string
		out  string
		want []string
	}{
		{
			name: "no such key",
			out:  "No such key: GroupMembership\n",
			want: nil,
		},
		{
			name: "single line, one member",
			out:  "GroupMembership: root\n",
			want: []string{"root"},
		},
		{
			name: "single line, multiple members",
			out:  "GroupMembership: root harry\n",
			want: []string{"root", "harry"},
		},
		{
			// dscl(1) wraps a value list to one entry per indented line when
			// a value contains an embedded space.
			name: "wrapped, one entry per line",
			out:  "GroupMembership:\n alice smith\n bob\n",
			want: []string{"alice smith", "bob"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.DeepEqual(t, tt.want, parseGroupMembership([]byte(tt.out)))
		})
	}
}

func TestDarwinGroupSetupCmds(t *testing.T) {
	const instUser = "myuser"

	tests := []struct {
		name               string
		groupName          string
		groupAlreadyExists bool
		wantArgs           [][]string
	}{
		{
			name:               "group does not exist yet",
			groupName:          "mygroup",
			groupAlreadyExists: false,
			wantArgs: [][]string{
				{"sudo", "chmod", "go-rx", "/Users/myuser"},
				nil, // sudoers sh -c; checked separately below
				{"sudo", "dscl", ".", "-create", "/Groups/mygroup"},
				{"sudo", "dscl", ".", "-merge", "/Groups/mygroup", "GroupMembership", "myuser"},
			},
		},
		{
			name:               "group already exists",
			groupName:          "mygroup",
			groupAlreadyExists: true,
			wantArgs: [][]string{
				{"sudo", "chmod", "go-rx", "/Users/myuser"},
				nil, // sudoers sh -c; checked separately below
				{"sudo", "dscl", ".", "-merge", "/Groups/mygroup", "GroupMembership", "myuser"},
			},
		},
		{
			name:               "no group",
			groupName:          "",
			groupAlreadyExists: false,
			wantArgs: [][]string{
				{"sudo", "chmod", "go-rx", "/Users/myuser"},
				nil, // sudoers sh -c; checked separately below
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmds, err := groupSetupCmds(t.Context(), instUser, tt.groupName, tt.groupAlreadyExists)
			assert.NilError(t, err)
			assert.Equal(t, len(tt.wantArgs), len(cmds))
			for i, want := range tt.wantArgs {
				if want == nil {
					assert.DeepEqual(t, []string{"sudo", "sh", "-c"}, cmds[i].Args[:3])
					continue
				}
				assert.DeepEqual(t, want, cmds[i].Args)
			}

			// The group name must never be interpolated into a shell string:
			// every command touching it takes it as a direct argument.
			for _, c := range cmds {
				if tt.groupName == "" {
					continue
				}
				if len(c.Args) >= 3 && c.Args[1] == "sh" && c.Args[2] == "-c" {
					assert.Assert(t, !strings.Contains(c.Args[3], tt.groupName))
				}
			}
		})
	}
}
