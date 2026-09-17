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
	"testing"

	"gotest.tools/v3/assert"
)

func TestFirstFree(t *testing.T) {
	used := func(uids ...int) map[int]struct{} {
		res := make(map[int]struct{})
		for _, uid := range uids {
			res[uid] = struct{}{}
		}
		return res
	}
	tests := []struct {
		name        string
		used        map[int]struct{}
		taken       func(int) bool
		min, max    int
		expected    int
		expectedErr string
	}{
		{
			name:     "empty",
			used:     used(),
			min:      501,
			max:      600,
			expected: 501,
		},
		{
			name:     "lowest free",
			used:     used(501, 502, 504),
			min:      501,
			max:      600,
			expected: 503,
		},
		{
			name: "skips taken",
			used: used(501),
			// e.g. the home directory left behind by `delete --keep-home`
			taken:    func(uid int) bool { return uid == 502 || uid == 503 },
			min:      501,
			max:      600,
			expected: 504,
		},
		{
			name:        "exhausted",
			used:        used(501, 502),
			min:         501,
			max:         502,
			expectedErr: "no free UID",
		},
		{
			name:        "exhausted by taken",
			used:        used(),
			taken:       func(int) bool { return true },
			min:         501,
			max:         502,
			expectedErr: "no free UID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uid, err := firstFree(tt.used, tt.taken, tt.min, tt.max)
			if tt.expectedErr != "" {
				assert.ErrorContains(t, err, tt.expectedErr)
				return
			}
			assert.NilError(t, err)
			assert.Equal(t, tt.expected, uid)
		})
	}
}

func TestUserNameAndHomeDir(t *testing.T) {
	assert.Equal(t, "u502", UserName(502))
	assert.Equal(t, "u1000", UserName(1000))
	// The home directory has to stay short enough for the Homebrew prefix
	// (home + "/h") to fit in brew.MaxPrefixLen(). See pkg/brew.
	assert.Equal(t, homeDirBase+"/u502", HomeDir(502))
}

func TestLabel(t *testing.T) {
	label := LabelFromInstance("default")
	assert.Assert(t, label != "default")
	assert.Equal(t, "default", InstanceFromLabel(label))
	// A label of another user must not be resolved as an instance name
	assert.Equal(t, "alcless_otheruser_default", InstanceFromLabel("alcless_otheruser_default"))
}
