package cmd

import (
	"fmt"

	"github.com/kamrify/localname/internal/config"
	"github.com/kamrify/localname/internal/daemon"
	"github.com/kamrify/localname/internal/hostfile"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop [name]",
	Short: "Stop proxying a domain, or stop everything",
	Long: `Stop proxying a specific domain, or stop all domains and shut down the daemon.

  localname stop myapp    # stop one domain
  localname stop          # stop everything`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return stopAll()
		}
		return stopOne(normalizeName(args[0]))
	},
}

func stopOne(name string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if _, idx := cfg.FindDomain(name); idx == -1 {
		return fmt.Errorf("%s.local is not running", name)
	}

	cfg.RemoveDomain(name)
	hostfile.Remove(name)

	if daemon.IsRunning() {
		if len(cfg.Domains) == 0 {
			daemon.SendIPC(daemon.Request{Type: daemon.MsgShutdown})
			fmt.Printf("Stopped %s.local (daemon shut down)\n", name)
		} else {
			daemon.SendIPC(daemon.Request{Type: daemon.MsgReload})
			fmt.Printf("Stopped %s.local\n", name)
		}
	} else {
		fmt.Printf("Stopped %s.local\n", name)
	}

	return nil
}

func stopAll() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if len(cfg.Domains) == 0 && !daemon.IsRunning() {
		fmt.Println("Nothing is running.")
		return nil
	}

	for _, d := range cfg.Domains {
		hostfile.Remove(d.Name)
	}

	cfg.Domains = nil
	cfg.Save()

	if daemon.IsRunning() {
		daemon.SendIPC(daemon.Request{Type: daemon.MsgShutdown})
	}

	fmt.Println("Stopped all domains.")
	return nil
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
