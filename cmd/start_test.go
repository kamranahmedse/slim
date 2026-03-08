package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/kamranahmedse/slim/internal/config"
	"github.com/kamranahmedse/slim/internal/daemon"
)

func setupStartTestHooks(t *testing.T) func() {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := config.Init(); err != nil {
		t.Fatalf("config.Init: %v", err)
	}

	prevEnsureFirstRun := startEnsureFirstRunFn
	prevWithLock := startWithLockFn
	prevAddHost := startAddHostFn
	prevLeafCert := startEnsureLeafCertFn
	prevIsChild := startDaemonIsChildFn
	prevIsRunning := startDaemonIsRunningFn
	prevNewPortFwd := startNewPortFwdFn
	prevEnsurePorts := startEnsurePortsFn
	prevRunDetached := startDaemonRunDetachedFn
	prevWait := startDaemonWaitFn
	prevSendIPC := startDaemonSendIPCFn

	prevPort := startPort
	prevDomain := startDomain
	prevLogMode := startLogMode
	prevCors := startCors
	prevWaitEnabled := startWait
	prevWaitTimeout := startWaitTimeout
	prevRoutes := append([]string(nil), startRoutes...)

	startWithLockFn = config.WithLock
	startPort = 0
	startDomain = ""
	startLogMode = ""
	startCors = false
	startWait = false
	startWaitTimeout = 30 * time.Second
	startRoutes = nil

	return func() {
		startEnsureFirstRunFn = prevEnsureFirstRun
		startWithLockFn = prevWithLock
		startAddHostFn = prevAddHost
		startEnsureLeafCertFn = prevLeafCert
		startDaemonIsChildFn = prevIsChild
		startDaemonIsRunningFn = prevIsRunning
		startNewPortFwdFn = prevNewPortFwd
		startEnsurePortsFn = prevEnsurePorts
		startDaemonRunDetachedFn = prevRunDetached
		startDaemonWaitFn = prevWait
		startDaemonSendIPCFn = prevSendIPC

		startPort = prevPort
		startDomain = prevDomain
		startLogMode = prevLogMode
		startCors = prevCors
		startWait = prevWaitEnabled
		startWaitTimeout = prevWaitTimeout
		startRoutes = prevRoutes
	}
}

func TestParseRouteFlags(t *testing.T) {
	tests := []struct {
		name    string
		flags   []string
		want    []config.Route
		wantErr string
	}{
		{
			name:  "empty",
			flags: nil,
			want:  nil,
		},
		{
			name:  "single route",
			flags: []string{"/api=8080"},
			want:  []config.Route{{Path: "/api", Port: 8080}},
		},
		{
			name:  "multiple routes",
			flags: []string{"/api=8080", "/ws=9000"},
			want:  []config.Route{{Path: "/api", Port: 8080}, {Path: "/ws", Port: 9000}},
		},
		{
			name:    "missing equals",
			flags:   []string{"/api8080"},
			wantErr: "expected path=port",
		},
		{
			name:    "invalid port",
			flags:   []string{"/api=notaport"},
			wantErr: "invalid route port",
		},
		{
			name:    "missing leading slash",
			flags:   []string{"api=8080"},
			wantErr: "must start with /",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRouteFlags(tt.flags)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("expected %d routes, got %d", len(tt.want), len(got))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("route[%d]: got %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestValidateStartWaitFlags(t *testing.T) {
	tests := []struct {
		name           string
		timeoutChanged bool
		wait           bool
		timeout        time.Duration
		wantErr        string
	}{
		{
			name:           "timeout without wait",
			timeoutChanged: true,
			wait:           false,
			timeout:        30 * time.Second,
			wantErr:        "--timeout requires --wait",
		},
		{
			name:           "wait with non-positive timeout",
			timeoutChanged: false,
			wait:           true,
			timeout:        0,
			wantErr:        "--timeout must be greater than 0",
		},
		{
			name:           "valid wait flags",
			timeoutChanged: true,
			wait:           true,
			timeout:        30 * time.Second,
		},
		{
			name:           "default no wait",
			timeoutChanged: false,
			wait:           false,
			timeout:        30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStartWaitFlags(tt.timeoutChanged, tt.wait, tt.timeout)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
			}
		})
	}
}

func TestStartUsesBaseDomainFlag(t *testing.T) {
	restore := setupStartTestHooks(t)
	defer restore()

	startPort = 3000
	startDomain = "local.example.com"

	startEnsureFirstRunFn = func() error { return nil }
	addedHosts := make([]string, 0, 1)
	startAddHostFn = func(host string) error {
		addedHosts = append(addedHosts, host)
		return nil
	}
	issuedCerts := make([]string, 0, 1)
	startEnsureLeafCertFn = func(host string) error {
		issuedCerts = append(issuedCerts, host)
		return nil
	}
	startDaemonIsChildFn = func() bool { return true }
	startDaemonIsRunningFn = func() bool { return true }

	var gotType daemon.MessageType
	startDaemonSendIPCFn = func(req daemon.Request) (*daemon.Response, error) {
		gotType = req.Type
		return &daemon.Response{OK: true}, nil
	}

	if err := startCmd.RunE(startCmd, []string{"myapp"}); err != nil {
		t.Fatalf("start: %v", err)
	}

	if gotType != daemon.MsgReload {
		t.Fatalf("expected reload IPC, got %q", gotType)
	}
	if len(addedHosts) != 1 || addedHosts[0] != "myapp.local.example.com" {
		t.Fatalf("expected resolved hostname host entry, got %v", addedHosts)
	}
	if len(issuedCerts) != 1 || issuedCerts[0] != "myapp.local.example.com" {
		t.Fatalf("expected resolved hostname certificate, got %v", issuedCerts)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Domains) != 1 {
		t.Fatalf("expected 1 stored domain, got %d", len(cfg.Domains))
	}
	if cfg.Domains[0].Name != "myapp" {
		t.Fatalf("expected stored name %q, got %q", "myapp", cfg.Domains[0].Name)
	}
	if cfg.Domains[0].Hostname != "local.example.com" {
		t.Fatalf("expected stored base domain %q, got %q", "local.example.com", cfg.Domains[0].Hostname)
	}
	if got := cfg.Domains[0].ResolvedHostname(); got != "myapp.local.example.com" {
		t.Fatalf("expected resolved hostname %q, got %q", "myapp.local.example.com", got)
	}
}

func TestStartRejectsInvalidBaseDomain(t *testing.T) {
	restore := setupStartTestHooks(t)
	defer restore()

	startPort = 3000
	startDomain = "local_example.com"

	err := startCmd.RunE(startCmd, []string{"myapp"})
	if err == nil {
		t.Fatal("expected invalid --domain error")
	}
	if !strings.Contains(err.Error(), "invalid --domain") {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg, loadErr := config.Load()
	if loadErr != nil {
		t.Fatalf("Load: %v", loadErr)
	}
	if len(cfg.Domains) != 0 {
		t.Fatalf("expected config to remain empty, got %+v", cfg.Domains)
	}
}
