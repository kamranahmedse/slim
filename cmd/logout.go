package cmd

import (
	"fmt"
	"os"

	"github.com/kamranahmedse/slim/internal/config"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out of your slim account",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLogout()
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}

func runLogout() error {
	err := os.Remove(config.AuthPath())
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("Already logged out.")
			return nil
		}
		return fmt.Errorf("failed to remove auth file: %w", err)
	}

	fmt.Println("Logged out.")
	return nil
}
