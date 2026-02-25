package cmd

import (
	"fmt"

	"github.com/kamrify/localname/internal/cert"
	"github.com/kamrify/localname/internal/portfwd"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "One-time setup: generate CA, trust it, and configure port forwarding",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cert.CAExists() {
			fmt.Println("Root CA already exists, skipping generation.")
		} else {
			fmt.Print("Generating root CA... ")
			if err := cert.GenerateCA(); err != nil {
				return err
			}
			fmt.Println("done")
		}

		fmt.Println("Trusting root CA (you may be prompted for your password)...")
		if err := cert.TrustCA(); err != nil {
			return err
		}
		fmt.Println("Root CA trusted.")

		fmt.Print("Configuring port forwarding (80→10080, 443→10443)... ")
		pf := portfwd.New()
		if err := pf.Enable(); err != nil {
			fmt.Println("skipped")
			fmt.Printf("  Warning: %v\n", err)
			fmt.Println("  You can set this up manually or re-run setup with sudo.")
		} else {
			fmt.Println("done")
		}

		fmt.Println("\nSetup complete! You can now add domains with 'localname add'.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
