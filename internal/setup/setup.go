package setup

import (
	"fmt"
	"net"

	"github.com/kamranahmedse/slim/internal/cert"
	"github.com/kamranahmedse/slim/internal/config"
	"github.com/kamranahmedse/slim/internal/system"
	"github.com/kamranahmedse/slim/internal/term"
)

func EnsureFirstRun() error {
	if !cert.CAExists() {
		err := term.RunSteps([]term.Step{
			{
				Name: "Generating root CA",
				Run: func() (string, error) {
					return "done", cert.GenerateCA()
				},
			},
			{
				Name:        "Trusting root CA (you may be prompted for your password)",
				Interactive: true,
				Run: func() (string, error) {
					return "done", cert.TrustCA()
				},
			},
		})
		if err != nil {
			return err
		}
	}

	pf := system.NewPortForwarder()
	if !pf.IsEnabled() {
		err := term.RunSteps([]term.Step{
			{
				Name: fmt.Sprintf("Setting up port forwarding (80→%d, 443→%d)", config.ProxyHTTPPort, config.ProxyHTTPSPort),
				Run: func() (string, error) {
					if err := pf.Enable(); err != nil {
						return fmt.Sprintf("skipped (%v)", err), nil
					}
					return "done", nil
				},
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func EnsureProxyPortsAvailable() error {
	addrs := []string{
		fmt.Sprintf(":%d", config.ProxyHTTPPort),
		fmt.Sprintf(":%d", config.ProxyHTTPSPort),
	}
	for _, addr := range addrs {
		if err := ensurePortAvailable(addr); err != nil {
			return err
		}
	}
	return nil
}

func ensurePortAvailable(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("proxy listener port %s is unavailable: %w (another local proxy/old daemon may already be running)", addr, err)
	}
	_ = ln.Close()
	return nil
}
