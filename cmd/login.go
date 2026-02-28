package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/kamranahmedse/slim/internal/config"
	"github.com/spf13/cobra"
)

const apiBase = "https://app.slim.sh"

type authInfo struct {
	Token string `json:"token"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with your slim account",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLogin()
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}

func runLogin() error {
	resp, err := http.Post(apiBase+"/api/auth/cli", "application/json", nil)
	if err != nil {
		return fmt.Errorf("failed to start login: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}

	var cliResp struct {
		Code string `json:"code"`
		URL  string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&cliResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Println("Opening browser to log in...")
	if err := openBrowser(cliResp.URL); err != nil {
		fmt.Printf("Could not open browser. Please visit:\n  %s\n", cliResp.URL)
	}

	fmt.Println("Waiting for authentication...")
	client := &http.Client{Timeout: 5 * time.Second}
	deadline := time.Now().Add(30 * time.Second)

	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)

		pollResp, err := client.Get(fmt.Sprintf("%s/api/auth/cli/poll?code=%s", apiBase, cliResp.Code))
		if err != nil {
			continue
		}

		var result struct {
			Status string `json:"status"`
			Token  string `json:"token"`
			User   struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			} `json:"user"`
		}
		json.NewDecoder(pollResp.Body).Decode(&result)
		pollResp.Body.Close()

		if result.Status != "complete" {
			continue
		}

		auth := authInfo{
			Token: result.Token,
			Name:  result.User.Name,
			Email: result.User.Email,
		}

		if err := saveAuth(auth); err != nil {
			return fmt.Errorf("failed to save credentials: %w", err)
		}

		fmt.Printf("Logged in as %s (%s)\n", auth.Name, auth.Email)
		return nil
	}

	return fmt.Errorf("login timed out — please try again")
}

func saveAuth(auth authInfo) error {
	if err := os.MkdirAll(config.Dir(), 0755); err != nil {
		return err
	}

	data, err := json.Marshal(auth)
	if err != nil {
		return err
	}

	return os.WriteFile(config.AuthPath(), data, 0600)
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	default:
		return fmt.Errorf("unsupported platform")
	}
}
