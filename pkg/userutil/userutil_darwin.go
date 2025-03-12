package userutil

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sethvargo/go-password/password"

	"github.com/AkihiroSuda/alcless/pkg/sudo"
)

func Users(ctx context.Context) ([]string, error) {
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "dscl", ".", "list", "/Users")
	cmd.Stderr = &stderr
	slog.DebugContext(ctx, "Running command", "cmd", cmd.Args)
	b, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run %v: %w (stderr=%q)", cmd.Args, err, stderr.String())
	}
	var res []string
	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		res = append(res, scanner.Text())
	}
	return res, scanner.Err()
}

type Attribute string

const (
	AttributeUserShell = Attribute("UserShell")
)

func ReadAttribute(ctx context.Context, username string, k Attribute) (string, error) {
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "dscl", ".", "-read", "/Users/"+username, string(k))
	cmd.Stderr = &stderr
	slog.DebugContext(ctx, "Running command", "cmd", cmd.Args)
	b, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run %v: %w (stderr=%q)", cmd.Args, err, stderr.String())
	}
	s := string(b)
	s = strings.TrimPrefix(s, string(k)+":")
	s = strings.TrimSpace(s)
	return s, nil
}

func AddUserCmds(ctx context.Context, instUser string, tty bool) ([]*exec.Cmd, error) {
	sudoersContent, err := sudo.Sudoers(instUser)
	if err != nil {
		return nil, err
	}
	sudoersPath, err := sudo.SudoersPath(instUser)
	if err != nil {
		return nil, err
	}
	sudoersCmd := fmt.Sprintf("echo '%s' >'%s'", sudoersContent, sudoersPath)
	pw := "-"
	if !tty {
		pw, err := password.Generate(64, 10, 10, false, false)
		if err != nil {
			return nil, err
		}
		slog.WarnContext(ctx, "Generated a random password, as tty is not available. THE PASSWORD IS SHOWN IN THIS SCREEN.", "user", instUser, "password", pw)
	}
	return []*exec.Cmd{
		exec.CommandContext(ctx, "sudo", "sysadminctl", "-addUser", instUser, "-password", pw),
		exec.CommandContext(ctx, "sudo", "chmod", "go-rx", filepath.Join("/Users", instUser)),
		exec.CommandContext(ctx, "sudo", "sh", "-c", sudoersCmd),
	}, nil
}

func DeleteUserCmds(ctx context.Context, instUser string) ([]*exec.Cmd, error) {
	sudoersPath, err := sudo.SudoersPath(instUser)
	if err != nil {
		return nil, err
	}
	cmds := []*exec.Cmd{
		// Not sure what "-secure" does
		exec.CommandContext(ctx, "sudo", "sysadminctl", "-deleteUser", instUser, "-secure"),
		exec.CommandContext(ctx, "sudo", "rm", "-f", sudoersPath),
	}
	return cmds, nil
}
