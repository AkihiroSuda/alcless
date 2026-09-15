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
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/user"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"github.com/containerd/containerd/v2/pkg/identifiers"

	"github.com/AkihiroSuda/alcless/pkg/userutil"
)

// Version is the format version of [Instance].
const Version = 1

type Instance struct {
	// Version is the format version. See [Version].
	Version int `json:"version"`
	// Name is the instance name, e.g., "default".
	Name string `json:"name"`
	// User is the user name of the instance user, e.g., "u502".
	User string `json:"user"`
	// UID is the UID of the instance user, e.g., 502.
	// Recorded so that a stale entry cannot be redirected onto a recycled UID.
	UID int `json:"uid"`

	// Home is the home directory of the instance user.
	// Resolved from the user database; not persisted.
	Home string `json:"home,omitempty"`
	// Legacy indicates that the instance was created by an older version of
	// Alcoholless, with a long home directory. Not persisted.
	Legacy bool `json:"legacy,omitempty"`
}

// Dir returns the directory that holds the instance metadata,
// e.g., "/Users/exampleuser/.config/alcless".
//
// Deliberately not [os.UserConfigDir]: on macOS that returns
// "$HOME/Library/Application Support" and ignores $XDG_CONFIG_HOME.
func Dir() (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configHome = filepath.Join(homeDir, ".config")
	}
	if !filepath.IsAbs(configHome) {
		return "", fmt.Errorf("expected an absolute path, got %q", configHome)
	}
	return filepath.Join(configHome, "alcless"), nil
}

func instancesDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "instances"), nil
}

func instancePath(instName string) (string, error) {
	if err := ValidateName(instName); err != nil {
		return "", err
	}
	dir, err := instancesDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, instName+".json"), nil
}

