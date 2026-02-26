package cmd

import (
	"fmt"

	"github.com/kamranahmedse/slim/internal/config"
	"github.com/kamranahmedse/slim/internal/daemon"
	"github.com/kamranahmedse/slim/internal/system"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop [name]",
	Short: "Stop proxying a domain, or stop everything",
	Long: `Stop proxying a specific domain, or stop all domains and shut down the daemon.

  slim stop myapp    # stop one domain
  slim stop          # stop everything`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return stopAll()
		}
		return stopOne(normalizeName(args[0]))
	},
}

func stopOne(name string) error {
	var remainingDomains int

	if err := config.WithLock(func() error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if _, idx := cfg.FindDomain(name); idx == -1 {
			return fmt.Errorf("%s.local is not running", name)
		}

		if err := cfg.RemoveDomain(name); err != nil {
			return err
		}
		remainingDomains = len(cfg.Domains)
		return nil
	}); err != nil {
		return err
	}

	if err := system.RemoveHost(name); err != nil {
		return fmt.Errorf("updating /etc/hosts: %w", err)
	}

	if daemon.IsRunning() {
		if remainingDomains == 0 {
			if _, err := daemon.SendIPC(daemon.Request{Type: daemon.MsgShutdown}); err != nil {
				return fmt.Errorf("stopping daemon: %w", err)
			}
			fmt.Printf("Stopped %s.local (daemon shut down)\n", name)
		} else {
			if _, err := daemon.SendIPC(daemon.Request{Type: daemon.MsgReload}); err != nil {
				return fmt.Errorf("reloading daemon: %w", err)
			}
			fmt.Printf("Stopped %s.local\n", name)
		}
	} else {
		fmt.Printf("Stopped %s.local\n", name)
	}

	return nil
}

func stopAll() error {
	var domains []config.Domain

	if err := config.WithLock(func() error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		domains = cfg.Domains
		if len(domains) > 0 {
			cfg.Domains = nil
			return cfg.Save()
		}
		return nil
	}); err != nil {
		return err
	}

	if len(domains) == 0 && !daemon.IsRunning() {
		fmt.Println("Nothing is running.")
		return nil
	}

	for _, d := range domains {
		if err := system.RemoveHost(d.Name); err != nil {
			fmt.Printf("Warning: failed to remove %s.local from /etc/hosts: %v\n", d.Name, err)
		}
	}

	if daemon.IsRunning() {
		if _, err := daemon.SendIPC(daemon.Request{Type: daemon.MsgShutdown}); err != nil {
			return fmt.Errorf("stopping daemon: %w", err)
		}
	}

	fmt.Println("Stopped all domains.")
	return nil
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
