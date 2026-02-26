package cmd

import (
	"fmt"
	"strings"

	"github.com/kamranahmedse/localname/internal/config"
	"github.com/spf13/cobra"
)

var Version = "0.0.1"

var rootCmd = &cobra.Command{
	Use:   "localname",
	Short: "Map custom .local domains to local dev server ports",
	Long: `localname maps custom .local domains to local dev server ports with HTTPS,
mDNS for LAN access, and WebSocket passthrough for HMR.

  localname start myapp --port 3000    # start proxying
  localname start api --port 8080      # add another domain
  localname list                       # see what's running
  localname stop myapp                 # stop one domain
  localname stop                       # stop everything`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("localname %s\n", Version)
		return nil
	},
}

func Execute() error {
	if err := config.Init(); err != nil {
		return err
	}
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func normalizeName(input string) string {
	input = strings.TrimSuffix(input, ".local")
	input = strings.TrimSuffix(input, ".")
	return strings.ToLower(input)
}
