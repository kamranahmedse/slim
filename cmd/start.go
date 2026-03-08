package cmd

import (
	"fmt"
	"strconv"
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
var startDomain string
var startLogMode string
var startCors bool
var startWait bool
var startWaitTimeout time.Duration
var startRoutes []string

var (
	startEnsureFirstRunFn    = setup.EnsureFirstRun
	startWithLockFn          = config.WithLock
	startAddHostFn           = system.AddHost
	startEnsureLeafCertFn    = cert.EnsureLeafCert
	startDaemonIsChildFn     = daemon.IsChild
	startDaemonIsRunningFn   = daemon.IsRunning
	startNewPortFwdFn        = system.NewPortForwarder
	startEnsurePortsFn       = setup.EnsureProxyPortsAvailable
	startDaemonRunDetachedFn = daemon.RunDetached
	startDaemonWaitFn        = daemon.WaitForDaemon
	startDaemonSendIPCFn     = daemon.SendIPC
)

var startCmd = &cobra.Command{
	Use:   "start [name] --port [port]",
	Short: "Start proxying a domain",
	Long: `Map a local domain to a local port and start proxying.
Runs first-time setup automatically if needed.

  slim start myapp --port 3000
  # https://myapp.test → localhost:3000
  slim start myapp --port 3000 --domain local.example.com`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := normalizeName(args[0])
		baseDomain := config.NormalizeHostname(startDomain)

		if err := config.ValidateDomain(name, startPort); err != nil {
			return err
		}
		if baseDomain != "" {
			if err := config.ValidateDomainName(baseDomain); err != nil {
				return fmt.Errorf("invalid --domain: %w", err)
			}
		}
		if err := validateStartWaitFlags(cmd.Flags().Changed("timeout"), startWait, startWaitTimeout); err != nil {
			return err
		}
		if startLogMode != "" {
			if err := config.ValidateLogMode(startLogMode); err != nil {
				return err
			}
		}

		routes, err := parseRouteFlags(startRoutes)
		if err != nil {
			return err
		}

		if err := startEnsureFirstRunFn(); err != nil {
			return err
		}

		if err := startWithLockFn(func() error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("cors") {
				cfg.Cors = startCors
			}
			if startLogMode != "" {
				cfg.LogMode = strings.ToLower(strings.TrimSpace(startLogMode))
			}
			return cfg.SetDomainHostname(name, baseDomain, startPort, routes)
		}); err != nil {
			return err
		}

		domain := config.Domain{Name: name, Hostname: baseDomain, Port: startPort, Routes: routes}

		if err := startAddHostFn(domain.ResolvedHostname()); err != nil {
			return fmt.Errorf("updating /etc/hosts: %w", err)
		}

		if err := startEnsureLeafCertFn(domain.ResolvedHostname()); err != nil {
			return fmt.Errorf("generating certificate: %w", err)
		}

		if !startDaemonIsChildFn() {
			pf := startNewPortFwdFn()
			if shouldReloadPortForwarding(pf, startDaemonIsRunningFn()) {
				if err := pf.EnsureLoaded(); err != nil {
					return fmt.Errorf("loading port forwarding rules: %w", err)
				}
			}
		}

		if !startDaemonIsRunningFn() {
			if err := startEnsurePortsFn(); err != nil {
				return err
			}
			if err := startDaemonRunDetachedFn(); err != nil {
				return fmt.Errorf("starting daemon: %w", err)
			}
			if err := startDaemonWaitFn(); err != nil {
				return err
			}
		} else {
			if _, err := startDaemonSendIPCFn(daemon.Request{Type: daemon.MsgReload}); err != nil {
				return fmt.Errorf("reloading daemon: %w", err)
			}
		}

		if !startDaemonIsChildFn() {
			pf := startNewPortFwdFn()
			if shouldReloadPortForwarding(pf, true) {
				if err := pf.EnsureLoaded(); err != nil {
					return fmt.Errorf("loading port forwarding rules: %w", err)
				}
			}
		}

		if startWait {
			waitPorts := []int{startPort}
			for _, r := range routes {
				waitPorts = append(waitPorts, r.Port)
			}
			for _, p := range waitPorts {
				fmt.Printf("Waiting for localhost:%d (timeout %s)... ", p, startWaitTimeout)
				if err := proxy.WaitForUpstream(p, startWaitTimeout); err != nil {
					fmt.Println("timed out")
					return err
				}
				fmt.Println("ready")
			}
		}

		printServices([]config.Domain{domain})
		return nil
	},
}

func parseRouteFlags(flags []string) ([]config.Route, error) {
	if len(flags) == 0 {
		return nil, nil
	}
	routes := make([]config.Route, 0, len(flags))
	for _, f := range flags {
		parts := strings.SplitN(f, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid route %q: expected path=port (e.g. /api=8080)", f)
		}
		port, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid route port %q: %w", parts[1], err)
		}
		if err := config.ValidateRoute(parts[0], port); err != nil {
			return nil, err
		}
		routes = append(routes, config.Route{Path: parts[0], Port: port})
	}
	return routes, nil
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
	startCmd.Flags().StringVar(&startDomain, "domain", "", "Base domain to use instead of .test (e.g. local.example.com)")
	startCmd.Flags().StringArrayVar(&startRoutes, "route", nil, "Route a path to a different port (e.g. /api=8080), repeatable")
	startCmd.Flags().StringVar(&startLogMode, "log-mode", "", "Access log mode: full|minimal|off")
	startCmd.Flags().BoolVar(&startCors, "cors", false, "Enable CORS headers on proxied responses")
	startCmd.Flags().BoolVar(&startWait, "wait", false, "Wait for the upstream app to become reachable before returning")
	startCmd.Flags().DurationVar(&startWaitTimeout, "timeout", 30*time.Second, "Maximum time to wait for upstream with --wait")
	_ = startCmd.MarkFlagRequired("port")
	rootCmd.AddCommand(startCmd)
}
