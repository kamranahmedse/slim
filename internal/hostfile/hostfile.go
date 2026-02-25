package hostfile

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const hostsPath = "/etc/hosts"
const marker = "# localname"

func Add(name string) error {
	hostname := name + ".local"
	entry := fmt.Sprintf("127.0.0.1 %s %s", hostname, marker)

	content, err := os.ReadFile(hostsPath)
	if err != nil {
		return fmt.Errorf("reading hosts file: %w", err)
	}

	if strings.Contains(string(content), hostname) {
		return nil
	}

	updated := strings.TrimRight(string(content), "\n") + "\n" + entry + "\n"
	return writeHosts(updated)
}

func Remove(name string) error {
	hostname := name + ".local"

	content, err := os.ReadFile(hostsPath)
	if err != nil {
		return fmt.Errorf("reading hosts file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	var filtered []string
	for _, line := range lines {
		if strings.Contains(line, hostname) && strings.Contains(line, marker) {
			continue
		}
		filtered = append(filtered, line)
	}

	return writeHosts(strings.Join(filtered, "\n"))
}

func RemoveAll() error {
	content, err := os.ReadFile(hostsPath)
	if err != nil {
		return fmt.Errorf("reading hosts file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	var filtered []string
	for _, line := range lines {
		if strings.Contains(line, marker) {
			continue
		}
		filtered = append(filtered, line)
	}

	return writeHosts(strings.Join(filtered, "\n"))
}

func writeHosts(content string) error {
	err := os.WriteFile(hostsPath, []byte(content), 0644)
	if err == nil {
		return nil
	}

	if !os.IsPermission(err) {
		return fmt.Errorf("writing hosts file: %w", err)
	}

	cmd := exec.Command("sudo", "tee", hostsPath)
	cmd.Stdin = strings.NewReader(content)
	cmd.Stdout = nil
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("writing hosts file with sudo: %w", err)
	}
	return nil
}
