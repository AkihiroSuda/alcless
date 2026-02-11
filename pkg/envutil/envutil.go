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

// Package envutil is from https://github.com/reproducible-containers/repro-get/blob/v0.4.0/pkg/envutil/envutil.go
package envutil

import (
	"fmt"
	"os"
	"strconv"

	"log/slog"
)

func Bool(envName string, defaultValue bool) bool {
	v, ok := os.LookupEnv(envName)
	if !ok {
		return defaultValue
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		slog.Warn(fmt.Sprintf("Failed to parse %q ($%s) as a boolean: %v", v, envName, err))
		return defaultValue
	}
	return b
}
