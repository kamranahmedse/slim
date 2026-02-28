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
		info, err := auth.Login()
		if err != nil {
			return err
		}
		fmt.Printf("Logged in as %s (%s)\n", info.Name, info.Email)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
