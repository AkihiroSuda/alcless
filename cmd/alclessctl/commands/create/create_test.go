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

package create

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestResolveInstName(t *testing.T) {
	tests := []struct {
		args0       string
		flagName    string
		expected    string
		expectedErr string
	}{
		{
			expected: "default",
		},
		{
			args0:    "foo",
			expected: "foo",
		},
		{
			flagName: "foo",
			expected: "foo",
		},
		{
			args0:    "foo",
			flagName: "foo",
			expected: "foo",
		},
		{
			args0:       "foo",
			flagName:    "bar",
			expectedErr: "cannot be specified together",
		},
		{
			args0:       "template://foo",
			expectedErr: "unknown template",
		},
		{
			args0:       "template://foo",
			flagName:    "foo",
			expectedErr: "unknown template",
		},
		{
			args0:    "template://default",
			expected: "default",
		},
		{
			args0:    "template://default",
			flagName: "foo",
			expected: "foo",
		},
		{
			args0:       "foo",
			flagName:    "template://default",
			expectedErr: "must not contain a slash",
		},
	}

	for _, tt := range tests {
		testName := tt.args0 + "-" + tt.flagName
		t.Run(testName, func(t *testing.T) {
			instName, err := resolveInstName(tt.args0, tt.flagName)
			if tt.expectedErr == "" {
				assert.NilError(t, err)
				assert.Equal(t, tt.expected, instName)
			} else {
				assert.ErrorContains(t, err, tt.expectedErr)
			}
		})
	}
}
