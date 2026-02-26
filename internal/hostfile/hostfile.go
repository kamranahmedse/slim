package hostfile

import (
	"fmt"
	"os"
	"strings"

	"github.com/kamrify/localname/internal/osutil"
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
	return osutil.WriteFileElevated(hostsPath, updated)
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

	return osutil.WriteFileElevated(hostsPath, strings.Join(filtered, "\n"))
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

	return osutil.WriteFileElevated(hostsPath, strings.Join(filtered, "\n"))
}
