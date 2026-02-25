package cert

import (
	"fmt"
	"os/exec"
	"runtime"
)

func TrustCA() error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("CA trust is only supported on macOS currently")
	}

	cmd := exec.Command("sudo", "security", "add-trusted-cert",
		"-d", "-r", "trustRoot",
		"-k", "/Library/Keychains/System.keychain",
		CACertPath(),
	)
	cmd.Stdin = nil
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("trusting CA: %s: %w", string(output), err)
	}
	return nil
}

func UntrustCA() error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("CA untrust is only supported on macOS currently")
	}

	cmd := exec.Command("sudo", "security", "remove-trusted-cert",
		"-d", CACertPath(),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("untrusting CA: %s: %w", string(output), err)
	}
	return nil
}
