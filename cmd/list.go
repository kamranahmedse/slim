package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/kamranahmedse/slim/internal/config"
	"github.com/kamranahmedse/slim/internal/daemon"
	"github.com/kamranahmedse/slim/internal/proxy"
	"github.com/kamranahmedse/slim/internal/term"
	"github.com/spf13/cobra"
)

var listJSON bool

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all domains and their status",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if len(cfg.Domains) == 0 {
			fmt.Println("No domains configured. Use 'slim start' to add one.")
			return nil
		}

		running := daemon.IsRunning()

		type entry struct {
			Domain  string `json:"domain"`
			Port    int    `json:"port"`
			Healthy *bool  `json:"healthy,omitempty"`
		}

		entries := make([]entry, len(cfg.Domains))
		for i, d := range cfg.Domains {
			e := entry{
				Domain: d.Name + ".local",
				Port:   d.Port,
			}
			entries[i] = e
		}
		if running {
			ports := make([]int, len(cfg.Domains))
			for i, d := range cfg.Domains {
				ports[i] = d.Port
			}
			health := proxy.CheckUpstreams(ports)
			for i := range entries {
				entries[i].Healthy = &health[i]
			}
		}

		if listJSON {
			data, err := json.MarshalIndent(entries, "", "  ")
			if err != nil {
				return fmt.Errorf("marshaling JSON: %w", err)
			}
			fmt.Println(string(data))
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "DOMAIN\tPORT\tSTATUS")

		for _, e := range entries {
			status := term.Dim + "-" + term.Reset
			if e.Healthy != nil {
				if *e.Healthy {
					status = term.Green + "● reachable" + term.Reset
				} else {
					status = term.Red + "● unreachable" + term.Reset
				}
			}
			fmt.Fprintf(w, "%s\t%d\t%s\n", e.Domain, e.Port, status)
		}
		w.Flush()

		if !running {
			fmt.Println("\nProxy is not running. Use 'slim start' to start it.")
		}

		return nil
	},
}

func init() {
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output as JSON")
	rootCmd.AddCommand(listCmd)
}
