package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"text/tabwriter"

	"github.com/kamranahmedse/localname/internal/config"
	"github.com/kamranahmedse/localname/internal/daemon"
	"github.com/kamranahmedse/localname/internal/log"
	"github.com/kamranahmedse/localname/internal/proxy"
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
			entries[i] = e
		}
		if running {
			health := make([]bool, len(cfg.Domains))
			var wg sync.WaitGroup
			sem := make(chan struct{}, 16)
			for i, d := range cfg.Domains {
				wg.Add(1)
				go func(idx int, port int) {
					defer wg.Done()
					sem <- struct{}{}
					health[idx] = proxy.CheckUpstream(port)
					<-sem
				}(i, d.Port)
			}
			wg.Wait()
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
			status := log.Dim + "-" + log.Reset
			if e.Healthy != nil {
				if *e.Healthy {
					status = log.Green + "● reachable" + log.Reset
				} else {
					status = log.Red + "● unreachable" + log.Reset
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
