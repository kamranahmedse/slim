package osutil

import (
	"os"
	"os/exec"
)

func RunPrivileged(name string, args ...string) ([]byte, error) {
	if os.Geteuid() == 0 {
		return exec.Command(name, args...).CombinedOutput()
	}
	all := append([]string{name}, args...)
	return exec.Command("sudo", all...).CombinedOutput()
}

func CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
