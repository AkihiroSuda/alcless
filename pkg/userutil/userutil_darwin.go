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

func GroupUsers(ctx context.Context, group string) ([]string, error) {
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "dscl", ".", "list", "/Groups/"+group, "GroupMembership")
	cmd.Stderr = &stderr
	slog.DebugContext(ctx, "Running command", "cmd", cmd.Args)
	b, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run %v: %w (stderr=%q)", cmd.Args, err, stderr.String())
	}

	lines := strings.Split(string(b), "\n")
	var res []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) >= 2 && parts[0] == "GroupMembership:" {
			users := strings.Fields(parts[1])
			res = append(res, users...)
		}
	}
	return res, nil
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

func GroupSetupCmds(ctx context.Context, instUser, groupName string) ([]*exec.Cmd, error) {
	var cmds []*exec.Cmd

	sudoersContent, err := sudo.Sudoers(instUser)
	if err != nil {
		return nil, err
	}
	sudoersPath, err := sudo.SudoersPath(instUser)
	if err != nil {
		return nil, err
	}

	cmds = append(cmds,
		exec.CommandContext(ctx, "sudo", "chmod", "go-rx", filepath.Join("/Users", instUser)),
		exec.CommandContext(ctx, "sudo", "sh", "-c", fmt.Sprintf("mkdir -p /etc/sudoers.d && echo '%s' >'%s' && chmod 440 '%s'", sudoersContent, sudoersPath, sudoersPath)),
	)

	if groupName != "" {
		cmds = append(cmds,
			exec.CommandContext(ctx, "sudo", "sh", "-c", fmt.Sprintf("dscl . -create /Groups/%s 2>/dev/null || true", groupName)),
			exec.CommandContext(ctx, "sudo", "sh", "-c", fmt.Sprintf("dscl . -append /Groups/%s GroupMembership %s", groupName, instUser)),
		)
	}

	return cmds, nil
}
