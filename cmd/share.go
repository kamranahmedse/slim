package cmd

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/kamranahmedse/slim/internal/auth"
	"github.com/kamranahmedse/slim/internal/config"
	"github.com/kamranahmedse/slim/internal/log"
	"github.com/kamranahmedse/slim/internal/term"
	"github.com/kamranahmedse/slim/internal/tunnel"
	"github.com/spf13/cobra"
)

var sharePort int
var shareName string
var sharePassword string
var shareTTL time.Duration
var shareDomain string

var shareCmd = &cobra.Command{
	Use:   "share",
	Short: "Share a local port via tunnel",
	Long: `Expose a local dev server to the internet via a slim.show tunnel.

  slim share --port 3000
  slim share --port 3000 --subdomain cool
  slim share --port 3000 --password secret
  slim share --port 3000 --ttl 30m
  slim share --port 3000 --domain myapp.example.com`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		port := sharePort
		if port < 1 || port > 65535 {
			return fmt.Errorf("invalid port %d: must be between 1 and 65535", port)
		}

		if shareName != "" && shareDomain != "" {
			return fmt.Errorf("cannot use --subdomain and --domain together")
		}

		info, err := auth.Require()
		if err != nil {
			return err
		}
		token := info.Token

		serverURL := config.TunnelServerURL()

		subdomain := shareName

		password := sharePassword

		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		client := tunnel.NewClient(tunnel.ClientOptions{
			ServerURL: serverURL,
			Token:     token,
			Subdomain: subdomain,
			Domain:    shareDomain,
			LocalPort: port,
			Password:  password,
			TTL:       shareTTL,
			OnRequest: func(e tunnel.RequestEvent) {
				statusStyle := term.StyleForStatus(e.Status)
				fmt.Printf("  %s  %-4s %s  %s  %s\n",
					term.Dim.Render(time.Now().Format("15:04:05")),
					e.Method,
					e.Path,
					statusStyle.Render(fmt.Sprintf("%d", e.Status)),
					term.Dim.Render(log.FormatDuration(e.Duration)),
				)
			},
		})

		url, err := client.Connect(ctx)
		if err != nil {
			return fmt.Errorf("tunnel connection failed: %w", err)
		}

		fmt.Println()
		fmt.Printf("  %s → localhost:%d\n", url, port)
		if domainURL := client.DomainURL(); domainURL != "" {
			fmt.Printf("  %s → localhost:%d\n", domainURL, port)
		}
		if password != "" {
			fmt.Printf("  Password: %s\n", password)
		}
		fmt.Printf("\n  Press Ctrl+C to disconnect\n\n")

		<-ctx.Done()
		fmt.Println("\nDisconnected.")
		return nil
	},
}

func init() {
	shareCmd.Flags().IntVarP(&sharePort, "port", "p", 0, "Local port to expose")
	_ = shareCmd.MarkFlagRequired("port")
	shareCmd.Flags().StringVar(&shareName, "subdomain", "", "Vanity subdomain name")
	shareCmd.Flags().StringVar(&sharePassword, "password", "", "Require password for tunnel access")
	shareCmd.Flags().DurationVar(&shareTTL, "ttl", 1*time.Hour, "Tunnel time-to-live (max 1h)")
	shareCmd.Flags().StringVar(&shareDomain, "domain", "", "Custom domain for this tunnel")
	rootCmd.AddCommand(shareCmd)
}
