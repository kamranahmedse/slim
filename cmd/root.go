package cmd

import (
	"strings"

	"github.com/spf13/cobra"
)

var Version = "dev"

var rootCmd = &cobra.Command{
	Use:     "localname",
	Short:   "Map custom .local domains to local dev server ports",
	Version: Version,
	Long: `localname maps custom .local domains to local dev server ports with HTTPS,
mDNS for LAN access, and WebSocket passthrough for HMR.

  localname add myapp --port 3000
  localname up
  # https://myapp.local → localhost:3000`,
}

func Execute() error {
	return rootCmd.Execute()
}

func normalizeName(input string) string {
	input = strings.TrimSuffix(input, ".local")
	input = strings.TrimSuffix(input, ".")
	return strings.ToLower(input)
}
