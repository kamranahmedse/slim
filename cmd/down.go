package cmd

import (
	"fmt"

	"github.com/kamranahmedse/slim/internal/config"
	"github.com/kamranahmedse/slim/internal/daemon"
	"github.com/kamranahmedse/slim/internal/project"
	"github.com/kamranahmedse/slim/internal/system"
	"github.com/spf13/cobra"
)

var (
	downDiscoverFn      = project.Discover
	downWithLockFn      = config.WithLock
	downLoadFn          = config.Load
	downRemoveHostFn    = system.RemoveHost
	downDaemonRunningFn = daemon.IsRunning
	downDaemonSendIPCFn = daemon.SendIPC
)

var downConfigPath string

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop project services from .slim.yaml",
	Long: `Discover .slim.yaml and stop only the services defined in it.
Other domains not in the project config are left running.

  slim down
  slim down --config /path/to/.slim.yaml`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		var pc *project.ProjectConfig
		var err error

		if downConfigPath != "" {
			pc, err = project.Load(downConfigPath)
		} else {
			pc, _, err = downDiscoverFn()
		}
		if err != nil {
			return err
		}

		if err := pc.Validate(); err != nil {
			return err
		}

		var remainingDomains int
		if err := downWithLockFn(func() error {
			cfg, err := downLoadFn()
			if err != nil {
				return err
			}
			remove := make(map[string]bool, len(pc.Services))
			for _, svc := range pc.Services {
				remove[svc.Domain] = true
			}
			filtered := cfg.Domains[:0]
			for _, d := range cfg.Domains {
				if !remove[d.Name] {
					filtered = append(filtered, d)
				}
			}
			cfg.Domains = filtered
			remainingDomains = len(cfg.Domains)
			return cfg.Save()
		}); err != nil {
			return err
		}

		for _, svc := range pc.Services {
			if err := downRemoveHostFn(svc.Domain); err != nil {
				fmt.Printf("Warning: failed to remove %s.local from /etc/hosts: %v\n", svc.Domain, err)
			}
		}

		if downDaemonRunningFn() {
			if remainingDomains == 0 {
				if _, err := downDaemonSendIPCFn(daemon.Request{Type: daemon.MsgShutdown}); err != nil {
					return fmt.Errorf("stopping daemon: %w", err)
				}
			} else {
				if _, err := downDaemonSendIPCFn(daemon.Request{Type: daemon.MsgReload}); err != nil {
					return fmt.Errorf("reloading daemon: %w", err)
				}
			}
		}

		fmt.Printf("Stopped %d project service(s).\n", len(pc.Services))
		return nil
	},
}

func init() {
	downCmd.Flags().StringVarP(&downConfigPath, "config", "c", "", "Path to .slim.yaml")
	rootCmd.AddCommand(downCmd)
}
