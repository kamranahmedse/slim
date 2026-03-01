package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"text/tabwriter"
	"time"

	"github.com/kamranahmedse/slim/internal/auth"
	"github.com/kamranahmedse/slim/internal/config"
	"github.com/kamranahmedse/slim/internal/daemon"
	"github.com/kamranahmedse/slim/internal/proxy"
	"github.com/kamranahmedse/slim/internal/term"
	"github.com/spf13/cobra"
)

var listJSON bool

type activeTunnel struct {
	Subdomain    string `json:"subdomain"`
	URL          string `json:"url"`
	HasPassword  bool   `json:"has_password"`
	ConnectedAt  string `json:"connected_at"`
	ExpiresAt    string `json:"expires_at"`
	RequestCount uint64 `json:"request_count"`
}

func fetchActiveTunnels(token string) []activeTunnel {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", config.APIBaseURL()+"/api/tunnels/active", nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var tunnels []activeTunnel
	if err := json.NewDecoder(resp.Body).Decode(&tunnels); err != nil {
		return nil
	}

	return tunnels
}

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all domains and tunnels",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		running := daemon.IsRunning()

		type domainEntry struct {
			Domain  string `json:"domain"`
			Port    int    `json:"port"`
			Healthy *bool  `json:"healthy,omitempty"`
		}

		var domains []domainEntry
		for _, d := range cfg.Domains {
			domains = append(domains, domainEntry{
				Domain: d.Name + ".local",
				Port:   d.Port,
			})
		}

		if running && len(domains) > 0 {
			ports := make([]int, len(cfg.Domains))
			for i, d := range cfg.Domains {
				ports[i] = d.Port
			}
			health := proxy.CheckUpstreams(ports)
			for i := range domains {
				domains[i].Healthy = &health[i]
			}
		}

		info, _ := auth.LoadAuth()
		var tunnels []activeTunnel
		if info != nil {
			tunnels = fetchActiveTunnels(info.Token)
		}

		if len(domains) == 0 && len(tunnels) == 0 {
			fmt.Println("No domains or tunnels. Use 'slim start' or 'slim share' to create one.")
			return nil
		}

		if listJSON {
			data, err := json.MarshalIndent(map[string]any{
				"domains": domains,
				"tunnels": tunnels,
			}, "", "  ")
			if err != nil {
				return fmt.Errorf("marshaling JSON: %w", err)
			}
			fmt.Println(string(data))
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

		if len(domains) > 0 {
			fmt.Fprintln(w, "DOMAIN\tPORT\tSTATUS")
			for _, e := range domains {
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
		}

		if len(tunnels) > 0 {
			if len(domains) > 0 {
				fmt.Fprintln(w)
			}
			fmt.Fprintln(w, "TUNNEL\tURL\tREQUESTS")
			for _, t := range tunnels {
				fmt.Fprintf(w, "%s\t%s\t%d\n", t.Subdomain+".slim.sh", t.URL, t.RequestCount)
			}
		}

		w.Flush()

		if len(domains) > 0 && !running {
			fmt.Println("\nProxy is not running. Use 'slim start' to start it.")
		}

		return nil
	},
}

func init() {
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output as JSON")
	rootCmd.AddCommand(listCmd)
}
