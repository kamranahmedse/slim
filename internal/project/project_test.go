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

func TestLoadAndValidateWithBaseDomains(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, FileName)

	content := `base_domain: local.example.com
services:
  - domain: myapp
    port: 3000
  - domain: dashboard
    base_domain: preview.example.com
    port: 5173
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	pc, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if pc.BaseDomain != "local.example.com" {
		t.Fatalf("expected project base_domain, got %q", pc.BaseDomain)
	}
	if pc.Services[1].BaseDomain != "preview.example.com" {
		t.Fatalf("expected service base_domain, got %q", pc.Services[1].BaseDomain)
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

func TestServiceName(t *testing.T) {
	tests := []struct {
		domain string
		want   string
	}{
		{"myapp", "myapp"},
		{"dashboard", "dashboard"},
	}
	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			svc := Service{Domain: tt.domain, Port: 3000}
			if got := svc.Name(); got != tt.want {
				t.Errorf("Service{Domain: %q}.Name() = %q, want %q", tt.domain, got, tt.want)
			}
		})
	}
}

func TestServiceHostname(t *testing.T) {
	tests := []struct {
		name              string
		service           Service
		projectBaseDomain string
		want              string
	}{
		{
			name:    "default",
			service: Service{Domain: "myapp", Port: 3000},
			want:    "myapp.test",
		},
		{
			name:              "project base domain",
			service:           Service{Domain: "myapp", Port: 3000},
			projectBaseDomain: "local.example.com",
			want:              "myapp.local.example.com",
		},
		{
			name:              "service base domain overrides project",
			service:           Service{Domain: "myapp", BaseDomain: "preview.example.com", Port: 3000},
			projectBaseDomain: "local.example.com",
			want:              "myapp.preview.example.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.service.Hostname(tt.projectBaseDomain); got != tt.want {
				t.Errorf("Service{Domain: %q}.Hostname() = %q, want %q", tt.service.Domain, got, tt.want)
			}
		})
	}
}

func TestServiceConfigDomain(t *testing.T) {
	t.Run("default base domain", func(t *testing.T) {
		svc := Service{Domain: "myapp", Port: 3000, Routes: []config.Route{{Path: "/api", Port: 8080}}}
		d := svc.ConfigDomain("")
		if d.Name != "myapp" {
			t.Fatalf("expected Name %q, got %q", "myapp", d.Name)
		}
		if d.Hostname != "" {
			t.Fatalf("expected empty Hostname for simple domain, got %q", d.Hostname)
		}
		if d.Port != 3000 {
			t.Fatalf("expected Port 3000, got %d", d.Port)
		}
		if len(d.Routes) != 1 || d.Routes[0].Path != "/api" {
			t.Fatalf("unexpected routes: %+v", d.Routes)
		}
		if got := d.ResolvedHostname(); got != "myapp.test" {
			t.Fatalf("expected ResolvedHostname %q, got %q", "myapp.test", got)
		}
	})

	t.Run("custom base domain", func(t *testing.T) {
		svc := Service{Domain: "dashboard", BaseDomain: "local.example.com", Port: 4000}
		d := svc.ConfigDomain("")
		if d.Name != "dashboard" {
			t.Fatalf("expected Name %q, got %q", "dashboard", d.Name)
		}
		if d.Hostname != "local.example.com" {
			t.Fatalf("expected Hostname %q, got %q", "local.example.com", d.Hostname)
		}
		if got := d.ResolvedHostname(); got != "dashboard.local.example.com" {
			t.Fatalf("expected ResolvedHostname %q, got %q", "dashboard.local.example.com", got)
		}
	})

	t.Run("project base domain", func(t *testing.T) {
		svc := Service{Domain: "dashboard", Port: 4000}
		d := svc.ConfigDomain("local.example.com")
		if d.Hostname != "local.example.com" {
			t.Fatalf("expected Hostname %q, got %q", "local.example.com", d.Hostname)
		}
		if got := d.ResolvedHostname(); got != "dashboard.local.example.com" {
			t.Fatalf("expected ResolvedHostname %q, got %q", "dashboard.local.example.com", got)
		}
	})
}

func TestValidateDuplicateResolvedHostnames(t *testing.T) {
	pc := &ProjectConfig{
		BaseDomain: "example.com",
		Services: []Service{
			{Domain: "myapp", BaseDomain: "local.example.com", Port: 3000},
			{Domain: "myapp.local", BaseDomain: "example.com", Port: 4000},
		},
	}
	err := pc.Validate()
	if err == nil {
		t.Fatal("expected error for duplicate resolved hostnames")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateDuplicateNamesAcrossBaseDomains(t *testing.T) {
	pc := &ProjectConfig{
		Services: []Service{
			{Domain: "myapp", BaseDomain: "local.example.com", Port: 3000},
			{Domain: "myapp", BaseDomain: "preview.example.com", Port: 4000},
		},
	}
	err := pc.Validate()
	if err == nil {
		t.Fatal("expected error for duplicate service names")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("unexpected error: %v", err)
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
