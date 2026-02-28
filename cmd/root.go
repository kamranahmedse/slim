package cmd

import (
	"fmt"
	"strings"

	"github.com/kamranahmedse/slim/internal/config"
	"github.com/spf13/cobra"
)

var Version = "0.0.1"

var rootCmd = &cobra.Command{
	Use:   "slim",
	Short: "Map custom .local domains to local dev server ports",
	Long: `slim maps custom .local domains to local dev server ports with HTTPS,
mDNS for LAN access, and WebSocket passthrough for HMR.

  slim start myapp --port 3000    # start proxying
  slim start api --port 8080      # add another domain
  slim list                       # see what's running
  slim stop myapp                 # stop one domain
  slim stop                       # stop everything`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cmd.SilenceUsage = true
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("slim %s\n", Version)
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
	input = strings.ToLower(strings.TrimSpace(input))
	input = strings.TrimSuffix(input, ".")
	input = strings.TrimSuffix(input, ".local")
	return strings.TrimSuffix(input, ".")
}
