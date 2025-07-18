package brew

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"os/user"
	"path/filepath"

	"github.com/AkihiroSuda/alcless/pkg/sudo"
)

func InstalledCmd(ctx context.Context, instUser, homeDir string) *exec.Cmd {
	return sudo.Cmd(ctx, instUser, "", filepath.Join(homeDir, "homebrew/bin/brew"), []string{"--version"})
}

func Installed(ctx context.Context, instUser string) error {
	instUserInfo, err := user.Lookup(instUser)
	if err != nil {
		return err
	}
	if instUserInfo.HomeDir == "" {
		return fmt.Errorf("user %q does not have the home directory", instUser)
	}
	var stderr bytes.Buffer
	cmd := InstalledCmd(ctx, instUser, instUserInfo.HomeDir)
	cmd.Stderr = &stderr
	slog.DebugContext(ctx, "Running command", "cmd", cmd.Args)
	b, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to run %v: %w (stderr=%q)", cmd.Args, err, stderr.String())
	}
	slog.DebugContext(ctx, "Homebrew has been already installed", "user", instUser, "version", string(b))
	return nil
}

func InstallCmds(ctx context.Context, instUser string) []*exec.Cmd {
	cmds := []*exec.Cmd{
		sudo.Cmd(ctx, instUser, "", "git", []string{"clone", "https://github.com/Homebrew/brew", "homebrew"}),
		sudo.Cmd(ctx, instUser, "", "sh", []string{"-c", `echo 'eval "$("${HOME}/homebrew/bin/brew" shellenv)"' | tee -a "${HOME}/.bash_profile" >> "${HOME}/.zshenv"`}),
	}
	return cmds
}
