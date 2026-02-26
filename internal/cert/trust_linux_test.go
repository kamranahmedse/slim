//go:build linux

package cert

import (
	"errors"
	"reflect"
	"testing"
)

func TestTrustCAUsesUpdateCACertificates(t *testing.T) {
	restore := snapshotTrustLinuxTestHooks()
	defer restore()

	readCertFileFn = func(string) ([]byte, error) { return []byte("pem"), nil }
	commandExistsFn = func(name string) bool { return name == "update-ca-certificates" }

	var anchorPath string
	var anchorContent []byte
	writeAnchorFileFn = func(path string, content []byte) error {
		anchorPath = path
		anchorContent = content
		return nil
	}

	var commands [][]string
	runPrivilegedTrustFn = func(name string, args ...string) ([]byte, error) {
		commands = append(commands, append([]string{name}, args...))
		return nil, nil
	}

	if err := TrustCA(); err != nil {
		t.Fatalf("TrustCA: %v", err)
	}

	if anchorPath != debianAnchorPath {
		t.Fatalf("expected anchor path %q, got %q", debianAnchorPath, anchorPath)
	}
	if string(anchorContent) != "pem" {
		t.Fatalf("expected anchor content to be written")
	}

	wantCommands := [][]string{{"update-ca-certificates"}}
	if !reflect.DeepEqual(commands, wantCommands) {
		t.Fatalf("unexpected commands: got %v want %v", commands, wantCommands)
	}
}

func TestTrustCAUsesUpdateCATrust(t *testing.T) {
	restore := snapshotTrustLinuxTestHooks()
	defer restore()

	readCertFileFn = func(string) ([]byte, error) { return []byte("pem"), nil }
	commandExistsFn = func(name string) bool { return name == "update-ca-trust" }
	detectTrustAnchorPathFn = func() string { return archAnchorPath }

	var anchorPath string
	writeAnchorFileFn = func(path string, content []byte) error {
		anchorPath = path
		return nil
	}

	var commands [][]string
	runPrivilegedTrustFn = func(name string, args ...string) ([]byte, error) {
		commands = append(commands, append([]string{name}, args...))
		return nil, nil
	}

	if err := TrustCA(); err != nil {
		t.Fatalf("TrustCA: %v", err)
	}

	if anchorPath != archAnchorPath {
		t.Fatalf("expected anchor path %q, got %q", archAnchorPath, anchorPath)
	}

	wantCommands := [][]string{{"update-ca-trust", "extract"}}
	if !reflect.DeepEqual(commands, wantCommands) {
		t.Fatalf("unexpected commands: got %v want %v", commands, wantCommands)
	}
}

func TestTrustCAFailsWhenNoSupportedTool(t *testing.T) {
	restore := snapshotTrustLinuxTestHooks()
	defer restore()

	readCertFileFn = func(string) ([]byte, error) { return []byte("pem"), nil }
	commandExistsFn = func(string) bool { return false }

	if err := TrustCA(); err == nil {
		t.Fatal("expected TrustCA to fail without supported tools")
	}
}

func TestUntrustCADeletesAnchorsAndUpdatesStore(t *testing.T) {
	restore := snapshotTrustLinuxTestHooks()
	defer restore()

	commandExistsFn = func(name string) bool { return name == "update-ca-certificates" }

	var removed []string
	removeAnchorFileFn = func(path string) error {
		removed = append(removed, path)
		return nil
	}

	var commands [][]string
	runPrivilegedTrustFn = func(name string, args ...string) ([]byte, error) {
		commands = append(commands, append([]string{name}, args...))
		return nil, nil
	}

	if err := UntrustCA(); err != nil {
		t.Fatalf("UntrustCA: %v", err)
	}

	wantRemoved := []string{debianAnchorPath, rhelAnchorPath, archAnchorPath}
	if !reflect.DeepEqual(removed, wantRemoved) {
		t.Fatalf("unexpected removed paths: got %v want %v", removed, wantRemoved)
	}

	wantCommands := [][]string{{"update-ca-certificates"}}
	if !reflect.DeepEqual(commands, wantCommands) {
		t.Fatalf("unexpected commands: got %v want %v", commands, wantCommands)
	}
}

func TestUntrustCAPropagatesRemoveError(t *testing.T) {
	restore := snapshotTrustLinuxTestHooks()
	defer restore()

	removeAnchorFileFn = func(path string) error {
		if path == rhelAnchorPath {
			return errors.New("boom")
		}
		return nil
	}
	commandExistsFn = func(string) bool { return false }

	if err := UntrustCA(); err == nil {
		t.Fatal("expected UntrustCA to fail on remove error")
	}
}

func snapshotTrustLinuxTestHooks() func() {
	prevRead := readCertFileFn
	prevExists := commandExistsFn
	prevWrite := writeAnchorFileFn
	prevRun := runPrivilegedTrustFn
	prevRemove := removeAnchorFileFn
	prevDetect := detectTrustAnchorPathFn

	return func() {
		readCertFileFn = prevRead
		commandExistsFn = prevExists
		writeAnchorFileFn = prevWrite
		runPrivilegedTrustFn = prevRun
		removeAnchorFileFn = prevRemove
		detectTrustAnchorPathFn = prevDetect
	}
}
