package httperr

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"
)

func TestNetworkHint_DNS(t *testing.T) {
	err := &net.DNSError{Err: "no such host", Name: "app.slim.sh"}
	hint := NetworkHint(err)
	if !strings.Contains(hint, "check your internet connection") {
		t.Fatalf("got %q", hint)
	}
}

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "i/o timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

func TestNetworkHint_Timeout(t *testing.T) {
	err := fmt.Errorf("dial tcp: %w", timeoutErr{})
	hint := NetworkHint(err)
	if !strings.Contains(hint, "timed out") {
		t.Fatalf("got %q", hint)
	}
}

func TestNetworkHint_ConnectionRefused(t *testing.T) {
	err := fmt.Errorf("dial tcp 127.0.0.1:443: connection refused")
	hint := NetworkHint(err)
	if !strings.Contains(hint, "server may be down") {
		t.Fatalf("got %q", hint)
	}
}

func TestNetworkHint_Unreachable(t *testing.T) {
	err := fmt.Errorf("dial tcp: network is unreachable")
	hint := NetworkHint(err)
	if !strings.Contains(hint, "check your internet connection") {
		t.Fatalf("got %q", hint)
	}
}

func TestNetworkHint_GenericError(t *testing.T) {
	err := fmt.Errorf("something unexpected")
	hint := NetworkHint(err)
	if hint != "something unexpected" {
		t.Fatalf("got %q", hint)
	}
}

func TestNetworkHint_Nil(t *testing.T) {
	hint := NetworkHint(nil)
	if hint != "" {
		t.Fatalf("expected empty, got %q", hint)
	}
}

func TestWrap_NetworkError(t *testing.T) {
	inner := &net.DNSError{Err: "no such host", Name: "app.slim.sh"}
	err := Wrap("login failed", inner)
	if !strings.Contains(err.Error(), "login failed") {
		t.Fatalf("missing context: %q", err)
	}
	if !strings.Contains(err.Error(), "check your internet connection") {
		t.Fatalf("missing hint: %q", err)
	}
}

func TestWrap_NonNetworkError(t *testing.T) {
	inner := fmt.Errorf("parse error")
	err := Wrap("login failed", inner)
	if err.Error() != "login failed: parse error" {
		t.Fatalf("got %q", err)
	}
	if !errors.Is(err, inner) {
		t.Fatal("expected wrapped error to be unwrappable")
	}
}

func TestWrap_Nil(t *testing.T) {
	err := Wrap("context", nil)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}
