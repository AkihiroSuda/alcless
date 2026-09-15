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
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/AkihiroSuda/alcless/pkg/userutil"
)

// self returns an instance that points at the user running the test, so that
// the user database lookups in load() succeed.
func self(t *testing.T) *Instance {
	t.Helper()
	u, err := user.Current()
	assert.NilError(t, err)
	uid, err := strconv.Atoi(u.Uid)
	assert.NilError(t, err)
	return &Instance{
		Version: Version,
		Name:    "default",
		User:    u.Username,
		UID:     uid,
	}
}

func members(names ...string) map[string]struct{} {
	res := make(map[string]struct{})
	for _, name := range names {
		res[name] = struct{}{}
	}
	return res
}

func TestDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
	dir, err := Dir()
	assert.NilError(t, err)
	assert.Equal(t, "/tmp/xdg/alcless", dir)

	// Not os.UserConfigDir(), which ignores XDG_CONFIG_HOME on macOS
	t.Setenv("XDG_CONFIG_HOME", "")
	dir, err = Dir()
	assert.NilError(t, err)
	home, err := os.UserHomeDir()
	assert.NilError(t, err)
	assert.Equal(t, filepath.Join(home, ".config", "alcless"), dir)

	t.Setenv("XDG_CONFIG_HOME", "relative/path")
	_, err = Dir()
	assert.ErrorContains(t, err, "absolute")
}

func TestSaveAndLoad(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	inst := self(t)
	assert.NilError(t, Save(inst))

	path, err := instancePath(inst.Name)
	assert.NilError(t, err)
	st, err := os.Stat(path)
	assert.NilError(t, err)
	assert.Equal(t, os.FileMode(0o600), st.Mode().Perm())

	got, err := load(path, members(inst.User))
	assert.NilError(t, err)
	assert.Equal(t, inst.Name, got.Name)
	assert.Equal(t, inst.User, got.User)
	assert.Equal(t, inst.UID, got.UID)
	assert.Assert(t, got.Home != "")
	assert.Assert(t, !got.Legacy)

	assert.NilError(t, Remove(inst.Name))
	// Removing a non-existent instance is not an error
	assert.NilError(t, Remove(inst.Name))
}

// TestLoadRejectsUntrustworthy covers the checks that keep a tampered or stale
// entry from redirecting a command onto the wrong account.
func TestLoadRejectsUntrustworthy(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	inst := self(t)
	path, err := instancePath(inst.Name)
	assert.NilError(t, err)

	t.Run("world-writable", func(t *testing.T) {
		assert.NilError(t, Save(inst))
		assert.NilError(t, os.Chmod(path, 0o666))
		_, err := load(path, members(inst.User))
		assert.ErrorContains(t, err, "writable only by the owner")
	})

	t.Run("not-a-group-member", func(t *testing.T) {
		assert.NilError(t, Save(inst))
		_, err := load(path, members())
		assert.ErrorContains(t, err, `not a member of the "`+userutil.GroupName+`" group`)
	})

	t.Run("uid-mismatch", func(t *testing.T) {
		// e.g. the recorded user name was deleted and the UID recycled
		mismatched := *inst
		mismatched.UID = inst.UID + 12345
		assert.NilError(t, Save(&mismatched))
		_, err := load(path, members(inst.User))
		assert.ErrorContains(t, err, "expected")
	})

	t.Run("unknown-user", func(t *testing.T) {
		unknown := *inst
		unknown.User = "alcless-no-such-user"
		assert.NilError(t, Save(&unknown))
		_, err := load(path, members(unknown.User))
		assert.ErrorContains(t, err, "failed to look up the user")
	})

	t.Run("wrong-version", func(t *testing.T) {
		wrong := *inst
		wrong.Version = Version + 1
		assert.NilError(t, Save(&wrong))
		_, err := load(path, members(inst.User))
		assert.ErrorContains(t, err, "unexpected version")
	})

	t.Run("name-mismatch", func(t *testing.T) {
		// The file name is authoritative, so that a renamed file cannot shadow
		// another instance.
		assert.NilError(t, Save(inst))
		b, err := os.ReadFile(path)
		assert.NilError(t, err)
		other := filepath.Join(filepath.Dir(path), "other.json")
		assert.NilError(t, os.WriteFile(other, b, 0o600))
		_, err = load(other, members(inst.User))
		assert.ErrorContains(t, err, "unexpected instance name")
	})
}

func TestValidateName(t *testing.T) {
	assert.NilError(t, ValidateName("default"))
	assert.ErrorContains(t, ValidateName("alcless_exampleuser_default"), "must not start with")
	assert.Assert(t, ValidateName("../escape") != nil)
	assert.Assert(t, ValidateName("") != nil)
}

func TestInstancePathRejectsTraversal(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	_, err := instancePath("../../etc/passwd")
	assert.Assert(t, err != nil)
}
