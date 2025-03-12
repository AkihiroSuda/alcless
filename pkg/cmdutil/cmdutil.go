// Package cmdutil is expected to be used for running potentially destructive commands with a user confirmation.
package cmdutil

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"

	"al.essio.dev/pkg/shellescape"
	"github.com/spf13/cobra"
)

type RunOpts struct {
	Confirm bool
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
}

func RunOptsFromCobra(cmd *cobra.Command) (*RunOpts, error) {
	flags := cmd.Flags()
	tty, err := flags.GetBool("tty")
	if err != nil {
		return nil, err
	}
	return &RunOpts{
		Confirm: tty,
		Stdin:   cmd.InOrStdin(),
		Stdout:  cmd.OutOrStdout(),
		Stderr:  cmd.ErrOrStderr(),
	}, nil
}

// RunOptsFromCobraNoStdin is similar to RunOptsFromCobra
// but for non-destructive commands.
func RunOptsFromCobraNoStdin(cmd *cobra.Command) (*RunOpts, error) {
	return &RunOpts{
		Stdout: cmd.OutOrStdout(),
		Stderr: cmd.ErrOrStderr(),
	}, nil
}

func Run(ctx context.Context, cmds []*exec.Cmd, opts *RunOpts) error {
	if opts == nil {
		opts = &RunOpts{}
	}

	var stderr io.Writer = os.Stderr
	if opts.Stderr != nil {
		stderr = opts.Stderr
	}
	if opts.Confirm {
		fmt.Fprintln(stderr, "⚠️  The following commands will be executed:")
		for _, c := range cmds {
			fmt.Fprintln(stderr, shellescape.QuoteCommand(c.Args))
		}
		fmt.Fprintln(stderr, "❓ Press return to continue, or Ctrl-C to abort")
		if _, err := fmt.Scanln(); err != nil {
			return err
		}
		fmt.Fprintln(stderr, "CONTINUE")
	}

	for _, c := range cmds {
		argsEscaped := shellescape.QuoteCommand(c.Args)
		if opts.Confirm && len(cmds) > 1 {
			// Always the progress when running multiple destructive commands
			slog.InfoContext(ctx, "Running command", "cmd", argsEscaped)
		} else {
			slog.DebugContext(ctx, "Running command", "cmd", argsEscaped)
		}
		c.Stderr = stderr
		if opts.Stdout != nil {
			c.Stdout = opts.Stdout
		}
		if opts.Stdin != nil {
			c.Stdin = opts.Stdin
		}
		if err := c.Run(); err != nil {
			return fmt.Errorf("failed to run: %v: %w", argsEscaped, err)
		}
		slog.DebugContext(ctx, "Completed command", "cmd", argsEscaped)
	}
	return nil
}

func RunWithCobra(ctx context.Context, cmds []*exec.Cmd, cobraCmd *cobra.Command) error {
	opts, err := RunOptsFromCobra(cobraCmd)
	if err != nil {
		return err
	}
	return Run(ctx, cmds, opts)
}
