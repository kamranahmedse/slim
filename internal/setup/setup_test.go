package setup

import "testing"

func TestEnsureProxyPortsAvailableIsNoop(t *testing.T) {
	if err := EnsureProxyPortsAvailable(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
