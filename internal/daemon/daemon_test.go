package daemon

import (
	"encoding/json"
	"net"
	"os"
	"testing"

	"github.com/kamranahmedse/slim/internal/config"
	"github.com/kamranahmedse/slim/internal/proxy"
)

func TestHandleIPCUnknownMessage(t *testing.T) {
	resp := handleIPC(Request{Type: MessageType("unknown")}, &proxy.Server{}, newMDNSResponder())
	if resp.OK {
		t.Fatalf("expected failure for unknown message, got %+v", resp)
	}
	if resp.Error == "" {
		t.Fatalf("expected error message for unknown message, got %+v", resp)
	}
}

func TestHandleStatusIncludesDomainHealth(t *testing.T) {
	initDaemonStateTestConfig(t)

	healthyLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen healthy: %v", err)
	}
	defer healthyLn.Close()
	healthyPort := healthyLn.Addr().(*net.TCPAddr).Port

	unhealthyLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen unhealthy: %v", err)
	}
	unhealthyPort := unhealthyLn.Addr().(*net.TCPAddr).Port
	_ = unhealthyLn.Close()

	cfg := &config.Config{
		Domains: []config.Domain{
			{Name: "healthy", Port: healthyPort},
			{Name: "unhealthy", Port: unhealthyPort},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save config: %v", err)
	}

	resp := handleStatus()
	if !resp.OK {
		t.Fatalf("expected OK status response, got %+v", resp)
	}

	var status StatusData
	if err := json.Unmarshal(resp.Data, &status); err != nil {
		t.Fatalf("Unmarshal status: %v", err)
	}
	if !status.Running {
		t.Fatalf("expected running=true, got %+v", status)
	}
	if len(status.Domains) != 2 {
		t.Fatalf("expected 2 domains, got %+v", status.Domains)
	}

	healthByName := map[string]bool{}
	for _, d := range status.Domains {
		healthByName[d.Name] = d.Healthy
	}

	if !healthByName["healthy"] {
		t.Fatalf("expected healthy domain to be reachable, got %+v", status.Domains)
	}
	if healthByName["unhealthy"] {
		t.Fatalf("expected unhealthy domain to be unreachable, got %+v", status.Domains)
	}
}

func TestIsRunningFalseForStaleSocketPath(t *testing.T) {
	initDaemonStateTestConfig(t)

	if err := os.WriteFile(config.SocketPath(), []byte("not-a-socket"), 0644); err != nil {
		t.Fatalf("WriteFile stale socket path: %v", err)
	}

	if IsRunning() {
		t.Fatal("expected IsRunning to be false for stale non-socket path")
	}
}

func initDaemonStateTestConfig(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := config.Init(); err != nil {
		t.Fatalf("config.Init: %v", err)
	}
	if err := os.MkdirAll(config.Dir(), 0755); err != nil {
		t.Fatalf("MkdirAll config dir: %v", err)
	}
}
