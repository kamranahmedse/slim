package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "localname",
	Short: "Map custom .local domains to local dev server ports",
	Long: `localname maps custom .local domains to local dev server ports with HTTPS,
mDNS for LAN access, and WebSocket passthrough for HMR.

  localname add myapp --port 3000
  localname up
  # https://myapp.local → localhost:3000`,
}

func Execute() error {
	return rootCmd.Execute()
}
