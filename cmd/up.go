package cmd

import (
	"fmt"

	"github.com/kamrify/localname/internal/cert"
	"github.com/kamrify/localname/internal/config"
	"github.com/kamrify/localname/internal/daemon"
	"github.com/spf13/cobra"
)

var detach bool

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start the reverse proxy",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !cert.CAExists() {
			return fmt.Errorf("root CA not found — run 'localname setup' first")
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if len(cfg.Domains) == 0 {
			return fmt.Errorf("no domains configured — add one with 'localname add <name> --port <port>'")
		}

		if daemon.IsRunning() {
			return fmt.Errorf("localname is already running — use 'localname down' to stop it first")
		}

		if detach {
			return daemon.RunDetached()
		}
		return daemon.RunForeground()
	},
}

func init() {
	upCmd.Flags().BoolVarP(&detach, "detach", "d", false, "Run in the background")
	rootCmd.AddCommand(upCmd)
}
