package daemon

import (
	"encoding/json"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kamranahmedse/slim/internal/config"
)

func TestIPCServerRoundTrip(t *testing.T) {
	initDaemonTestConfig(t)

	srv, err := NewIPCServer(func(req Request) Response {
		if req.Type != MsgStatus {
			return Response{OK: false, Error: "unexpected request"}
		}
		return Response{OK: true, Data: json.RawMessage(`{"ok":true}`)}
	})
	if err != nil {
		t.Fatalf("NewIPCServer: %v", err)
	}
	defer srv.Close()
	go srv.Serve()

	resp, err := SendIPC(Request{Type: MsgStatus})
	if err != nil {
		t.Fatalf("SendIPC: %v", err)
	}
	if !resp.OK {
		t.Fatalf("expected OK response, got %+v", resp)
	}
	if string(resp.Data) != `{"ok":true}` {
		t.Fatalf("unexpected response payload: %s", string(resp.Data))
	}
}

func TestIPCServerReturnsErrorOnInvalidJSON(t *testing.T) {
	initDaemonTestConfig(t)

	srv, err := NewIPCServer(func(req Request) Response {
		return Response{OK: true}
	})
	if err != nil {
		t.Fatalf("NewIPCServer: %v", err)
	}
	defer srv.Close()
	go srv.Serve()

	conn, err := net.Dial("unix", config.SocketPath())
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := conn.Write([]byte("not json\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}

	var resp Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		t.Fatalf("Decode response: %v", err)
	}
	if resp.OK {
		t.Fatalf("expected error response, got %+v", resp)
	}
	if resp.Error == "" {
		t.Fatalf("expected decode error message, got %+v", resp)
	}
}

func TestSendIPCWhenDaemonNotRunning(t *testing.T) {
	initDaemonTestConfig(t)
	_ = os.Remove(config.SocketPath())

	_, err := SendIPC(Request{Type: MsgStatus})
	if err == nil {
		t.Fatal("expected SendIPC to fail when socket is missing")
	}
	if !strings.Contains(err.Error(), "is slim running?") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIPCServerCloseRemovesSocket(t *testing.T) {
	initDaemonTestConfig(t)

	srv, err := NewIPCServer(func(req Request) Response { return Response{OK: true} })
	if err != nil {
		t.Fatalf("NewIPCServer: %v", err)
	}
	sockPath := config.SocketPath()
	if _, err := os.Stat(sockPath); err != nil {
		t.Fatalf("expected socket file to exist: %v", err)
	}

	srv.Close()
	if _, err := os.Stat(sockPath); !os.IsNotExist(err) {
		t.Fatalf("expected socket to be removed on close, got err=%v", err)
	}
}

func initDaemonTestConfig(t *testing.T) {
	t.Helper()
	home, err := os.MkdirTemp("", "slim-daemon-test-")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(home)
	})
	t.Setenv("HOME", home)
	if err := config.Init(); err != nil {
		t.Fatalf("config.Init: %v", err)
	}
}
