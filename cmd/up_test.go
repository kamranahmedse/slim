package cmd

import (
	"strings"
	"testing"

	"github.com/kamranahmedse/slim/internal/config"
	"github.com/kamranahmedse/slim/internal/daemon"
	"github.com/kamranahmedse/slim/internal/project"
)

func setupUpTestHooks(t *testing.T) func() {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := config.Init(); err != nil {
		t.Fatalf("config.Init: %v", err)
	}

	prevDiscover := upDiscoverFn
	prevFirstRun := upEnsureFirstRunFn
	prevWithLock := upWithLockFn
	prevLoad := upLoadFn
	prevAddHost := upAddHostFn
	prevLeafCert := upEnsureLeafCertFn
	prevRunning := upDaemonIsRunningFn
	prevIsChild := upDaemonIsChildFn
	prevNewPortFwd := upNewPortFwdFn
	prevPorts := upEnsurePortsFn
	prevDetached := upDaemonRunDetachedFn
	prevWait := upDaemonWaitFn
	prevIPC := upDaemonSendIPCFn

	upWithLockFn = config.WithLock
	upLoadFn = config.Load

	return func() {
		upDiscoverFn = prevDiscover
		upEnsureFirstRunFn = prevFirstRun
		upWithLockFn = prevWithLock
		upLoadFn = prevLoad
		upAddHostFn = prevAddHost
		upEnsureLeafCertFn = prevLeafCert
		upDaemonIsRunningFn = prevRunning
		upDaemonIsChildFn = prevIsChild
		upNewPortFwdFn = prevNewPortFwd
		upEnsurePortsFn = prevPorts
		upDaemonRunDetachedFn = prevDetached
		upDaemonWaitFn = prevWait
		upDaemonSendIPCFn = prevIPC
	}
}

func TestUpStartsDaemonForProjectServices(t *testing.T) {
	restore := setupUpTestHooks(t)
	defer restore()

	pc := &project.ProjectConfig{
		Services: []project.Service{
			{Domain: "myapp", Port: 3000},
			{Domain: "api", Port: 8080},
		},
	}

	upDiscoverFn = func() (*project.ProjectConfig, string, error) {
		return pc, "/tmp/.slim.yaml", nil
	}
	upEnsureFirstRunFn = func() error { return nil }
	upAddHostFn = func(string) error { return nil }
	upEnsureLeafCertFn = func(string) error { return nil }
	upDaemonIsRunningFn = func() bool { return false }
	upDaemonIsChildFn = func() bool { return true }
	upEnsurePortsFn = func() error { return nil }
	upDaemonRunDetachedFn = func() error { return nil }
	upDaemonWaitFn = func() error { return nil }

	err := upCmd.RunE(upCmd, nil)
	if err != nil {
		t.Fatalf("up: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Domains) != 2 {
		t.Fatalf("expected 2 domains, got %d", len(cfg.Domains))
	}
}

func TestUpUsesProjectBaseDomain(t *testing.T) {
	restore := setupUpTestHooks(t)
	defer restore()

	pc := &project.ProjectConfig{
		BaseDomain: "local.example.com",
		Services: []project.Service{
			{Domain: "myapp", Port: 3000},
		},
	}

	upDiscoverFn = func() (*project.ProjectConfig, string, error) {
		return pc, "/tmp/.slim.yaml", nil
	}
	upEnsureFirstRunFn = func() error { return nil }
	addedHosts := make([]string, 0, 1)
	upAddHostFn = func(host string) error {
		addedHosts = append(addedHosts, host)
		return nil
	}
	upEnsureLeafCertFn = func(string) error { return nil }
	upDaemonIsRunningFn = func() bool { return false }
	upDaemonIsChildFn = func() bool { return true }
	upEnsurePortsFn = func() error { return nil }
	upDaemonRunDetachedFn = func() error { return nil }
	upDaemonWaitFn = func() error { return nil }

	if err := upCmd.RunE(upCmd, nil); err != nil {
		t.Fatalf("up: %v", err)
	}

	if len(addedHosts) != 1 || addedHosts[0] != "myapp.local.example.com" {
		t.Fatalf("expected custom resolved hostname, got %v", addedHosts)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Domains) != 1 || cfg.Domains[0].ResolvedHostname() != "myapp.local.example.com" {
		t.Fatalf("unexpected stored domains: %+v", cfg.Domains)
	}
}

func TestUpServiceBaseDomainOverridesProject(t *testing.T) {
	restore := setupUpTestHooks(t)
	defer restore()

	pc := &project.ProjectConfig{
		BaseDomain: "local.example.com",
		Services: []project.Service{
			{Domain: "myapp", BaseDomain: "preview.example.com", Port: 3000},
		},
	}

	upDiscoverFn = func() (*project.ProjectConfig, string, error) {
		return pc, "/tmp/.slim.yaml", nil
	}
	upEnsureFirstRunFn = func() error { return nil }
	addedHosts := make([]string, 0, 1)
	upAddHostFn = func(host string) error {
		addedHosts = append(addedHosts, host)
		return nil
	}
	upEnsureLeafCertFn = func(string) error { return nil }
	upDaemonIsRunningFn = func() bool { return false }
	upDaemonIsChildFn = func() bool { return true }
	upEnsurePortsFn = func() error { return nil }
	upDaemonRunDetachedFn = func() error { return nil }
	upDaemonWaitFn = func() error { return nil }

	if err := upCmd.RunE(upCmd, nil); err != nil {
		t.Fatalf("up: %v", err)
	}

	if len(addedHosts) != 1 || addedHosts[0] != "myapp.preview.example.com" {
		t.Fatalf("expected service override hostname, got %v", addedHosts)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Domains) != 1 || cfg.Domains[0].Hostname != "preview.example.com" || cfg.Domains[0].ResolvedHostname() != "myapp.preview.example.com" {
		t.Fatalf("unexpected stored domains: %+v", cfg.Domains)
	}
}

func TestUpReloadsDaemonWhenRunning(t *testing.T) {
	restore := setupUpTestHooks(t)
	defer restore()

	pc := &project.ProjectConfig{
		Services: []project.Service{
			{Domain: "myapp", Port: 3000},
		},
	}

	upDiscoverFn = func() (*project.ProjectConfig, string, error) {
		return pc, "/tmp/.slim.yaml", nil
	}
	upEnsureFirstRunFn = func() error { return nil }
	upAddHostFn = func(string) error { return nil }
	upEnsureLeafCertFn = func(string) error { return nil }
	upDaemonIsChildFn = func() bool { return true }
	upDaemonIsRunningFn = func() bool { return true }

	var gotType daemon.MessageType
	upDaemonSendIPCFn = func(req daemon.Request) (*daemon.Response, error) {
		gotType = req.Type
		return &daemon.Response{OK: true}, nil
	}

	err := upCmd.RunE(upCmd, nil)
	if err != nil {
		t.Fatalf("up: %v", err)
	}

	if gotType != daemon.MsgReload {
		t.Fatalf("expected reload IPC, got %q", gotType)
	}
}

func TestUpValidationError(t *testing.T) {
	restore := setupUpTestHooks(t)
	defer restore()

	pc := &project.ProjectConfig{
		Services: []project.Service{
			{Domain: "myapp", Port: 3000},
			{Domain: "myapp", Port: 4000},
		},
	}

	upDiscoverFn = func() (*project.ProjectConfig, string, error) {
		return pc, "/tmp/.slim.yaml", nil
	}

	err := upCmd.RunE(upCmd, nil)
	if err == nil {
		t.Fatal("expected validation error for duplicate domains")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("unexpected error: %v", err)
	}
}
