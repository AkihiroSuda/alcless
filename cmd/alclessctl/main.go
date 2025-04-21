// Portions from https://github.com/lima-vm/lima/blob/v1.1.0-beta.0/cmd/limactl/main.go
//
// SPDX-FileCopyrightText: Copyright The Lima Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/lmittmann/tint"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/AkihiroSuda/alcless/cmd/alclessctl/commands/create"
	"github.com/AkihiroSuda/alcless/cmd/alclessctl/commands/delete"
	"github.com/AkihiroSuda/alcless/cmd/alclessctl/commands/list"
	"github.com/AkihiroSuda/alcless/cmd/alclessctl/commands/shell"
	"github.com/AkihiroSuda/alcless/cmd/alclessctl/version"
	"github.com/AkihiroSuda/alcless/pkg/envutil"
)

var logLevel = new(slog.LevelVar)

func main() {
	logHandler := tint.NewHandler(os.Stderr, &tint.Options{
		Level:      logLevel,
		TimeFormat: time.Kitchen,
	})
	slog.SetDefault(slog.New(logHandler))
	if err := newRootCommand().Execute(); err != nil {
		exitCode := 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			if ps := exitErr.ProcessState; ps != nil {
				exitCode = ps.ExitCode()
			}
		}
		slog.Error("exiting with an error: " + err.Error())
		os.Exit(exitCode)
	}
}

const example = `
  Create the default instance:
  $ alclessctl create default

  Run commands (long form):
  $ cd ~/SOME_DIRECTORY
  $ alclessctl shell default brew install xz
  $ alclessctl shell default xz SOME_FILE

  Run commands (short form):
  $ cd ~/SOME_DIRECTORY
  $ alcless brew install xz
  $ alcless xz

  Delete the default instance:
  $ alclessctl create default`

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alclessctl",
		Short: "Alcoholless: lightweight sandbox for Homebrew",
		Long: `Alcoholless: lightweight sandbox for Homebrew

⚠️ Do NOT report any issue that happens with Alcoholless to the upstream Homebrew.
`,
		Example:       example,
		Version:       version.GetVersion(),
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	flags := cmd.PersistentFlags()
	flags.Bool("debug", envutil.Bool("DEBUG", false), "debug mode [$DEBUG]")
	// Follows limactl's CLI convention, although "tty" was a sort of misnomer.
	flags.Bool("tty", term.IsTerminal(int(os.Stdout.Fd())), "enable TUI interactions. Defaults to true when stdout is a terminal. Set to false for automation.")
	flags.BoolP("yes", "y", false, "Alias of --tty=false")
	flags.Bool("plain", false, "plain mode (no Homebrew integration, file syncing, etc.)")

	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		flags := cmd.Flags()
		if debug, _ := flags.GetBool("debug"); debug {
			logLevel.Set(slog.LevelDebug)
		}
		if flags.Changed("yes") && flags.Changed("tty") {
			return errors.New("cannot use both --tty and --yes flags at the same time")
		}
		if flags.Changed("yes") {
			// Sets the value of the yesValue flag by using the yes flag.
			yesValue, _ := flags.GetBool("yes")
			if yesValue {
				// Sets to the default value false
				err := flags.Set("tty", "false")
				if err != nil {
					return err
				}
			}
		}
		return nil
	}

	cmd.AddCommand(
		list.New(),
		create.New(),
		delete.New(),
		shell.New(),
	)
	return cmd
}
