package setup

import (
	"fmt"
	"net"

	"github.com/kamranahmedse/slim/internal/cert"
	"github.com/kamranahmedse/slim/internal/config"
	"github.com/kamranahmedse/slim/internal/system"
)

func EnsureFirstRun() error {
	if !cert.CAExists() {
		fmt.Print("First-time setup: generating root CA... ")
		if err := cert.GenerateCA(); err != nil {
			return err
		}
		fmt.Println("done")

		fmt.Println("Trusting root CA (you may be prompted for your password)...")
		if err := cert.TrustCA(); err != nil {
			return fmt.Errorf("trusting CA: %w", err)
		}
	}

	pf := system.NewPortForwarder()
	if !pf.IsEnabled() {
		fmt.Printf("Setting up port forwarding (80→%d, 443→%d)... ", config.ProxyHTTPPort, config.ProxyHTTPSPort)
		if err := pf.Enable(); err != nil {
			fmt.Printf("skipped (%v)\n", err)
		} else {
			fmt.Println("done")
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
