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
	"errors"
	"os/user"
	"strings"
)

// Prefix is the prefix of the user accounts.
var Prefix = "alcless_" + me() + "_"

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

func UserFromInstance(instName string) string {
	return Prefix + instName
}

func InstanceFromUser(username string) string {
	return strings.TrimPrefix(username, Prefix)
}

func Exists(name string) (bool, error) {
	if _, err := user.Lookup(name); err != nil {
		var uee user.UnknownUserError
		if errors.As(err, &uee) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
