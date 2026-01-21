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
			expectError: false,
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
