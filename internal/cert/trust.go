package cert

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

func TrustCA() error {
	if runtime.GOOS != "darwin" {
		return errors.New("trusting CA is only supported on macOS")
	}

	cmd := exec.Command("sudo", "security", "add-trusted-cert",
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
	if runtime.GOOS != "darwin" {
		return errors.New("untrusting CA is only supported on macOS")
	}

	cmd := exec.Command("sudo", "security", "remove-trusted-cert",
		"-d", CACertPath(),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("untrusting CA: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}
