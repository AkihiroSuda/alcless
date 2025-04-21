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
