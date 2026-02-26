package proxy

import (
	"net"
	"strconv"
	"testing"
	"time"
)

func TestWaitForUpstreamReadyImmediately(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	if err := WaitForUpstream(port, 500*time.Millisecond); err != nil {
		t.Fatalf("WaitForUpstream unexpected error: %v", err)
	}
}

func TestWaitForUpstreamBecomesReady(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	go func() {
		time.Sleep(250 * time.Millisecond)
		readyLn, listenErr := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
		if listenErr != nil {
			return
		}
		defer readyLn.Close()
		time.Sleep(250 * time.Millisecond)
	}()

	if err := WaitForUpstream(port, 2*time.Second); err != nil {
		t.Fatalf("WaitForUpstream unexpected error: %v", err)
	}
}

func TestWaitForUpstreamTimeout(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	if err := WaitForUpstream(port, 300*time.Millisecond); err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestWaitForUpstreamInvalidTimeout(t *testing.T) {
	if err := WaitForUpstream(3000, 0); err == nil {
		t.Fatal("expected error for invalid timeout")
	}
}
