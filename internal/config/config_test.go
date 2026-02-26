package config

import (
	"strings"
	"testing"
)

func TestValidateDomain(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{"myapp", 3000, false},
		{"my-app", 8080, false},
		{"a", 1, false},
		{"abc123", 65535, false},
		{"a-b-c", 3000, false},
		{"123", 3000, false},
		{"", 3000, true},
		{"-abc", 3000, true},
		{"abc-", 3000, true},
		{"ABC", 3000, true},
		{"my_app", 3000, true},
		{"my.app", 3000, true},
		{"my app", 3000, true},
		{strings.Repeat("a", 63), 3000, false},
		{strings.Repeat("a", 64), 3000, true},
		{"myapp", 0, true},
		{"myapp", -1, true},
		{"myapp", 65536, true},
	}

	for _, tt := range tests {
		err := ValidateDomain(tt.name, tt.port)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateDomain(%q, %d) error = %v, wantErr %v", tt.name, tt.port, err, tt.wantErr)
		}
	}
}

func TestConfigLifecycle(t *testing.T) {
	baseDir = t.TempDir()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load empty: %v", err)
	}
	if len(cfg.Domains) != 0 {
		t.Fatalf("expected 0 domains, got %d", len(cfg.Domains))
	}

	if err := cfg.SetDomain("myapp", 3000); err != nil {
		t.Fatalf("SetDomain: %v", err)
	}

	cfg, err = Load()
	if err != nil {
		t.Fatalf("Load after set: %v", err)
	}
	if len(cfg.Domains) != 1 {
		t.Fatalf("expected 1 domain, got %d", len(cfg.Domains))
	}
	if cfg.Domains[0].Name != "myapp" || cfg.Domains[0].Port != 3000 {
		t.Fatalf("unexpected domain: %+v", cfg.Domains[0])
	}

	d, idx := cfg.FindDomain("myapp")
	if d == nil || idx != 0 {
		t.Fatalf("FindDomain: got %v at %d", d, idx)
	}

	d, idx = cfg.FindDomain("nonexistent")
	if d != nil || idx != -1 {
		t.Fatalf("FindDomain nonexistent: got %v at %d", d, idx)
	}

	if err := cfg.SetDomain("myapp", 4000); err != nil {
		t.Fatalf("SetDomain update: %v", err)
	}
	cfg, _ = Load()
	if cfg.Domains[0].Port != 4000 {
		t.Fatalf("expected port 4000, got %d", cfg.Domains[0].Port)
	}

	if err := cfg.SetDomain("api", 8080); err != nil {
		t.Fatalf("SetDomain second: %v", err)
	}
	cfg, _ = Load()
	if len(cfg.Domains) != 2 {
		t.Fatalf("expected 2 domains, got %d", len(cfg.Domains))
	}

	if err := cfg.RemoveDomain("myapp"); err != nil {
		t.Fatalf("RemoveDomain: %v", err)
	}
	cfg, _ = Load()
	if len(cfg.Domains) != 1 || cfg.Domains[0].Name != "api" {
		t.Fatalf("unexpected domains after remove: %+v", cfg.Domains)
	}

	if err := cfg.RemoveDomain("nonexistent"); err == nil {
		t.Fatal("expected error removing nonexistent domain")
	}
}

func TestLogMode(t *testing.T) {
	cfg := &Config{}
	if got := cfg.EffectiveLogMode(); got != LogModeFull {
		t.Fatalf("expected default log mode %q, got %q", LogModeFull, got)
	}

	valid := []string{"", "full", "minimal", "off", " Full "}
	for _, mode := range valid {
		if err := ValidateLogMode(mode); err != nil {
			t.Fatalf("ValidateLogMode(%q) error: %v", mode, err)
		}
	}

	if err := ValidateLogMode("verbose"); err == nil {
		t.Fatal("expected error for invalid log mode")
	}
}
