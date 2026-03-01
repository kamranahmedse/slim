//go:build darwin

package cert

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var execCommandDarwinFn = exec.Command

func TrustCA() error {
	cmd := execCommandDarwinFn("sudo", "security", "add-trusted-cert",
		"-d", "-r", "trustRoot",
		"-k", "/Library/Keychains/System.keychain",
		CACertPath(),
	)
	cmd.Stdin = nil
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("trusting CA: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func UntrustCA() error {
	if _, err := os.Stat(CACertPath()); os.IsNotExist(err) {
		return nil
	}
	cmd := execCommandDarwinFn("sudo", "security", "remove-trusted-cert",
		"-d", CACertPath(),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("untrusting CA: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}
