package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/kamranahmedse/slim/internal/cert"
	"github.com/kamranahmedse/slim/internal/config"
	"github.com/kamranahmedse/slim/internal/daemon"
	"github.com/kamranahmedse/slim/internal/proxy"
	"github.com/kamranahmedse/slim/internal/setup"
	"github.com/kamranahmedse/slim/internal/system"
	"github.com/spf13/cobra"
)

var startPort int
var startLogMode string
var startWait bool
var startWaitTimeout time.Duration

var startCmd = &cobra.Command{
	Use:   "start [name] --port [port]",
	Short: "Start proxying a domain",
	Long: `Map a .local domain to a local port and start proxying.
Runs first-time setup automatically if needed.

  slim start myapp --port 3000
  # https://myapp.local → localhost:3000`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := normalizeName(args[0])

		if err := config.ValidateDomain(name, startPort); err != nil {
			return err
		}
		if err := validateStartWaitFlags(cmd.Flags().Changed("timeout"), startWait, startWaitTimeout); err != nil {
			return err
		}
		if startLogMode != "" {
			if err := config.ValidateLogMode(startLogMode); err != nil {
				return err
			}
		}

		if err := setup.EnsureFirstRun(); err != nil {
			return err
		}

		if err := config.WithLock(func() error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if startLogMode != "" {
				cfg.LogMode = strings.ToLower(strings.TrimSpace(startLogMode))
			}
			return cfg.SetDomain(name, startPort)
		}); err != nil {
			return err
		}

		if err := system.AddHost(name); err != nil {
			return fmt.Errorf("updating /etc/hosts: %w", err)
		}

		if err := cert.EnsureLeafCert(name); err != nil {
			return fmt.Errorf("generating certificate: %w", err)
		}

		if !daemon.IsRunning() {
			if err := setup.EnsureProxyPortsAvailable(); err != nil {
				return err
			}
			if err := daemon.RunDetached(); err != nil {
				return fmt.Errorf("starting daemon: %w", err)
			}
			if err := daemon.WaitForDaemon(); err != nil {
				return err
			}
		} else {
			if _, err := daemon.SendIPC(daemon.Request{Type: daemon.MsgReload}); err != nil {
				return fmt.Errorf("reloading daemon: %w", err)
			}
		}

		if startWait {
			fmt.Printf("Waiting for localhost:%d (timeout %s)... ", startPort, startWaitTimeout)
			if err := proxy.WaitForUpstream(startPort, startWaitTimeout); err != nil {
				fmt.Println("timed out")
				return err
			}
			fmt.Println("ready")
		}

		fmt.Printf("https://%s.local → localhost:%d\n", name, startPort)
		return nil
	},
}

func validateStartWaitFlags(timeoutChanged bool, wait bool, timeout time.Duration) error {
	if timeoutChanged && !wait {
		return fmt.Errorf("--timeout requires --wait")
	}
	if wait && timeout <= 0 {
		return fmt.Errorf("--timeout must be greater than 0")
	}
	return nil
}

func init() {
	startCmd.Flags().IntVarP(&startPort, "port", "p", 0, "Local port to proxy to (required)")
	startCmd.Flags().StringVar(&startLogMode, "log-mode", "", "Access log mode: full|minimal|off")
	startCmd.Flags().BoolVar(&startWait, "wait", false, "Wait for the upstream app to become reachable before returning")
	startCmd.Flags().DurationVar(&startWaitTimeout, "timeout", 30*time.Second, "Maximum time to wait for upstream with --wait")
	startCmd.MarkFlagRequired("port")
	rootCmd.AddCommand(startCmd)
}
