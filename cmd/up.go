package cmd

import (
	"fmt"
	"strings"

	"github.com/kamranahmedse/slim/internal/cert"
	"github.com/kamranahmedse/slim/internal/config"
	"github.com/kamranahmedse/slim/internal/daemon"
	"github.com/kamranahmedse/slim/internal/project"
	"github.com/kamranahmedse/slim/internal/setup"
	"github.com/kamranahmedse/slim/internal/system"
	"github.com/spf13/cobra"
)

var (
	upDiscoverFn          = project.Discover
	upEnsureFirstRunFn    = setup.EnsureFirstRun
	upWithLockFn          = config.WithLock
	upLoadFn              = config.Load
	upAddHostFn           = system.AddHost
	upEnsureLeafCertFn    = cert.EnsureLeafCert
	upDaemonIsRunningFn   = daemon.IsRunning
	upEnsurePortsFn       = setup.EnsureProxyPortsAvailable
	upDaemonRunDetachedFn = daemon.RunDetached
	upDaemonWaitFn        = daemon.WaitForDaemon
	upDaemonSendIPCFn     = daemon.SendIPC
)

var upConfigPath string

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start all services from .slim.yaml",
	Long: `Discover .slim.yaml in the current or parent directories,
then start all services defined in it.

  slim up
  slim up --config /path/to/.slim.yaml`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		var pc *project.ProjectConfig
		var path string
		var err error

		if upConfigPath != "" {
			path = upConfigPath
			pc, err = project.Load(path)
		} else {
			pc, path, err = upDiscoverFn()
		}
		if err != nil {
			return err
		}

		if err := pc.Validate(); err != nil {
			return err
		}

		fmt.Printf("Using %s\n", path)

		if err := upEnsureFirstRunFn(); err != nil {
			return err
		}

		if err := upWithLockFn(func() error {
			cfg, err := upLoadFn()
			if err != nil {
				return err
			}
			if pc.LogMode != "" {
				cfg.LogMode = strings.ToLower(strings.TrimSpace(pc.LogMode))
			}
			for _, svc := range pc.Services {
				if existing, idx := cfg.FindDomain(svc.Domain); existing != nil {
					cfg.Domains[idx].Port = svc.Port
					cfg.Domains[idx].Routes = svc.Routes
				} else {
					cfg.Domains = append(cfg.Domains, config.Domain{
						Name:   svc.Domain,
						Port:   svc.Port,
						Routes: svc.Routes,
					})
				}
			}
			return cfg.Save()
		}); err != nil {
			return err
		}

		for _, svc := range pc.Services {
			if err := upAddHostFn(svc.Domain); err != nil {
				return fmt.Errorf("updating /etc/hosts for %s: %w", svc.Domain, err)
			}
			if err := upEnsureLeafCertFn(svc.Domain); err != nil {
				return fmt.Errorf("generating certificate for %s: %w", svc.Domain, err)
			}
		}

		if !upDaemonIsRunningFn() {
			if err := upEnsurePortsFn(); err != nil {
				return err
			}
			if err := upDaemonRunDetachedFn(); err != nil {
				return fmt.Errorf("starting daemon: %w", err)
			}
			if err := upDaemonWaitFn(); err != nil {
				return err
			}
		} else {
			if _, err := upDaemonSendIPCFn(daemon.Request{Type: daemon.MsgReload}); err != nil {
				return fmt.Errorf("reloading daemon: %w", err)
			}
		}

		domains := make([]config.Domain, len(pc.Services))
		for i, svc := range pc.Services {
			domains[i] = config.Domain{Name: svc.Domain, Port: svc.Port, Routes: svc.Routes}
		}
		printServices(domains)

		return nil
	},
}

func init() {
	upCmd.Flags().StringVarP(&upConfigPath, "config", "c", "", "Path to .slim.yaml")
	rootCmd.AddCommand(upCmd)
}
