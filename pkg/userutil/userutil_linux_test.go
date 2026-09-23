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

func TestDeleteUserCmds(t *testing.T) {
	const instUser = "alcless_exampleuser_default"
	tests := []struct {
		name         string
		opts         DeleteOpts
		expectedArgs []string
		expectedErr  string
	}{
		{
			name:         "default",
			expectedArgs: []string{"sudo", "userdel", "--remove", instUser},
		},
		{
			// --secure is not implemented on Linux, and falls back to a normal deletion
			name:         "secure",
			opts:         DeleteOpts{Secure: true},
			expectedArgs: []string{"sudo", "userdel", "--remove", instUser},
		},
		{
			name:         "keep-home",
			opts:         DeleteOpts{KeepHome: true},
			expectedArgs: []string{"sudo", "userdel", instUser},
		},
		{
			name:        "secure-and-keep-home",
			opts:        DeleteOpts{Secure: true, KeepHome: true},
			expectedErr: "conflicts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmds, err := DeleteUserCmds(t.Context(), instUser, tt.opts)
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

func TestGroupSetupCmds(t *testing.T) {
	const instUser = "myuser"

	tests := []struct {
		name      string
		groupName string
		wantArgs  [][]string
	}{
		{
			name:      "with group",
			groupName: "mygroup",
			wantArgs: [][]string{
				{"sudo", "chmod", "go-rx", "/home/myuser"},
				nil, // sudoers sh -c; checked separately below
				{"sudo", "groupadd", "-f", "mygroup"},
				{"sudo", "usermod", "-aG", "mygroup", "myuser"},
			},
		},
		{
			name:      "no group",
			groupName: "",
			wantArgs: [][]string{
				{"sudo", "chmod", "go-rx", "/home/myuser"},
				nil, // sudoers sh -c; checked separately below
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmds, err := GroupSetupCmds(t.Context(), instUser, tt.groupName)
			assert.NilError(t, err)
			assert.Equal(t, len(tt.wantArgs), len(cmds))
			for i, want := range tt.wantArgs {
				if want == nil {
					assert.DeepEqual(t, []string{"sudo", "sh", "-c"}, cmds[i].Args[:3])
					continue
				}
				assert.DeepEqual(t, want, cmds[i].Args)
			}
		})
	}
}

func TestParseGetentGroup(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		wantGID     string
		wantMembers []string
		expectedErr string
	}{
		{
			name:        "no members",
			line:        "mygroup:x:1001:",
			wantGID:     "1001",
			wantMembers: nil,
		},
		{
			name:        "one member",
			line:        "mygroup:x:1001:alice",
			wantGID:     "1001",
			wantMembers: []string{"alice"},
		},
		{
			name:        "multiple members",
			line:        "mygroup:x:1001:alice,bob,carol",
			wantGID:     "1001",
			wantMembers: []string{"alice", "bob", "carol"},
		},
		{
			name:        "malformed: too few fields",
			line:        "mygroup:x:1001",
			expectedErr: "unexpected getent group entry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gid, members, err := parseGetentGroup(tt.line)
			if tt.expectedErr != "" {
				assert.ErrorContains(t, err, tt.expectedErr)
				return
			}
			assert.NilError(t, err)
			assert.Equal(t, tt.wantGID, gid)
			assert.DeepEqual(t, tt.wantMembers, members)
		})
	}
}

func TestUsersWithPrimaryGID(t *testing.T) {
	passwd := []byte(strings.Join([]string{
		"alice:x:1000:1001:Alice:/home/alice:/bin/bash",
		"bob:x:1002:1003:Bob:/home/bob:/bin/bash",
		"carol:x:1004:1001:Carol:/home/carol:/bin/bash",
	}, "\n") + "\n")

	tests := []struct {
		name string
		gid  string
		want []string
	}{
		{name: "one match", gid: "1003", want: []string{"bob"}},
		{name: "multiple matches", gid: "1001", want: []string{"alice", "carol"}},
		{name: "no match", gid: "9999", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := usersWithPrimaryGID(passwd, tt.gid)
			assert.DeepEqual(t, tt.want, got)
		})
	}
}

func TestDedup(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{name: "nil", in: nil, want: nil},
		{name: "no duplicates", in: []string{"a", "b"}, want: []string{"a", "b"}},
		{
			name: "duplicates preserve first-seen order",
			in:   []string{"b", "a", "b", "c", "a"},
			want: []string{"b", "a", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.DeepEqual(t, tt.want, dedup(tt.in))
		})
	}
}

// TestGroupMembershipMerge pins the composition GroupUsers performs: a user
// listed as a supplementary member AND a user whose primary group is the
// target group must appear exactly once in the merged, deduped result.
func TestGroupMembershipMerge(t *testing.T) {
	const groupLine = "mygroup:x:1001:alice,bob"
	passwd := []byte(strings.Join([]string{
		"bob:x:1000:1001:Bob:/home/bob:/bin/bash",       // supplementary member, also primary GID
		"carol:x:1002:1001:Carol:/home/carol:/bin/bash", // primary GID only
	}, "\n") + "\n")

	gid, members, err := parseGetentGroup(groupLine)
	assert.NilError(t, err)
	primary := usersWithPrimaryGID(passwd, gid)
	got := dedup(append(members, primary...))

	assert.DeepEqual(t, []string{"alice", "bob", "carol"}, got)
}
