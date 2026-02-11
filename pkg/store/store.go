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
	"fmt"
	"strings"

	"github.com/containerd/containerd/v2/pkg/identifiers"

	"github.com/AkihiroSuda/alcless/pkg/userutil"
)

type Instance struct {
	Name string `json:"name"`
	User string `json:"user"`
}

func Instances(ctx context.Context) ([]Instance, error) {
	users, err := userutil.Users(ctx)
	if err != nil {
		return nil, err
	}
	var res []Instance
	for _, u := range users {
		if !strings.HasPrefix(u, userutil.Prefix) {
			continue
		}
		instName := userutil.InstanceFromUser(u)
		if err = ValidateName(instName); err != nil {
			return res, err
		}
		res = append(res, Instance{Name: instName, User: u})
	}
	return res, nil
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
