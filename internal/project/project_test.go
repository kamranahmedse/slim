package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kamranahmedse/slim/internal/config"
)

func TestFind(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "a", "b", "c")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	origGetwd := getwdFn
	origStat := statFn
	defer func() {
		getwdFn = origGetwd
		statFn = origStat
	}()

	// Place .slim.yaml in tmpDir, search from subDir
	configPath := filepath.Join(tmpDir, FileName)
	if err := os.WriteFile(configPath, []byte("services: []\n"), 0644); err != nil {
		t.Fatal(err)
	}

	getwdFn = func() (string, error) { return subDir, nil }
	statFn = os.Stat

	got, err := Find()
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if got != configPath {
		t.Fatalf("expected %q, got %q", configPath, got)
	}
}

func TestFindNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	origGetwd := getwdFn
	defer func() { getwdFn = origGetwd }()

	getwdFn = func() (string, error) { return tmpDir, nil }

	_, err := Find()
	if err == nil {
		t.Fatal("expected error when no .slim.yaml found")
	}
	if !strings.Contains(err.Error(), "no .slim.yaml found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadAndValidate(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, FileName)

	content := `services:
  - domain: myapp
    port: 3000
    routes:
      - path: /api
        port: 8080
  - domain: dashboard
    port: 5173
log_mode: minimal
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	pc, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(pc.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(pc.Services))
	}
	if pc.Services[0].Domain != "myapp" || pc.Services[0].Port != 3000 {
		t.Fatalf("unexpected first service: %+v", pc.Services[0])
	}
	if len(pc.Services[0].Routes) != 1 || pc.Services[0].Routes[0].Path != "/api" {
		t.Fatalf("unexpected routes: %+v", pc.Services[0].Routes)
	}
	if pc.LogMode != "minimal" {
		t.Fatalf("expected log_mode minimal, got %q", pc.LogMode)
	}

	if err := pc.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestValidateDuplicate(t *testing.T) {
	pc := &ProjectConfig{
		Services: []Service{
			{Domain: "myapp", Port: 3000},
			{Domain: "myapp", Port: 4000},
		},
	}
	err := pc.Validate()
	if err == nil {
		t.Fatal("expected error for duplicate domains")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateEmptyServices(t *testing.T) {
	pc := &ProjectConfig{}
	err := pc.Validate()
	if err == nil {
		t.Fatal("expected error for empty services")
	}
}

func TestValidateInvalidRoute(t *testing.T) {
	pc := &ProjectConfig{
		Services: []Service{
			{Domain: "myapp", Port: 3000, Routes: []config.Route{{Path: "api", Port: 8080}}},
		},
	}
	err := pc.Validate()
	if err == nil {
		t.Fatal("expected error for route without leading slash")
	}
}

func TestDiscover(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, FileName)

	content := `services:
  - domain: myapp
    port: 3000
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	origGetwd := getwdFn
	defer func() { getwdFn = origGetwd }()
	getwdFn = func() (string, error) { return tmpDir, nil }

	pc, foundPath, err := Discover()
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if foundPath != path {
		t.Fatalf("expected path %q, got %q", path, foundPath)
	}
	if len(pc.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(pc.Services))
	}
}
