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

package rsync

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"al.essio.dev/pkg/shellescape"
)

type opts struct {
	dryRun bool
}

type Opt func(o *opts) error

func WithDryRun() Opt {
	return func(o *opts) error {
		o.dryRun = true
		return nil
	}
}

func Cmd(ctx context.Context, instName string, src, dst string, o ...Opt) (*exec.Cmd, error) {
	var opts opts
	for _, f := range o {
		if err := f(&opts); err != nil {
			return nil, err
		}
	}
	selfExe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	rsyncE := fmt.Sprintf("%s shell --workdir=/ --plain", shellescape.Quote(selfExe))
	args := []string{
		"-rai",
		"--delete",
		"-e", rsyncE,
		src,
		dst,
	}
	if opts.dryRun {
		args = append([]string{"--dry-run"}, args...)
	}
	cmd := exec.CommandContext(ctx, "rsync", args...)
	return cmd, nil
}
