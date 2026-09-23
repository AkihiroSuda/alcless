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

package store

import (
	"context"
	"os"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/AkihiroSuda/alcless/pkg/userutil"
)

func TestInstancesFromGroup(t *testing.T) {
	// Save original state
	originalMode := userutil.Mode
	originalGroupName := os.Getenv("ALCLESS_GROUP")
	defer func() {
		userutil.Mode = originalMode
		if originalGroupName != "" {
			os.Setenv("ALCLESS_GROUP", originalGroupName)
		} else {
			os.Unsetenv("ALCLESS_GROUP")
		}
	}()

	t.Run("missing ALCLESS_GROUP environment variable", func(t *testing.T) {
		os.Unsetenv("ALCLESS_GROUP")
		userutil.Mode = "group"

		_, err := instancesFromGroup(context.Background())
		assert.ErrorContains(t, err, "ALCLESS_GROUP environment variable is not set")
	})
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "valid name",
			input:       "testuser",
			expectError: false,
		},
		{
			name:        "reserved prefix",
			input:       "alcless_testuser",
			expectError: true,
		},
		{
			name:        "empty name",
			input:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.input)
			if tt.expectError {
				assert.Assert(t, err != nil, "expected error but got nil")
			} else {
				assert.NilError(t, err)
			}
		})
	}
}
