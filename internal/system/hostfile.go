package system

import (
	"fmt"
	"os"
	"strings"
)

const hostsPath = "/etc/hosts"
const marker = "# slim"

var (
	readFileHostFn          = os.ReadFile
	writeFileElevatedHostFn = writeFileElevated
)

func AddHost(name string) error {
	hostname := name + ".local"
	entry := fmt.Sprintf("127.0.0.1 %s %s", hostname, marker)

	content, err := readFileHostFn(hostsPath)
	if err != nil {
		return fmt.Errorf("reading hosts file: %w", err)
	}

	if HasMarkedEntry(string(content), hostname) {
		return nil
	}

	updated := strings.TrimRight(string(content), "\n") + "\n" + entry + "\n"
	return writeFileElevatedHostFn(hostsPath, updated)
}

func RemoveHost(name string) error {
	hostname := name + ".local"

	content, err := readFileHostFn(hostsPath)
	if err != nil {
		return fmt.Errorf("reading hosts file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	var filtered []string
	for _, line := range lines {
		if lineHasHost(line, hostname) && strings.Contains(line, marker) {
			continue
		}
		filtered = append(filtered, line)
	}

	return writeFileElevatedHostFn(hostsPath, strings.Join(filtered, "\n"))
}

func RemoveAllHosts() error {
	content, err := readFileHostFn(hostsPath)
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

	return writeFileElevatedHostFn(hostsPath, strings.Join(filtered, "\n"))
}

func HasMarkedEntry(content, hostname string) bool {
	for _, line := range strings.Split(content, "\n") {
		if lineHasHost(line, hostname) && strings.Contains(line, marker) {
			return true
		}
	}
	return false
}

func lineHasHost(line, hostname string) bool {
	for _, field := range strings.Fields(line) {
		if field == hostname {
			return true
		}
	}
	return false
}
