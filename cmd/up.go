package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kamrify/localname/internal/cert"
	"github.com/kamrify/localname/internal/config"
	"github.com/kamrify/localname/internal/log"
	"github.com/kamrify/localname/internal/proxy"
	"github.com/spf13/cobra"
)

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
			log.Info("No domains configured. Add one with 'localname add <name> --port <port>'")
			return nil
		}

		srv := proxy.NewServer(cfg, ":10080", ":10443")

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-sigCh
			log.Info("Shutting down...")
			cancel()
			srv.Shutdown(ctx)
		}()

		return srv.Start()
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
}
