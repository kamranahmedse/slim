package setup

import (
	"net"
	"strings"
	"testing"
)

func TestEnsureProxyPortsAvailableFailsWhenInUse(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	err = ensurePortAvailable(ln.Addr().String())
	if err == nil {
		t.Fatal("expected error for an in-use port")
	}
	if !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("expected unavailable error, got: %v", err)
	}
}

func TestEnsurePortAvailableSuccess(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	if err := ensurePortAvailable(addr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
