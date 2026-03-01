package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
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

	var body struct {
		Tunnels []activeTunnel `json:"tunnels"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil
	}

	return body.Tunnels
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

		type routeEntry struct {
			Path    string `json:"path"`
			Port    int    `json:"port"`
			Healthy *bool  `json:"healthy,omitempty"`
		}

		type domainEntry struct {
			Domain  string       `json:"domain"`
			Port    int          `json:"port"`
			Healthy *bool        `json:"healthy,omitempty"`
			Routes  []routeEntry `json:"routes,omitempty"`
		}

		var domains []domainEntry
		for _, d := range cfg.Domains {
			entry := domainEntry{
				Domain: d.Name + ".local",
				Port:   d.Port,
			}
			for _, r := range d.Routes {
				entry.Routes = append(entry.Routes, routeEntry{Path: r.Path, Port: r.Port})
			}
			domains = append(domains, entry)
		}

		if running && len(domains) > 0 {
			var allPorts []int
			for _, d := range cfg.Domains {
				allPorts = append(allPorts, d.Port)
				for _, r := range d.Routes {
					allPorts = append(allPorts, r.Port)
				}
			}
			health := proxy.CheckUpstreams(allPorts)
			idx := 0
			for i := range domains {
				domains[i].Healthy = &health[idx]
				idx++
				for j := range domains[i].Routes {
					domains[i].Routes[j].Healthy = &health[idx]
					idx++
				}
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

		if len(domains) > 0 {
			var rows [][]string
			for _, e := range domains {
				status := term.Dim.Render("-")
				if e.Healthy != nil {
					if *e.Healthy {
						status = term.Green.Render("● reachable")
					} else {
						status = term.Red.Render("● unreachable")
					}
				}
				rows = append(rows, []string{e.Domain, fmt.Sprintf("%d", e.Port), status})
				for _, r := range e.Routes {
					rStatus := term.Dim.Render("-")
					if r.Healthy != nil {
						if *r.Healthy {
							rStatus = term.Green.Render("● reachable")
						} else {
							rStatus = term.Red.Render("● unreachable")
						}
					}
					rows = append(rows, []string{"  " + r.Path, fmt.Sprintf("%d", r.Port), rStatus})
				}
			}

			t := table.New().
				Headers("DOMAIN", "PORT", "STATUS").
				Rows(rows...).
				BorderTop(false).
				BorderBottom(false).
				BorderLeft(false).
				BorderRight(false).
				BorderColumn(false).
				BorderHeader(false).
				StyleFunc(func(row, col int) lipgloss.Style {
					s := lipgloss.NewStyle().PaddingRight(2)
					if row == table.HeaderRow {
						s = s.Bold(true).Faint(true)
					}
					return s
				})
			fmt.Println(t)
		}

		if len(tunnels) > 0 {
			if len(domains) > 0 {
				fmt.Println()
			}
			var rows [][]string
			for _, t := range tunnels {
				rows = append(rows, []string{t.Subdomain + ".slim.show", t.URL, fmt.Sprintf("%d", t.RequestCount)})
			}

			t := table.New().
				Headers("TUNNEL", "URL", "REQUESTS").
				Rows(rows...).
				BorderTop(false).
				BorderBottom(false).
				BorderLeft(false).
				BorderRight(false).
				BorderColumn(false).
				BorderHeader(false).
				StyleFunc(func(row, col int) lipgloss.Style {
					s := lipgloss.NewStyle().PaddingRight(2)
					if row == table.HeaderRow {
						s = s.Bold(true).Faint(true)
					}
					return s
				})
			fmt.Println(t)
		}

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
