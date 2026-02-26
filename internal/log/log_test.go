package log

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRequestWritesFullMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "access.log")
	if err := SetOutput(path, "full"); err != nil {
		t.Fatalf("SetOutput: %v", err)
	}
	t.Cleanup(Close)

	Request("myapp.local", "GET", "/health", 3000, 200, 12*time.Millisecond)
	Close()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	fields := strings.Split(lines[0], "\t")
	if len(fields) != 7 {
		t.Fatalf("expected 7 fields in full mode, got %d: %q", len(fields), lines[0])
	}
}

func TestRequestWritesMinimalMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "access.log")
	if err := SetOutput(path, "minimal"); err != nil {
		t.Fatalf("SetOutput: %v", err)
	}
	t.Cleanup(Close)

	Request("myapp.local", "GET", "/health", 3000, 200, 12*time.Millisecond)
	Close()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	fields := strings.Split(lines[0], "\t")
	if len(fields) != 4 {
		t.Fatalf("expected 4 fields in minimal mode, got %d: %q", len(fields), lines[0])
	}
}

func TestRequestOffModeWritesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "access.log")
	if err := SetOutput(path, "off"); err != nil {
		t.Fatalf("SetOutput: %v", err)
	}
	t.Cleanup(Close)

	Request("myapp.local", "GET", "/health", 3000, 200, 12*time.Millisecond)
	Close()

	_, err := os.Stat(path)
	if !os.IsNotExist(err) {
		t.Fatalf("expected no log file, got err=%v", err)
	}
}

func TestSetOutputReconfigureFlushesPreviousWriter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "access.log")
	if err := SetOutput(path, "full"); err != nil {
		t.Fatalf("SetOutput full: %v", err)
	}
	t.Cleanup(Close)

	Request("myapp.local", "GET", "/one", 3000, 200, 10*time.Millisecond)
	if err := SetOutput(path, "minimal"); err != nil {
		t.Fatalf("SetOutput minimal: %v", err)
	}
	Request("myapp.local", "GET", "/two", 3000, 200, 10*time.Millisecond)
	Close()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines after reconfigure, got %d", len(lines))
	}

	first := strings.Split(lines[0], "\t")
	second := strings.Split(lines[1], "\t")
	if len(first) != 7 || len(second) != 4 {
		t.Fatalf("expected full+minimal line formats, got %d and %d", len(first), len(second))
	}
}
