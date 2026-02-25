package cmd

import (
	"fmt"

	"github.com/kamrify/localname/internal/config"
	"github.com/spf13/cobra"
)

var addPort int

var addCmd = &cobra.Command{
	Use:   "add [name]",
	Short: "Add a domain mapping",
	Long:  `Add a .local domain that maps to a local port. For example, "localname add myapp --port 3000" maps myapp.local to localhost:3000.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if err := cfg.AddDomain(name, addPort); err != nil {
			return err
		}

		fmt.Printf("Added %s.local → localhost:%d\n", name, addPort)
		return nil
	},
}

func init() {
	addCmd.Flags().IntVarP(&addPort, "port", "p", 0, "Local port to proxy to (required)")
	addCmd.MarkFlagRequired("port")
	rootCmd.AddCommand(addCmd)
}
