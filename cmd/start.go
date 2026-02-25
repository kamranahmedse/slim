package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kamrify/localname/internal/cert"
	"github.com/kamrify/localname/internal/config"
	"github.com/kamrify/localname/internal/daemon"
	"github.com/kamrify/localname/internal/hostfile"
	"github.com/kamrify/localname/internal/log"
	"github.com/spf13/cobra"
)

var startPort int

var startCmd = &cobra.Command{
	Use:   "start [name] --port [port]",
	Short: "Start proxying a domain",
	Long: `Add a .local domain and start the proxy in one step.

  localname start myapp --port 3000
  # https://myapp.local → localhost:3000

On Ctrl+C the domain is removed and the proxy stops if no other
domains are active.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := normalizeName(args[0])

		if !cert.CAExists() {
			log.Info("Running first-time setup...")
			if err := cert.GenerateCA(); err != nil {
				return err
			}
			if err := cert.TrustCA(); err != nil {
				return fmt.Errorf("trusting CA: %w", err)
			}
			log.Info("Root CA generated and trusted.")
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		existing, _ := cfg.FindDomain(name)
		if existing != nil {
			if existing.Port != startPort {
				cfg.RemoveDomain(name)
				existing = nil
			}
		}

		if existing == nil {
			if err := cfg.AddDomain(name, startPort); err != nil {
				return err
			}
		}

		hostfile.Add(name)

		if daemon.IsRunning() {
			return attachToRunningDaemon(name, startPort)
		}

		return runAndCleanup(name)
	},
}

func attachToRunningDaemon(name string, port int) error {
	data, _ := json.Marshal(daemon.DomainData{Name: name, Port: port})
	daemon.SendIPC(daemon.Request{Type: daemon.MsgReload, Data: data})

	log.Info("https://%s.local → localhost:%d", name, port)
	log.Info("Proxy is already running, domain added. Press Ctrl+C to remove it.")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	cleanup(name)
	return nil
}

func runAndCleanup(name string) error {
	cleanup := func() {
		cleanupDomain(name)
	}

	// Override the default signal handler in daemon.run() so we can
	// clean up our domain first. We do this by deferring cleanup
	// since daemon.RunForeground blocks until shutdown.
	defer cleanup()

	return daemon.RunForeground()
}

func cleanup(name string) {
	log.Info("Removing %s.local...", name)

	cfg, err := config.Load()
	if err != nil {
		return
	}

	cfg.RemoveDomain(name)
	hostfile.Remove(name)

	if daemon.IsRunning() {
		daemon.SendIPC(daemon.Request{Type: daemon.MsgReload})

		reloaded, err := config.Load()
		if err != nil {
			return
		}
		if len(reloaded.Domains) == 0 {
			daemon.SendIPC(daemon.Request{Type: daemon.MsgShutdown})
		}
	}
}

func cleanupDomain(name string) {
	cfg, err := config.Load()
	if err != nil {
		return
	}

	cfg.RemoveDomain(name)
	hostfile.Remove(name)
}

func init() {
	startCmd.Flags().IntVarP(&startPort, "port", "p", 0, "Local port to proxy to (required)")
	startCmd.MarkFlagRequired("port")
	rootCmd.AddCommand(startCmd)
}
