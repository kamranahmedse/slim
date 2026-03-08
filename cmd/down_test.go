package cmd

import (
	"testing"

	"github.com/kamranahmedse/slim/internal/config"
	"github.com/kamranahmedse/slim/internal/daemon"
	"github.com/kamranahmedse/slim/internal/project"
)

func setupDownTestHooks(t *testing.T) func() {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := config.Init(); err != nil {
		t.Fatalf("config.Init: %v", err)
	}

	prevDiscover := downDiscoverFn
	prevWithLock := downWithLockFn
	prevLoad := downLoadFn
	prevRemove := downRemoveHostFn
	prevRunning := downDaemonRunningFn
	prevIPC := downDaemonSendIPCFn

	downWithLockFn = config.WithLock
	downLoadFn = config.Load

	return func() {
		downDiscoverFn = prevDiscover
		downWithLockFn = prevWithLock
		downLoadFn = prevLoad
		downRemoveHostFn = prevRemove
		downDaemonRunningFn = prevRunning
		downDaemonSendIPCFn = prevIPC
	}
}

func TestDownRemovesProjectServices(t *testing.T) {
	restore := setupDownTestHooks(t)
	defer restore()

	if err := seedDomains([]config.Domain{
		{Name: "myapp", Port: 3000},
		{Name: "api", Port: 8080},
		{Name: "other", Port: 9000},
	}); err != nil {
		t.Fatalf("seedDomains: %v", err)
	}

	pc := &project.ProjectConfig{
		Services: []project.Service{
			{Domain: "myapp", Port: 3000},
			{Domain: "api", Port: 8080},
		},
	}

	downDiscoverFn = func() (*project.ProjectConfig, string, error) {
		return pc, "/tmp/.slim.yaml", nil
	}
	downRemoveHostFn = func(string) error { return nil }
	downDaemonRunningFn = func() bool { return true }

	var gotType daemon.MessageType
	downDaemonSendIPCFn = func(req daemon.Request) (*daemon.Response, error) {
		gotType = req.Type
		return &daemon.Response{OK: true}, nil
	}

	err := downCmd.RunE(downCmd, nil)
	if err != nil {
		t.Fatalf("down: %v", err)
	}

	if gotType != daemon.MsgReload {
		t.Fatalf("expected reload IPC (other domain remains), got %q", gotType)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Domains) != 1 || cfg.Domains[0].Name != "other" {
		t.Fatalf("expected only 'other' to remain, got %+v", cfg.Domains)
	}
}

func TestDownShutdownsWhenNoDomains(t *testing.T) {
	restore := setupDownTestHooks(t)
	defer restore()

	if err := seedDomains([]config.Domain{
		{Name: "myapp", Port: 3000},
	}); err != nil {
		t.Fatalf("seedDomains: %v", err)
	}

	pc := &project.ProjectConfig{
		Services: []project.Service{
			{Domain: "myapp", Port: 3000},
		},
	}

	downDiscoverFn = func() (*project.ProjectConfig, string, error) {
		return pc, "/tmp/.slim.yaml", nil
	}
	downRemoveHostFn = func(string) error { return nil }
	downDaemonRunningFn = func() bool { return true }

	var gotType daemon.MessageType
	downDaemonSendIPCFn = func(req daemon.Request) (*daemon.Response, error) {
		gotType = req.Type
		return &daemon.Response{OK: true}, nil
	}

	err := downCmd.RunE(downCmd, nil)
	if err != nil {
		t.Fatalf("down: %v", err)
	}

	if gotType != daemon.MsgShutdown {
		t.Fatalf("expected shutdown IPC, got %q", gotType)
	}
}

func TestDownRemovesProjectBaseDomainHosts(t *testing.T) {
	restore := setupDownTestHooks(t)
	defer restore()

	if err := seedDomains([]config.Domain{
		{Name: "myapp", Hostname: "local.example.com", Port: 3000},
		{Name: "other", Port: 9000},
	}); err != nil {
		t.Fatalf("seedDomains: %v", err)
	}

	pc := &project.ProjectConfig{
		BaseDomain: "local.example.com",
		Services: []project.Service{
			{Domain: "myapp", Port: 3000},
		},
	}

	downDiscoverFn = func() (*project.ProjectConfig, string, error) {
		return pc, "/tmp/.slim.yaml", nil
	}
	removed := make([]string, 0, 1)
	downRemoveHostFn = func(host string) error {
		removed = append(removed, host)
		return nil
	}
	downDaemonRunningFn = func() bool { return false }

	if err := downCmd.RunE(downCmd, nil); err != nil {
		t.Fatalf("down: %v", err)
	}

	if len(removed) != 1 || removed[0] != "myapp.local.example.com" {
		t.Fatalf("expected resolved hostname removal, got %v", removed)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Domains) != 1 || cfg.Domains[0].Name != "other" {
		t.Fatalf("unexpected remaining domains: %+v", cfg.Domains)
	}
}

func TestDownServiceBaseDomainOverridesProject(t *testing.T) {
	restore := setupDownTestHooks(t)
	defer restore()

	if err := seedDomains([]config.Domain{
		{Name: "myapp", Hostname: "preview.example.com", Port: 3000},
		{Name: "other", Port: 9000},
	}); err != nil {
		t.Fatalf("seedDomains: %v", err)
	}

	pc := &project.ProjectConfig{
		BaseDomain: "local.example.com",
		Services: []project.Service{
			{Domain: "myapp", BaseDomain: "preview.example.com", Port: 3000},
		},
	}

	downDiscoverFn = func() (*project.ProjectConfig, string, error) {
		return pc, "/tmp/.slim.yaml", nil
	}
	removed := make([]string, 0, 1)
	downRemoveHostFn = func(host string) error {
		removed = append(removed, host)
		return nil
	}
	downDaemonRunningFn = func() bool { return false }

	if err := downCmd.RunE(downCmd, nil); err != nil {
		t.Fatalf("down: %v", err)
	}

	if len(removed) != 1 || removed[0] != "myapp.preview.example.com" {
		t.Fatalf("expected service override hostname removal, got %v", removed)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Domains) != 1 || cfg.Domains[0].Name != "other" {
		t.Fatalf("unexpected remaining domains: %+v", cfg.Domains)
	}
}
