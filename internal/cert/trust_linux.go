//go:build linux

package cert

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	debianAnchorPath = "/usr/local/share/ca-certificates/localname.crt"
	rhelAnchorPath   = "/etc/pki/ca-trust/source/anchors/localname.crt"
	archAnchorPath   = "/etc/ca-certificates/trust-source/anchors/localname.crt"
)

func TrustCA() error {
	certPEM, err := os.ReadFile(CACertPath())
	if err != nil {
		return fmt.Errorf("reading CA cert: %w", err)
	}

	if commandExists("update-ca-certificates") {
		if err := writeAnchorFile(debianAnchorPath, certPEM); err != nil {
			return err
		}
		if output, err := runPrivileged("update-ca-certificates"); err != nil {
			return fmt.Errorf("update-ca-certificates failed: %s: %w", strings.TrimSpace(string(output)), err)
		}
		return nil
	}

	if commandExists("update-ca-trust") {
		anchorPath := detectTrustAnchorPath()
		if err := writeAnchorFile(anchorPath, certPEM); err != nil {
			return err
		}
		if output, err := runPrivileged("update-ca-trust", "extract"); err != nil {
			return fmt.Errorf("update-ca-trust failed: %s: %w", strings.TrimSpace(string(output)), err)
		}
		return nil
	}

	return errors.New("no supported Linux CA trust tool found (need update-ca-certificates or update-ca-trust)")
}

func UntrustCA() error {
	for _, path := range []string{debianAnchorPath, rhelAnchorPath, archAnchorPath} {
		if err := removeFilePrivileged(path); err != nil {
			return err
		}
	}

	if commandExists("update-ca-certificates") {
		if output, err := runPrivileged("update-ca-certificates"); err != nil {
			return fmt.Errorf("update-ca-certificates failed: %s: %w", strings.TrimSpace(string(output)), err)
		}
		return nil
	}

	if commandExists("update-ca-trust") {
		if output, err := runPrivileged("update-ca-trust", "extract"); err != nil {
			return fmt.Errorf("update-ca-trust failed: %s: %w", strings.TrimSpace(string(output)), err)
		}
		return nil
	}

	return errors.New("no supported Linux CA trust tool found (need update-ca-certificates or update-ca-trust)")
}

func detectTrustAnchorPath() string {
	switch {
	case dirExists(filepath.Dir(rhelAnchorPath)):
		return rhelAnchorPath
	case dirExists(filepath.Dir(archAnchorPath)):
		return archAnchorPath
	default:
		return rhelAnchorPath
	}
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func writeAnchorFile(path string, content []byte) error {
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0755); err != nil {
		if !os.IsPermission(err) {
			return fmt.Errorf("creating anchor directory %s: %w", parent, err)
		}
		if output, mkdirErr := runPrivileged("mkdir", "-p", parent); mkdirErr != nil {
			return fmt.Errorf("creating anchor directory %s: %s: %w", parent, strings.TrimSpace(string(output)), mkdirErr)
		}
	}

	if err := os.WriteFile(path, content, 0644); err == nil {
		return nil
	} else if !os.IsPermission(err) {
		return fmt.Errorf("writing anchor file %s: %w", path, err)
	}

	cmd := exec.Command("sudo", "tee", path)
	cmd.Stdin = strings.NewReader(string(content))
	cmd.Stdout = nil
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("writing anchor file %s: %s: %w", path, strings.TrimSpace(string(output)), err)
	}
	return nil
}

func removeFilePrivileged(path string) error {
	if err := os.Remove(path); err == nil || os.IsNotExist(err) {
		return nil
	}

	output, err := runPrivileged("rm", "-f", path)
	if err != nil {
		return fmt.Errorf("removing %s: %s: %w", path, strings.TrimSpace(string(output)), err)
	}
	return nil
}

func runPrivileged(name string, args ...string) ([]byte, error) {
	if os.Geteuid() == 0 {
		return exec.Command(name, args...).CombinedOutput()
	}
	all := append([]string{name}, args...)
	return exec.Command("sudo", all...).CombinedOutput()
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
