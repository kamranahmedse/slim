package hostfile

import (
	"fmt"
	"os"
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

	if err := os.WriteFile(hostsPath, []byte(updated), 0644); err != nil {
		return fmt.Errorf("writing hosts file: %w (try running with sudo)", err)
	}
	return nil
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

	updated := strings.Join(filtered, "\n")
	if err := os.WriteFile(hostsPath, []byte(updated), 0644); err != nil {
		return fmt.Errorf("writing hosts file: %w (try running with sudo)", err)
	}
	return nil
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

	updated := strings.Join(filtered, "\n")
	if err := os.WriteFile(hostsPath, []byte(updated), 0644); err != nil {
		return fmt.Errorf("writing hosts file: %w (try running with sudo)", err)
	}
	return nil
}
