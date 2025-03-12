// Package sudo provides sudo utilities.
//
// su is wrapped inside sudo, so as to create a launchd session, which is necessary to isolate `open(1)`.
// sudo cannot create a session because `/etc/pam.d/sudo` lacks the config for `pam_launchd.so`.
package sudo

import (
	"context"
	"fmt"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"al.essio.dev/pkg/shellescape"
)

func SudoersPath(instUser string) (string, error) {
	return filepath.Join("/etc/sudoers.d/", instUser), nil
}

func Sudoers(instUser string) (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s ALL=(root) NOPASSWD: /usr/bin/su - %s -c *", currentUser.Username, instUser), nil
}

func Cmd(ctx context.Context, instUser, wd, cmdExe string, cmdArgs []string) *exec.Cmd {
	quotedArgs := make([]string, len(cmdArgs))
	for i, f := range cmdArgs {
		quotedArgs[i] = shellescape.Quote(f)
	}
	snippet := fmt.Sprintf("cd %s ; exec %s %s", // cd may fail
		shellescape.Quote(wd), // can be empty
		shellescape.Quote(cmdExe),
		strings.Join(quotedArgs, " "))
	cmd := exec.CommandContext(ctx, "sudo", "-n", "/usr/bin/su", "-", instUser, "-c", snippet)
	return cmd
}
