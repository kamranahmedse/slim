package cmd

import (
	"net"
	"strings"
	"testing"
)

func TestEnsurePortsAvailableSuccess(t *testing.T) {
	ln1, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen 1: %v", err)
	}
	addr1 := ln1.Addr().String()
	_ = ln1.Close()

	ln2, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen 2: %v", err)
	}
	addr2 := ln2.Addr().String()
	_ = ln2.Close()

	if err := ensurePortsAvailable([]string{addr1, addr2}); err != nil {
		t.Fatalf("ensurePortsAvailable unexpected error: %v", err)
	}
}

func TestEnsurePortsAvailableFailure(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	err = ensurePortsAvailable([]string{ln.Addr().String()})
	if err == nil {
		t.Fatal("expected ensurePortsAvailable to fail for an in-use port")
	}
	if !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("expected unavailable error, got: %v", err)
	}
}
