package system

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileElevatedDirectWriteSuccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hosts.test")
	content := "127.0.0.1 myapp.local # slim\n"

	if err := writeFileElevated(path, content); err != nil {
		t.Fatalf("writeFileElevated: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != content {
		t.Fatalf("unexpected content: got %q want %q", string(got), content)
	}
}

func TestWriteFileElevatedReturnsNonPermissionError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "hosts.test")

	err := writeFileElevated(path, "x")
	if err == nil {
		t.Fatal("expected error for missing parent directory")
	}
}
