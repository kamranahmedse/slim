package cmd

import (
	"fmt"
	"os"

	"github.com/kamrify/localname/internal/service"
	"github.com/spf13/cobra"
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage the localname system service",
}

var serviceInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install localname as a login service",
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := service.New()
		if svc.IsInstalled() {
			return fmt.Errorf("service is already installed")
		}

		binary, err := os.Executable()
		if err != nil {
			return fmt.Errorf("finding binary path: %w", err)
		}

		if err := svc.Install(binary); err != nil {
			return err
		}

		fmt.Println("Service installed. localname will start automatically on login.")
		return nil
	},
}

var serviceUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove the localname login service",
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := service.New()
		if !svc.IsInstalled() {
			fmt.Println("Service is not installed.")
			return nil
		}

		if err := svc.Uninstall(); err != nil {
			return err
		}

		fmt.Println("Service removed.")
		return nil
	},
}

func init() {
	serviceCmd.AddCommand(serviceInstallCmd)
	serviceCmd.AddCommand(serviceUninstallCmd)
	rootCmd.AddCommand(serviceCmd)
}
