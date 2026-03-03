package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/kamranahmedse/slim/internal/auth"
	"github.com/kamranahmedse/slim/internal/config"
	"github.com/kamranahmedse/slim/internal/log"
	"github.com/spf13/cobra"
)

type domainEntry struct {
	ID         string `json:"id"`
	Domain     string `json:"domain"`
	Verified   bool   `json:"verified"`
	TargetIP   string `json:"target_ip"`
	CreatedAt  string `json:"created_at"`
	VerifiedAt string `json:"verified_at,omitempty"`
}

var domainCmd = &cobra.Command{
	Use:   "domain",
	Short: "Manage custom domains",
}

var domainAddCmd = &cobra.Command{
	Use:   "add <domain>",
	Short: "Add a custom domain",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]

		info, err := auth.Require()
		if err != nil {
			return err
		}

		body, err := json.Marshal(map[string]string{"domain": domain})
		if err != nil {
			return fmt.Errorf("encoding request: %w", err)
		}

		client := &http.Client{Timeout: 10 * time.Second}
		req, err := http.NewRequest("POST", config.APIBaseURL()+"/api/domains", bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+info.Token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("adding domain: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			return apiError(resp, "failed to add domain")
		}

		var result struct {
			Domain   domainEntry `json:"domain"`
			TargetIP string     `json:"target_ip"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}

		targetIP := result.TargetIP
		if targetIP == "" {
			targetIP = result.Domain.TargetIP
		}

		fmt.Printf("\nDomain %s added.\n\n", domain)
		fmt.Printf("Add the following DNS record to verify ownership:\n\n")
		fmt.Printf("  Type:  A\n")
		fmt.Printf("  Name:  %s\n", domain)
		fmt.Printf("  Value: %s\n\n", targetIP)
		fmt.Printf("Then run: slim domain verify %s\n\n", domain)

		return nil
	},
}

var domainListCmd = &cobra.Command{
	Use:   "list",
	Short: "List custom domains",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		info, err := auth.Require()
		if err != nil {
			return err
		}

		domains, err := fetchDomains(info.Token)
		if err != nil {
			return err
		}

		if len(domains) == 0 {
			fmt.Println("No custom domains. Use 'slim domain add <domain>' to add one.")
			return nil
		}

		var rows [][]string
		for _, d := range domains {
			status := "pending"
			if d.Verified {
				status = "verified"
			}
			added := d.CreatedAt
			if t, err := time.Parse(time.RFC3339, d.CreatedAt); err == nil {
				added = log.FormatTimeAgo(t)
			}
			rows = append(rows, []string{d.Domain, status, added})
		}

		t := table.New().
			Headers("DOMAIN", "STATUS", "ADDED").
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

		return nil
	},
}

var domainVerifyCmd = &cobra.Command{
	Use:   "verify <domain>",
	Short: "Verify DNS for a custom domain",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]

		info, err := auth.Require()
		if err != nil {
			return err
		}

		domains, err := fetchDomains(info.Token)
		if err != nil {
			return err
		}

		domainID := findDomainID(domains, domain)
		if domainID == "" {
			return fmt.Errorf("domain %s not found — use 'slim domain add' first", domain)
		}

		client := &http.Client{Timeout: 10 * time.Second}
		req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/domains/%s/verify", config.APIBaseURL(), domainID), nil)
		if err != nil {
			return fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+info.Token)

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("verifying domain: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			msg := strings.TrimSpace(string(bodyBytes))
			if msg == "" {
				msg = resp.Status
			}
			return fmt.Errorf("verification failed: %s", msg)
		}

		fmt.Printf("Domain %s verified successfully.\n", domain)

		return nil
	},
}

var domainRemoveCmd = &cobra.Command{
	Use:   "remove <domain>",
	Short: "Remove a custom domain",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]

		info, err := auth.Require()
		if err != nil {
			return err
		}

		domains, err := fetchDomains(info.Token)
		if err != nil {
			return err
		}

		domainID := findDomainID(domains, domain)
		if domainID == "" {
			return fmt.Errorf("domain %s not found", domain)
		}

		client := &http.Client{Timeout: 10 * time.Second}
		req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/api/domains/%s", config.APIBaseURL(), domainID), nil)
		if err != nil {
			return fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+info.Token)

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("removing domain: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			return apiError(resp, "failed to remove domain")
		}

		fmt.Printf("Domain %s removed.\n", domain)
		return nil
	},
}

func findDomainID(domains []domainEntry, name string) string {
	for _, d := range domains {
		if strings.EqualFold(d.Domain, name) {
			return d.ID
		}
	}
	return ""
}

func apiError(resp *http.Response, action string) error {
	var errResp struct {
		Error string `json:"error"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&errResp)
	if errResp.Error != "" {
		return fmt.Errorf("%s: %s", action, errResp.Error)
	}
	return fmt.Errorf("%s: %s", action, resp.Status)
}

func fetchDomains(token string) ([]domainEntry, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", config.APIBaseURL()+"/api/domains", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching domains: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch domains: %s", resp.Status)
	}

	var body struct {
		Domains []domainEntry `json:"domains"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return body.Domains, nil
}

func init() {
	domainCmd.AddCommand(domainAddCmd)
	domainCmd.AddCommand(domainListCmd)
	domainCmd.AddCommand(domainVerifyCmd)
	domainCmd.AddCommand(domainRemoveCmd)
	rootCmd.AddCommand(domainCmd)
}
