package userutil

import (
	"os"
	"testing"

	"gotest.tools/v3/assert"
)

func TestGroupName(t *testing.T) {
	// Save original env
	originalGroup := os.Getenv(envGroup)
	defer func() {
		if originalGroup != "" {
			os.Setenv(envGroup, originalGroup)
		} else {
			os.Unsetenv(envGroup)
		}
		// Reset global state
		if originalGroup != "" {
			Mode = "group"
			groupName = originalGroup
		} else {
			Mode = "prefix"
			groupName = ""
		}
	}()

	tests := []struct {
		name     string
		envValue string
		expected string
	}{
		{
			name:     "empty environment variable",
			envValue: "",
			expected: "",
		},
		{
			name:     "set environment variable",
			envValue: "testgroup",
			expected: "testgroup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue == "" {
				os.Unsetenv(envGroup)
			} else {
				os.Setenv(envGroup, tt.envValue)
			}
			
			// Reinitialize
			if groupName = os.Getenv(envGroup); groupName != "" {
				Mode = "group"
			} else {
				Mode = "prefix"
				groupName = ""
			}

			result := GroupName()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUserFromInstance(t *testing.T) {
	// Save original state
	originalMode := Mode
	originalPrefix := Prefix
	originalGroupName := groupName
	defer func() {
		Mode = originalMode
		Prefix = originalPrefix
		groupName = originalGroupName
	}()

	tests := []struct {
		name     string
		mode     string
		prefix   string
		instName string
		expected string
	}{
		{
			name:     "prefix mode",
			mode:     "prefix",
			prefix:   "alcless_user_",
			instName: "default",
			expected: "alcless_user_default",
		},
		{
			name:     "group mode",
			mode:     "group",
			instName: "testuser",
			expected: "testuser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Mode = tt.mode
			Prefix = tt.prefix
			
			result := UserFromInstance(tt.instName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInstanceFromUser(t *testing.T) {
	// Save original state
	originalMode := Mode
	originalPrefix := Prefix
	defer func() {
		Mode = originalMode
		Prefix = originalPrefix
	}()

	tests := []struct {
		name     string
		mode     string
		prefix   string
		username string
		expected string
	}{
		{
			name:     "prefix mode",
			mode:     "prefix",
			prefix:   "alcless_user_",
			username: "alcless_user_default",
			expected: "default",
		},
		{
			name:     "group mode",
			mode:     "group",
			username: "testuser",
			expected: "testuser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Mode = tt.mode
			Prefix = tt.prefix
			
			result := InstanceFromUser(tt.username)
			assert.Equal(t, tt.expected, result)
		})
	}
}
