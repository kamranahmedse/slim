package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/kamranahmedse/slim/internal/config"
	"github.com/kamranahmedse/slim/internal/daemon"
	"github.com/kamranahmedse/slim/internal/system"
)

func TestStopOneDomainNotFound(t *testing.T) {
	restore := setupStopTestHooks(t)
	defer restore()

	if err := seedDomains([]config.Domain{{Name: "api", Port: 8080}}); err != nil {
		t.Fatalf("seedDomains: %v", err)
	}

	err := stopOne("myapp")
	if err == nil {
		t.Fatal("expected stopOne to fail for missing domain")
	}
	if !strings.Contains(err.Error(), "is not running") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStopOneSendsShutdownWhenLastDomain(t *testing.T) {
	restore := setupStopTestHooks(t)
	defer restore()

	if err := seedDomains([]config.Domain{{Name: "myapp", Port: 3000}}); err != nil {
		t.Fatalf("seedDomains: %v", err)
	}

	systemRemoveHostFn = func(string) error { return nil }
	daemonIsRunningFn = func() bool { return true }

	var gotType daemon.MessageType
	daemonSendIPCFn = func(req daemon.Request) (*daemon.Response, error) {
		gotType = req.Type
		return &daemon.Response{OK: true}, nil
	}

	if err := stopOne("myapp"); err != nil {
		t.Fatalf("stopOne: %v", err)
	}

	if gotType != daemon.MsgShutdown {
		t.Fatalf("expected IPC type %q, got %q", daemon.MsgShutdown, gotType)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Domains) != 0 {
		t.Fatalf("expected no remaining domains, got %+v", cfg.Domains)
	}
}

func TestStopOneSendsReloadWhenDomainsRemain(t *testing.T) {
	restore := setupStopTestHooks(t)
	defer restore()

	if err := seedDomains([]config.Domain{
		{Name: "myapp", Port: 3000},
		{Name: "api", Port: 8080},
	}); err != nil {
		t.Fatalf("seedDomains: %v", err)
	}

	systemRemoveHostFn = func(string) error { return nil }
	daemonIsRunningFn = func() bool { return true }

	var gotType daemon.MessageType
	daemonSendIPCFn = func(req daemon.Request) (*daemon.Response, error) {
		gotType = req.Type
		return &daemon.Response{OK: true}, nil
	}

	if err := stopOne("myapp"); err != nil {
		t.Fatalf("stopOne: %v", err)
	}
	if gotType != daemon.MsgReload {
		t.Fatalf("expected IPC type %q, got %q", daemon.MsgReload, gotType)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Domains) != 1 || cfg.Domains[0].Name != "api" {
		t.Fatalf("unexpected remaining domains: %+v", cfg.Domains)
	}
}

func TestStopAllNoDomainsNoDaemon(t *testing.T) {
	restore := setupStopTestHooks(t)
	defer restore()

	if err := seedDomains(nil); err != nil {
		t.Fatalf("seedDomains: %v", err)
	}
	daemonIsRunningFn = func() bool { return false }

	if err := stopAll(); err != nil {
		t.Fatalf("stopAll: %v", err)
	}
}

func TestStopAllRemovesHostsAndSendsShutdown(t *testing.T) {
	restore := setupStopTestHooks(t)
	defer restore()

	if err := seedDomains([]config.Domain{
		{Name: "myapp", Port: 3000},
		{Name: "api", Port: 8080},
	}); err != nil {
		t.Fatalf("seedDomains: %v", err)
	}

	removed := make([]string, 0, 2)
	systemRemoveHostFn = func(name string) error {
		removed = append(removed, name)
		if name == "myapp" {
			return errors.New("remove failed")
		}
		return nil
	}

	daemonIsRunningFn = func() bool { return true }
	var sendCalls int
	daemonSendIPCFn = func(req daemon.Request) (*daemon.Response, error) {
		sendCalls++
		if req.Type != daemon.MsgShutdown {
			t.Fatalf("expected shutdown IPC type, got %q", req.Type)
		}
		return &daemon.Response{OK: true}, nil
	}

	if err := stopAll(); err != nil {
		t.Fatalf("stopAll: %v", err)
	}
	if sendCalls != 1 {
		t.Fatalf("expected one shutdown IPC call, got %d", sendCalls)
	}
	if len(removed) != 2 {
		t.Fatalf("expected host removals for all domains, got %v", removed)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Domains) != 0 {
		t.Fatalf("expected domains to be cleared, got %+v", cfg.Domains)
	}
}

func TestStopAllDaemonShutdownError(t *testing.T) {
	restore := setupStopTestHooks(t)
	defer restore()

	if err := seedDomains([]config.Domain{{Name: "myapp", Port: 3000}}); err != nil {
		t.Fatalf("seedDomains: %v", err)
	}

	systemRemoveHostFn = func(string) error { return nil }
	daemonIsRunningFn = func() bool { return true }
	daemonSendIPCFn = func(req daemon.Request) (*daemon.Response, error) {
		return nil, errors.New("ipc down")
	}

	err := stopAll()
	if err == nil {
		t.Fatal("expected stopAll to fail when daemon shutdown IPC fails")
	}
	if !strings.Contains(err.Error(), "stopping daemon") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func setupStopTestHooks(t *testing.T) func() {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := config.Init(); err != nil {
		t.Fatalf("config.Init: %v", err)
	}

	prevWithLock := configWithLockStopFn
	prevLoad := configLoadStopFn
	prevRemoveHost := systemRemoveHostFn
	prevIsRunning := daemonIsRunningFn
	prevSendIPC := daemonSendIPCFn

	configWithLockStopFn = config.WithLock
	configLoadStopFn = config.Load
	systemRemoveHostFn = system.RemoveHost
	daemonIsRunningFn = daemon.IsRunning
	daemonSendIPCFn = daemon.SendIPC

	return func() {
		configWithLockStopFn = prevWithLock
		configLoadStopFn = prevLoad
		systemRemoveHostFn = prevRemoveHost
		daemonIsRunningFn = prevIsRunning
		daemonSendIPCFn = prevSendIPC
	}
}

func seedDomains(domains []config.Domain) error {
	cfg := &config.Config{
		Domains: domains,
	}
	return cfg.Save()
}
