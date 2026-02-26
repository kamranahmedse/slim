package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/kamrify/localname/internal/config"
	"github.com/kamrify/localname/internal/daemon"
	"github.com/kamrify/localname/internal/proxy"
	"github.com/spf13/cobra"
)

var listJSON bool

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all domains and their status",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if len(cfg.Domains) == 0 {
			fmt.Println("No domains configured. Use 'localname start' to add one.")
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
			if running {
				h := proxy.CheckUpstream(d.Port)
				e.Healthy = &h
			}
			entries[i] = e
		}

		if listJSON {
			data, _ := json.MarshalIndent(entries, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		if running {
			fmt.Fprintln(w, "DOMAIN\tPORT\tSTATUS")
		} else {
			fmt.Fprintln(w, "DOMAIN\tPORT\tSTATUS")
		}

		for _, e := range entries {
			status := "\033[2m-\033[0m"
			if e.Healthy != nil {
				if *e.Healthy {
					status = "\033[32m● reachable\033[0m"
				} else {
					status = "\033[31m● unreachable\033[0m"
				}
			}
			fmt.Fprintf(w, "%s\t%d\t%s\n", e.Domain, e.Port, status)
		}
		w.Flush()

		if !running {
			fmt.Println("\nProxy is not running. Use 'localname start' to start it.")
		}

		return nil
	},
}

func init() {
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output as JSON")
	rootCmd.AddCommand(listCmd)
}
