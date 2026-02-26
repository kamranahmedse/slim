package system

import (
	"errors"
	"strings"
	"testing"
)

func TestAddHostAppendsMarkedEntry(t *testing.T) {
	restore := snapshotHostFileHooks()
	defer restore()

	readFileHostFn = func(string) ([]byte, error) {
		return []byte("127.0.0.1 localhost\n"), nil
	}

	var wrotePath, wroteContent string
	writeFileElevatedHostFn = func(path string, content string) error {
		wrotePath = path
		wroteContent = content
		return nil
	}

	if err := AddHost("myapp"); err != nil {
		t.Fatalf("AddHost: %v", err)
	}
	if wrotePath != hostsPath {
		t.Fatalf("expected write path %q, got %q", hostsPath, wrotePath)
	}
	if !strings.Contains(wroteContent, "myapp.local") || !strings.Contains(wroteContent, marker) {
		t.Fatalf("expected hosts entry to be appended, got %q", wroteContent)
	}
}

func TestAddHostNoopWhenEntryAlreadyExists(t *testing.T) {
	restore := snapshotHostFileHooks()
	defer restore()

	readFileHostFn = func(string) ([]byte, error) {
		return []byte("127.0.0.1 myapp.local # slim\n"), nil
	}

	called := false
	writeFileElevatedHostFn = func(path string, content string) error {
		called = true
		return nil
	}

	if err := AddHost("myapp"); err != nil {
		t.Fatalf("AddHost: %v", err)
	}
	if called {
		t.Fatal("expected AddHost to skip write when marked entry already exists")
	}
}

func TestRemoveHostRemovesOnlyMarkedMatchingEntry(t *testing.T) {
	restore := snapshotHostFileHooks()
	defer restore()

	readFileHostFn = func(string) ([]byte, error) {
		return []byte(strings.Join([]string{
			"127.0.0.1 localhost",
			"127.0.0.1 myapp.local # slim",
			"127.0.0.1 myapp.local # another-tool",
			"127.0.0.1 api.local # slim",
			"",
		}, "\n")), nil
	}

	var wrote string
	writeFileElevatedHostFn = func(path string, content string) error {
		wrote = content
		return nil
	}

	if err := RemoveHost("myapp"); err != nil {
		t.Fatalf("RemoveHost: %v", err)
	}
	if strings.Contains(wrote, "myapp.local # slim") {
		t.Fatalf("expected marked myapp entry to be removed, got %q", wrote)
	}
	if !strings.Contains(wrote, "myapp.local # another-tool") {
		t.Fatalf("expected non-marked myapp entry to remain, got %q", wrote)
	}
	if !strings.Contains(wrote, "api.local # slim") {
		t.Fatalf("expected other marked entries to remain, got %q", wrote)
	}
}

func TestRemoveAllHostsRemovesAllMarkedEntries(t *testing.T) {
	restore := snapshotHostFileHooks()
	defer restore()

	readFileHostFn = func(string) ([]byte, error) {
		return []byte(strings.Join([]string{
			"127.0.0.1 localhost",
			"127.0.0.1 myapp.local # slim",
			"127.0.0.1 api.local # slim",
			"127.0.0.1 other.local # another-tool",
			"",
		}, "\n")), nil
	}

	var wrote string
	writeFileElevatedHostFn = func(path string, content string) error {
		wrote = content
		return nil
	}

	if err := RemoveAllHosts(); err != nil {
		t.Fatalf("RemoveAllHosts: %v", err)
	}
	if strings.Contains(wrote, "# slim") {
		t.Fatalf("expected all slim marked lines removed, got %q", wrote)
	}
	if !strings.Contains(wrote, "other.local # another-tool") {
		t.Fatalf("expected unrelated entries to remain, got %q", wrote)
	}
}

func TestHostMutatorsPropagateReadErrors(t *testing.T) {
	restore := snapshotHostFileHooks()
	defer restore()

	readFileHostFn = func(string) ([]byte, error) {
		return nil, errors.New("read failed")
	}

	if err := AddHost("myapp"); err == nil {
		t.Fatal("expected AddHost to fail on read error")
	}
	if err := RemoveHost("myapp"); err == nil {
		t.Fatal("expected RemoveHost to fail on read error")
	}
	if err := RemoveAllHosts(); err == nil {
		t.Fatal("expected RemoveAllHosts to fail on read error")
	}
}

func snapshotHostFileHooks() func() {
	prevRead := readFileHostFn
	prevWrite := writeFileElevatedHostFn
	return func() {
		readFileHostFn = prevRead
		writeFileElevatedHostFn = prevWrite
	}
}
