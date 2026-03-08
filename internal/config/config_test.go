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
		{"my.app", 3000, false},
		{"web.roadmap", 3000, false},
		{"a.b.c", 3000, false},
		{"my..app", 3000, true},
		{".myapp", 3000, true},
		{"myapp.", 3000, true},
		{"web.-bad", 3000, true},
		{"my app", 3000, true},
		{strings.Repeat("a", 63), 3000, false},
		{strings.Repeat("a", 64), 3000, true},
		{strings.Repeat("a", 63) + "." + strings.Repeat("b", 63), 3000, false},
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

	if err := cfg.SetDomain("myapp", 3000, nil); err != nil {
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
	if got := cfg.Domains[0].ResolvedHostname(); got != "myapp.test" {
		t.Fatalf("expected default hostname myapp.test, got %q", got)
	}

	d, idx := cfg.FindDomain("myapp")
	if d == nil || idx != 0 {
		t.Fatalf("FindDomain: got %v at %d", d, idx)
	}

	d, idx = cfg.FindDomain("nonexistent")
	if d != nil || idx != -1 {
		t.Fatalf("FindDomain nonexistent: got %v at %d", d, idx)
	}

	if err := cfg.SetDomain("myapp", 4000, nil); err != nil {
		t.Fatalf("SetDomain update: %v", err)
	}
	cfg, _ = Load()
	if cfg.Domains[0].Port != 4000 {
		t.Fatalf("expected port 4000, got %d", cfg.Domains[0].Port)
	}

	if err := cfg.SetDomain("api", 8080, nil); err != nil {
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

func TestValidateRoute(t *testing.T) {
	tests := []struct {
		path    string
		port    int
		wantErr bool
	}{
		{"/api", 8080, false},
		{"/", 3000, false},
		{"/api/v1", 9000, false},
		{"", 8080, true},
		{"api", 8080, true},
		{"/api", 0, true},
		{"/api", 65536, true},
	}

	for _, tt := range tests {
		err := ValidateRoute(tt.path, tt.port)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateRoute(%q, %d) error = %v, wantErr %v", tt.path, tt.port, err, tt.wantErr)
		}
	}
}

func TestMatchRoute(t *testing.T) {
	d := Domain{
		Name: "myapp",
		Port: 3000,
		Routes: []Route{
			{Path: "/api", Port: 8080},
			{Path: "/api/v2", Port: 9090},
			{Path: "/ws", Port: 9000},
		},
	}

	tests := []struct {
		reqPath  string
		wantPort int
	}{
		{"/", 3000},
		{"/about", 3000},
		{"/api", 8080},
		{"/api/users", 8080},
		{"/api/v2", 9090},
		{"/api/v2/items", 9090},
		{"/apikeys", 3000},
		{"/ws", 9000},
		{"/ws/chat", 9000},
	}

	for _, tt := range tests {
		got := d.MatchRoute(tt.reqPath)
		if got != tt.wantPort {
			t.Errorf("MatchRoute(%q) = %d, want %d", tt.reqPath, got, tt.wantPort)
		}
	}
}

func TestSetDomainWithRoutes(t *testing.T) {
	baseDir = t.TempDir()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	routes := []Route{{Path: "/api", Port: 8080}}
	if err := cfg.SetDomain("myapp", 3000, routes); err != nil {
		t.Fatalf("SetDomain with routes: %v", err)
	}

	cfg, _ = Load()
	if len(cfg.Domains[0].Routes) != 1 || cfg.Domains[0].Routes[0].Path != "/api" {
		t.Fatalf("unexpected routes: %+v", cfg.Domains[0].Routes)
	}

	if err := cfg.SetDomain("myapp", 3000, nil); err != nil {
		t.Fatalf("SetDomain clear routes: %v", err)
	}

	cfg, _ = Load()
	if len(cfg.Domains[0].Routes) != 0 {
		t.Fatalf("expected routes to be cleared, got %+v", cfg.Domains[0].Routes)
	}
}

func TestSetDomainWithBaseDomain(t *testing.T) {
	baseDir = t.TempDir()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if err := cfg.SetDomainHostname("myapp", "local.example.com", 3000, nil); err != nil {
		t.Fatalf("SetDomainHostname: %v", err)
	}

	cfg, _ = Load()
	if got := cfg.Domains[0].Hostname; got != "local.example.com" {
		t.Fatalf("expected stored base domain, got %q", got)
	}
	if got := cfg.Domains[0].ResolvedHostname(); got != "myapp.local.example.com" {
		t.Fatalf("expected resolved hostname, got %q", got)
	}

	d, idx := cfg.FindDomain("myapp.local.example.com")
	if d == nil || idx != 0 {
		t.Fatalf("FindDomain by hostname: got %v at %d", d, idx)
	}
}

func TestFindDomainByResolvedHostname(t *testing.T) {
	baseDir = t.TempDir()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Simple domain: findable by name and by resolved hostname
	if err := cfg.SetDomain("myapp", 3000, nil); err != nil {
		t.Fatalf("SetDomain: %v", err)
	}
	cfg, _ = Load()

	d, idx := cfg.FindDomain("myapp")
	if d == nil || idx != 0 {
		t.Fatalf("FindDomain by name: got %v at %d", d, idx)
	}

	d, idx = cfg.FindDomain("myapp.test")
	if d == nil || idx != 0 {
		t.Fatalf("FindDomain by resolved hostname: got %v at %d", d, idx)
	}

	// Custom base domain: findable by name and by resolved hostname
	if err := cfg.SetDomainHostname("api", "local.example.com", 8080, nil); err != nil {
		t.Fatalf("SetDomainHostname: %v", err)
	}
	cfg, _ = Load()

	d, idx = cfg.FindDomain("api")
	if d == nil || idx != 1 {
		t.Fatalf("FindDomain custom by name: got %v at %d", d, idx)
	}

	d, idx = cfg.FindDomain("api.local.example.com")
	if d == nil || idx != 1 {
		t.Fatalf("FindDomain custom by hostname: got %v at %d", d, idx)
	}

	// Non-existent
	d, idx = cfg.FindDomain("nope.example.com")
	if d != nil || idx != -1 {
		t.Fatalf("FindDomain nonexistent: got %v at %d", d, idx)
	}
}

func TestRemoveDomainByHostname(t *testing.T) {
	baseDir = t.TempDir()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if err := cfg.SetDomainHostname("api", "local.example.com", 8080, nil); err != nil {
		t.Fatalf("SetDomainHostname: %v", err)
	}
	cfg, _ = Load()

	// Remove by hostname should work since FindDomain matches on ResolvedHostname
	if err := cfg.RemoveDomain("api.local.example.com"); err != nil {
		t.Fatalf("RemoveDomain by hostname: %v", err)
	}
	cfg, _ = Load()
	if len(cfg.Domains) != 0 {
		t.Fatalf("expected 0 domains after remove, got %d", len(cfg.Domains))
	}
}

func TestResolveHostname(t *testing.T) {
	tests := []struct {
		name       string
		baseDomain string
		want       string
	}{
		{name: "myapp", want: "myapp.test"},
		{name: "myapp", baseDomain: "local.example.com", want: "myapp.local.example.com"},
		{name: "myapp", baseDomain: ".local.example.com.", want: "myapp.local.example.com"},
	}

	for _, tt := range tests {
		if got := ResolveHostname(tt.name, tt.baseDomain); got != tt.want {
			t.Fatalf("ResolveHostname(%q, %q) = %q, want %q", tt.name, tt.baseDomain, got, tt.want)
		}
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
