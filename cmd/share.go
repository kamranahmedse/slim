package cmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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
var sharePassword bool
var shareTTL time.Duration

var shareCmd = &cobra.Command{
	Use:   "share",
	Short: "Share a local port via tunnel",
	Long: `Expose a local dev server to the internet via a slim.sh tunnel.

  slim share --port 3000
  slim share --port 3000 --subdomain cool
  slim share --port 3000 --password
  slim share --port 3000 --ttl 30m`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		port := sharePort
		if port < 1 || port > 65535 {
			return fmt.Errorf("invalid port %d: must be between 1 and 65535", port)
		}

		info, err := auth.LoadAuth()
		if err != nil {
			return err
		}
		if info == nil {
			return fmt.Errorf("not logged in — run 'slim login' first")
		}
		token := info.Token

		serverURL := config.TunnelServerURL()

		subdomain := shareName

		password := ""
		if sharePassword {
			b := make([]byte, 16)
			if _, err := rand.Read(b); err != nil {
				return fmt.Errorf("generating password: %w", err)
			}
			password = hex.EncodeToString(b)[:16]
		}

		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		client := tunnel.NewClient(tunnel.ClientOptions{
			ServerURL: serverURL,
			Token:     token,
			Subdomain: subdomain,
			LocalPort: port,
			Password:  password,
			TTL:       shareTTL,
			OnRequest: func(e tunnel.RequestEvent) {
				statusColor := term.ColorForStatus(e.Status)
				fmt.Printf("  %s%s%s  %s%-4s%s %s  %s%d%s  %s%s%s\n",
					term.Dim, time.Now().Format("15:04:05"), term.Reset,
					"", e.Method, "",
					e.Path,
					statusColor, e.Status, term.Reset,
					term.Dim, log.FormatDuration(e.Duration), term.Reset,
				)
			},
		})

		url, err := client.Connect(ctx)
		if err != nil {
			return fmt.Errorf("tunnel connection failed: %w", err)
		}

		fmt.Println()
		fmt.Printf("  %s → localhost:%d\n", url, port)
		if sharePassword {
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
	shareCmd.Flags().BoolVar(&sharePassword, "password", false, "Require password (auto-generated)")
	shareCmd.Flags().DurationVar(&shareTTL, "ttl", 1*time.Hour, "Tunnel time-to-live (max 1h)")
	rootCmd.AddCommand(shareCmd)
}
