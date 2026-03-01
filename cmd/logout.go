package cmd

import (
	"fmt"
	"net/http"
	"time"

	"github.com/kamranahmedse/slim/internal/auth"
	"github.com/kamranahmedse/slim/internal/config"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out of your slim account",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		info, err := auth.LoadAuth()
		if err != nil {
			return err
		}

		if info != nil && info.Token != "" {
			revokeToken(info.Token)
		}

		if err := auth.Logout(); err != nil {
			return err
		}

		fmt.Println("Logged out.")
		return nil
	},
}

func revokeToken(token string) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("DELETE", config.APIBaseURL()+"/api/auth/token", nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
