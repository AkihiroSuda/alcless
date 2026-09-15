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
			expectedArgs: []string{"sudo", "sysadminctl", "-deleteUser", instUser},
		},
		{
			name:         "secure",
			opts:         DeleteOpts{Secure: true},
			expectedArgs: []string{"sudo", "sysadminctl", "-deleteUser", instUser, "-secure"},
		},
		{
			name:         "keep-home",
			opts:         DeleteOpts{KeepHome: true},
			expectedArgs: []string{"sudo", "sysadminctl", "-deleteUser", instUser, "-keepHome"},
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
