//go:build darwin

package cert

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kamranahmedse/slim/internal/config"
)

func TestTrustCAUsesExpectedSecurityCommand(t *testing.T) {
	restore := snapshotTrustDarwinExecHook()
	defer restore()
	initTrustDarwinConfig(t)

	var got [][]string
	execCommandDarwinFn = func(name string, args ...string) *exec.Cmd {
		got = append(got, append([]string{name}, args...))
		return exec.Command("sh", "-c", "exit 0")
	}

	if err := TrustCA(); err != nil {
		t.Fatalf("TrustCA: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("expected one command invocation, got %d", len(got))
	}

	wantPrefix := []string{
		"sudo", "security", "add-trusted-cert",
		"-d", "-r", "trustRoot",
		"-k", "/Library/Keychains/System.keychain",
	}
	for i := range wantPrefix {
		if got[0][i] != wantPrefix[i] {
			t.Fatalf("arg[%d] = %q, want %q", i, got[0][i], wantPrefix[i])
		}
	}
	if got[0][len(got[0])-1] != CACertPath() {
		t.Fatalf("expected cert path arg %q, got %q", CACertPath(), got[0][len(got[0])-1])
	}
}

func TestTrustCAErrorIncludesCommandOutput(t *testing.T) {
	restore := snapshotTrustDarwinExecHook()
	defer restore()

	execCommandDarwinFn = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "echo security failed; exit 1")
	}

	err := TrustCA()
	if err == nil {
		t.Fatal("expected TrustCA error")
	}
	if !strings.Contains(err.Error(), "security failed") {
		t.Fatalf("expected command output in error, got: %v", err)
	}
}

func TestUntrustCAUsesExpectedSecurityCommand(t *testing.T) {
	restore := snapshotTrustDarwinExecHook()
	defer restore()
	initTrustDarwinConfig(t)
	createDummyCACert(t)

	var got [][]string
	execCommandDarwinFn = func(name string, args ...string) *exec.Cmd {
		got = append(got, append([]string{name}, args...))
		return exec.Command("sh", "-c", "exit 0")
	}

	if err := UntrustCA(); err != nil {
		t.Fatalf("UntrustCA: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("expected one command invocation, got %d", len(got))
	}

	want := []string{"sudo", "security", "remove-trusted-cert", "-d", CACertPath()}
	if len(got[0]) != len(want) {
		t.Fatalf("unexpected arg count: got %d want %d", len(got[0]), len(want))
	}
	for i := range want {
		if got[0][i] != want[i] {
			t.Fatalf("arg[%d] = %q, want %q", i, got[0][i], want[i])
		}
	}
}

func TestUntrustCAErrorIncludesCommandOutput(t *testing.T) {
	restore := snapshotTrustDarwinExecHook()
	defer restore()
	initTrustDarwinConfig(t)
	createDummyCACert(t)

	execCommandDarwinFn = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "echo remove failed; exit 1")
	}

	err := UntrustCA()
	if err == nil {
		t.Fatal("expected UntrustCA error")
	}
	if !strings.Contains(err.Error(), "remove failed") {
		t.Fatalf("expected command output in error, got: %v", err)
	}
}

func snapshotTrustDarwinExecHook() func() {
	prev := execCommandDarwinFn
	return func() {
		execCommandDarwinFn = prev
	}
}

func initTrustDarwinConfig(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := config.Init(); err != nil {
		t.Fatalf("config.Init: %v", err)
	}
}

func createDummyCACert(t *testing.T) {
	t.Helper()
	path := CACertPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("creating CA dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("dummy"), 0644); err != nil {
		t.Fatalf("creating dummy CA cert: %v", err)
	}
}
