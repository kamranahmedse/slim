package osutil

import (
	"os"
	"os/exec"
	"strings"
)

func WriteFileElevated(path string, content string) error {
	err := os.WriteFile(path, []byte(content), 0644)
	if err == nil {
		return nil
	}

	if !os.IsPermission(err) {
		return err
	}

	cmd := exec.Command("sudo", "tee", path)
	cmd.Stdin = strings.NewReader(content)
	cmd.Stdout = nil
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
