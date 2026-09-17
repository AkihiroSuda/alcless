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

package brew

import (
	"runtime"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/AkihiroSuda/alcless/pkg/userutil"
)

func TestPrefix(t *testing.T) {
	assert.Equal(t, "/Users/u502/h", Prefix("/Users/u502"))
	assert.Equal(t, "/Users/u502/homebrew", LegacyPrefix("/Users/u502"))
}

// TestPrefixFitsMaxPrefixLen verifies the whole point of the short user names:
// the Homebrew prefix must not be longer than the prefix the official bottles
// were built for, otherwise every formula is built from source.
func TestPrefixFitsMaxPrefixLen(t *testing.T) {
	// The highest UID that still yields a 3-digit user name.
	// Allocation returns the lowest free UID, so this is a realistic worst case.
	prefix := Prefix(userutil.HomeDir(999))
	max := MaxPrefixLen()
	t.Logf("prefix=%q len=%d max=%d", prefix, len(prefix), max)

	if runtime.GOOS == "darwin" && runtime.GOARCH != "arm64" {
		// The bottles for macOS on Intel are built for /usr/local (10 chars),
		// which cannot be matched from under /Users (7 chars) at all.
		// alclessctl warns about this at `create` time.
		assert.Assert(t, len(prefix) > max, "expected the known macOS Intel limitation")
		return
	}
	assert.Assert(t, len(prefix) <= max, "prefix %q (%d chars) exceeds the limit of %d",
		prefix, len(prefix), max)
}

func TestMaxPrefixLen(t *testing.T) {
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			assert.Equal(t, len("/opt/homebrew"), MaxPrefixLen())
		} else {
			assert.Equal(t, len("/usr/local"), MaxPrefixLen())
		}
	case "linux":
		assert.Equal(t, len("/home/linuxbrew/.linuxbrew"), MaxPrefixLen())
	}
}
