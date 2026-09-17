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
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

// Prefix is the prefix of the instance labels.
//
// The label is stored in the RealName (macOS) / GECOS (Linux) field of the
// instance user. It is NOT the user name: the user name is [UserName].
//
// The label is purely cosmetic (it is what the macOS login window shows) and is
// never trusted, as the instance user may be able to rewrite its own GECOS
// field via chfn(1), depending on CHFN_RESTRICT in /etc/login.defs.
// The instance name is resolved via the store, not via the label.
var Prefix = "alcless_" + me() + "_"

// GroupName is the group that every instance user belongs to (supplementary).
//
// Only root can create a group or edit its membership, so this is a trustable
// marker for "this account was created by Alcoholless".
const GroupName = "alcless"

func me() string {
	u, err := user.Current()
	if err != nil {
		panic(err)
	}
	if u.Username == "" {
		panic("no username")
	}
	return u.Username
}

// Entry is an entry of the user database.
type Entry struct {
	// Name is the user name, e.g., "u502".
	Name string
	// Label is the RealName (macOS) / GECOS (Linux) field, e.g., "alcless_exampleuser_default".
	// Untrusted. See [Prefix].
	Label string
}

// LabelFromInstance returns the label for the instance, e.g., "alcless_exampleuser_default".
func LabelFromInstance(instName string) string {
	return Prefix + instName
}

// InstanceFromLabel returns the instance name for the label.
func InstanceFromLabel(label string) string {
	return strings.TrimPrefix(label, Prefix)
}

// UserName returns the user name for the UID, e.g., "u502".
//
// The name is kept short so that the home directory, and in turn the Homebrew
// prefix, fits in the length limit for pouring bottles. See [brew.MaxPrefixLen].
func UserName(uid int) string {
	return "u" + strconv.Itoa(uid)
}

// HomeDir returns the home directory for the UID, e.g., "/Users/u502".
func HomeDir(uid int) string {
	return filepath.Join(homeDirBase, UserName(uid))
}

// Exists reports whether a user with exactly this user name exists.
//
// The name has to match exactly: on macOS, getpwnam(3) also resolves a user by
// its RealName, so looking up a label such as "alcless_exampleuser_default"
// succeeds and returns the account that merely carries the label (e.g. "u502").
// Treating that as a hit would misidentify a current instance as an old-format
// one, and yield a user name that the sudoers rule does not permit.
func Exists(name string) (bool, error) {
	u, err := user.Lookup(name)
	if err != nil {
		var uee user.UnknownUserError
		if errors.As(err, &uee) {
			return false, nil
		}
		return false, err
	}
	return u.Username == name, nil
}

// Allocate returns the UID, the user name, and the home directory for a new instance user.
func Allocate(ctx context.Context) (int, string, string, error) {
	used, err := UIDs(ctx)
	if err != nil {
		return 0, "", "", err
	}
	uid, err := firstFree(used, taken, minUID, maxUID)
	if err != nil {
		return 0, "", "", err
	}
	return uid, UserName(uid), HomeDir(uid), nil
}

// maxUID is a sanity limit. The UIDs actually in use are expected to be much
// smaller, as [firstFree] returns the lowest free one.
const maxUID = 60000

// taken reports whether the UID cannot be used, even though the UID itself is unused.
//
// The home directory is checked because `alclessctl delete --keep-home` frees
// the UID but leaves the home directory behind; reusing the UID would make an
// unrelated instance inherit the files of the deleted one.
func taken(uid int) bool {
	if exists, err := Exists(UserName(uid)); err != nil || exists {
		return true
	}
	if _, err := os.Lstat(HomeDir(uid)); err == nil {
		return true
	}
	return false
}

// firstFree returns the lowest UID in [min, max] that is neither in used nor rejected by taken.
func firstFree(used map[int]struct{}, taken func(int) bool, min, max int) (int, error) {
	for uid := min; uid <= max; uid++ {
		if _, ok := used[uid]; ok {
			continue
		}
		if taken != nil && taken(uid) {
			continue
		}
		return uid, nil
	}
	return 0, fmt.Errorf("no free UID in [%d, %d]", min, max)
}

// groupExists reports whether the [GroupName] group exists.
func groupExists() (bool, error) {
	if _, err := user.LookupGroup(GroupName); err != nil {
		var uge user.UnknownGroupError
		if errors.As(err, &uge) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// DeleteGroupIfEmptyCmds returns the commands to remove the [GroupName] group,
// or nil when the group does not exist or still has a member.
//
// The group is shared by every instance of every host user, so it can only be
// removed once the last instance on the machine is gone. Call this after the
// commands of [DeleteUserCmds] have run, so that the membership of the deleted
// user is already dropped.
func DeleteGroupIfEmptyCmds(ctx context.Context) ([]*exec.Cmd, error) {
	exists, err := groupExists()
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	members, err := GroupMembers(ctx)
	if err != nil {
		return nil, err
	}
	if len(members) > 0 {
		return nil, nil
	}
	return deleteGroupCmds(ctx), nil
}

type Attribute string

const (
	AttributeUserShell = Attribute("UserShell")
)

// DeleteOpts is the options for [DeleteUserCmds].
type DeleteOpts struct {
	// Secure securely erases the home directory (slow).
	// Not implemented on Linux.
	Secure bool
	// KeepHome keeps the home directory of the user.
	// Conflicts with Secure.
	KeepHome bool
}
