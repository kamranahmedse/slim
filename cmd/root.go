package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/kamranahmedse/slim/internal/config"
	"github.com/kamranahmedse/slim/internal/term"
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
	rootCmd.SilenceErrors = true

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "\n  %s %s\n\n", term.Red.Render("Error:"), err)
		return err
	}

	return nil
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

func printServices(domains []config.Domain) {
	maxLen := 0
	for _, d := range domains {
		u := len("https://") + len(d.Name) + len(".local")
		if u > maxLen {
			maxLen = u
		}
		for _, r := range d.Routes {
			if ru := u + len(r.Path); ru > maxLen {
				maxLen = ru
			}
		}
	}

	arrow := term.Dim.Render("→")

	for _, d := range domains {
		url := fmt.Sprintf("https://%s.local", d.Name)
		fmt.Printf("  %s %s  %s  %s\n",
			term.CheckMark, term.Green.Render(fmt.Sprintf("%-*s", maxLen, url)),
			arrow, term.Dim.Render(fmt.Sprintf("localhost:%d", d.Port)))
		for _, r := range d.Routes {
			fmt.Printf("    %s  %s  %s\n",
				term.Green.Render(fmt.Sprintf("%-*s", maxLen, url+r.Path)),
				arrow, term.Dim.Render(fmt.Sprintf("localhost:%d", r.Port)))
		}
	}
}
