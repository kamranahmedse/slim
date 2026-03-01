package cmd

import (
	"fmt"

	"github.com/kamranahmedse/slim/internal/auth"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with your slim account",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		existing, _ := auth.LoadAuth()

		info, err := auth.Login()
		if err != nil {
			return err
		}

		if existing != nil && existing.Token == info.Token {
			fmt.Printf("Already logged in as %s (%s)\n", info.Name, info.Email)
		} else {
			fmt.Printf("Logged in as %s (%s)\n", info.Name, info.Email)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
