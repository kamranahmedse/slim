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
