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