// Save writes the instance metadata.
//
// The file is written atomically, via a temporary file, so that an existing
// file cannot keep a too permissive mode (os.WriteFile does not reset the mode
// of an existing file).
func Save(inst *Instance) error {
	path, err := instancePath(inst.Name)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err = os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	// Home and Legacy are resolved from the user database on load, so writing
	// them would only risk going stale.
	persisted := *inst
	persisted.Home = ""
	persisted.Legacy = false
	b, err := json.MarshalIndent(persisted, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+inst.Name+".json.*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename
	if err = tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err = tmp.Write(append(b, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// Remove removes the instance metadata. Removing a non-existent instance is not an error.
func Remove(instName string) error {
	path, err := instancePath(instName)
	if err != nil {
		return err
	}
	if err = os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// verifyOwnerAndMode ensures that the metadata is owned by the current user and
// is not writable by anybody else, so that it cannot be tampered with by the
// instance user.
func verifyOwnerAndMode(st os.FileInfo) error {
	if perm := st.Mode().Perm(); perm&0o022 != 0 {
		return fmt.Errorf("expected the file to be writable only by the owner, got mode %#o", perm)
	}
	sys, ok := st.Sys().(*syscall.Stat_t)
	if !ok {
		return errors.New("failed to detect the owner of the file")
	}
	if int(sys.Uid) != os.Getuid() {
		return fmt.Errorf("expected the file to be owned by UID %d, got %d", os.Getuid(), sys.Uid)
	}
	return nil
}

// load reads and validates the instance metadata.
//
// The metadata is only trusted when it is owned by the current user and is not
// writable by anybody else, and when it still corresponds to a live instance
// user: the recorded UID has to match, and the user has to be a member of the
// [userutil.GroupName] group, which only root can edit.
func load(path string, members map[string]struct{}) (*Instance, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if err = verifyOwnerAndMode(st); err != nil {
		return nil, fmt.Errorf("refusing to use %q: %w", path, err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var inst Instance
	if err = json.Unmarshal(b, &inst); err != nil {
		return nil, fmt.Errorf("failed to parse %q: %w", path, err)
	}
	if inst.Version != Version {
		return nil, fmt.Errorf("unexpected version %d in %q (expected %d)", inst.Version, path, Version)
	}
	if expected := strings.TrimSuffix(filepath.Base(path), ".json"); inst.Name != expected {
		return nil, fmt.Errorf("unexpected instance name %q in %q", inst.Name, path)
	}
	u, err := user.Lookup(inst.User)
	if err != nil {
		return nil, fmt.Errorf("failed to look up the user %q of the instance %q: %w", inst.User, inst.Name, err)
	}
	// An exact match is required: macOS getpwnam(3) also resolves by RealName.
	if u.Username != inst.User {
		return nil, fmt.Errorf("the user %q of the instance %q resolved to the user %q",
			inst.User, inst.Name, u.Username)
	}
	if u.Uid != strconv.Itoa(inst.UID) {
		return nil, fmt.Errorf("the user %q of the instance %q has UID %s, expected %d",
			inst.User, inst.Name, u.Uid, inst.UID)
	}
	if _, ok := members[inst.User]; !ok {
		return nil, fmt.Errorf("the user %q of the instance %q is not a member of the %q group",
			inst.User, inst.Name, userutil.GroupName)
	}
	inst.Home = u.HomeDir
	return &inst, nil
}

func Instances(ctx context.Context) ([]Instance, error) {
	members, err := userutil.GroupMembers(ctx)
	if err != nil {
		return nil, err
	}
	dir, err := instancesDir()
	if err != nil {
		return nil, err
	}
	var res []Instance
	seenUsers := make(map[string]struct{})
	ents, err := os.ReadDir(dir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	for _, ent := range ents {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".json") {
			continue
		}
		inst, err := load(filepath.Join(dir, ent.Name()), members)
		if err != nil {
			slog.WarnContext(ctx, "Ignoring an invalid instance metadata file", "error", err)
			continue
		}
		res = append(res, *inst)
		seenUsers[inst.User] = struct{}{}
	}

	// Instances created by the older versions of Alcoholless carry the label in
	// the user name itself, and have no metadata file.
	users, err := userutil.Users(ctx)
	if err != nil {
		return nil, err
	}
	for _, u := range users {
		if !strings.HasPrefix(u.Name, userutil.Prefix) {
			continue
		}
		instName := userutil.InstanceFromLabel(u.Name)
		if err = ValidateName(instName); err != nil {
			slog.WarnContext(ctx, "Ignoring a user with an invalid instance name", "user", u.Name, "error", err)
			continue
		}
		if slices.ContainsFunc(res, func(i Instance) bool { return i.Name == instName }) {
			slog.WarnContext(ctx, "Ignoring an old-format user, as an instance with the same name already exists",
				"user", u.Name, "instance", instName)
			continue
		}
		res = append(res, *legacyInstance(instName, u.Name))
		seenUsers[u.Name] = struct{}{}
	}

	// Accounts that Alcoholless created but that no longer have metadata, e.g.
	// because the configuration directory was lost. They still consume disk space.
	for _, u := range users {
		if _, ok := members[u.Name]; !ok {
			continue
		}
		if _, ok := seenUsers[u.Name]; ok {
			continue
		}
		slog.WarnContext(ctx, "Found an orphan instance user with no metadata. Delete it manually if it is no longer needed.",
			"user", u.Name, "label", u.Label)
	}

	slices.SortFunc(res, func(a, b Instance) int { return strings.Compare(a.Name, b.Name) })
	return res, nil
}

// legacyInstance returns the instance for a user created by an older version of
// Alcoholless, whose user name carries the label.
func legacyInstance(instName, instUser string) *Instance {
	inst := Instance{Version: Version, Name: instName, User: instUser, Legacy: true}
	if info, err := user.Lookup(instUser); err == nil {
		inst.Home = info.HomeDir
		if uid, err := strconv.Atoi(info.Uid); err == nil {
			inst.UID = uid
		}
	}
	return &inst
}

// Inspect returns the instance, or nil if it does not exist.
func Inspect(ctx context.Context, instName string) (*Instance, error) {
	path, err := instancePath(instName) // validates instName
	if err != nil {
		return nil, err
	}
	members, err := userutil.GroupMembers(ctx)
	if err != nil {
		return nil, err
	}
	inst, err := load(path, members)
	if err == nil {
		return inst, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		slog.WarnContext(ctx, "Ignoring an invalid instance metadata file", "instance", instName, "error", err)
	}
	// Fall back to the old format, whose user name carries the label
	legacyUser := userutil.LabelFromInstance(instName)
	if exists, err := userutil.Exists(legacyUser); err != nil {
		return nil, err
	} else if !exists {
		return nil, nil
	}
	return legacyInstance(instName, legacyUser), nil
}

// WarnIfLegacy warns that the instance was created by an older version of Alcoholless.
func (inst *Instance) WarnIfLegacy(ctx context.Context) {
	if inst == nil || !inst.Legacy {
		return
	}
	slog.WarnContext(ctx, "This instance was created by an older version of Alcoholless, and has a long home directory path, which makes Homebrew build bottles from source instead of pouring them. Consider recreating the instance.",
		"instance", inst.Name, "user", inst.User, "home", inst.Home,
		"hint", fmt.Sprintf("alclessctl delete %s && alclessctl create %s", inst.Name, inst.Name))
}

func ValidateName(name string) error {
	const reserved = "alcless_"
	if strings.HasPrefix(name, reserved) {
		return fmt.Errorf("instance name must not start with %q", reserved)
	}
	if err := identifiers.Validate(name); err != nil {
		return err
	}
	return nil
}
