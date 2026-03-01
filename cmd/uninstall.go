package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/kamranahmedse/slim/internal/cert"
	"github.com/kamranahmedse/slim/internal/config"
	"github.com/kamranahmedse/slim/internal/daemon"
	"github.com/kamranahmedse/slim/internal/system"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove all slim data and configuration",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if os.Geteuid() != 0 {
			exe, err := os.Executable()
			if err != nil {
				return fmt.Errorf("failed to find slim binary: %w", err)
			}
			sudoCmd := exec.Command("sudo", "--preserve-env=HOME", exe, "uninstall")
			sudoCmd.Stdin = os.Stdin
			sudoCmd.Stdout = os.Stdout
			sudoCmd.Stderr = os.Stderr
			return sudoCmd.Run()
		}

		fmt.Println("Uninstalling slim...")

		if daemon.IsRunning() {
			fmt.Print("  Stopping daemon... ")
			if _, err := daemon.SendIPC(daemon.Request{Type: daemon.MsgShutdown}); err != nil {
				fmt.Printf("skipped (%v)\n", err)
			} else {
				fmt.Println("done")
			}
		}

		fmt.Print("  Removing CA from trust store... ")
		if err := cert.UntrustCA(); err != nil {
			fmt.Printf("skipped (%v)\n", err)
		} else {
			fmt.Println("done")
		}

		fmt.Print("  Removing port forwarding rules... ")
		pf := system.NewPortForwarder()
		if err := pf.Disable(); err != nil {
			fmt.Printf("skipped (%v)\n", err)
		} else {
			fmt.Println("done")
		}

		fmt.Print("  Cleaning /etc/hosts... ")
		if err := system.RemoveAllHosts(); err != nil {
			fmt.Printf("skipped (%v)\n", err)
		} else {
			fmt.Println("done")
		}

		fmt.Print("  Removing ~/.slim/... ")
		os.RemoveAll(config.Dir())
		fmt.Println("done")

		fmt.Print("  Removing slim binary... ")
		if exe, err := os.Executable(); err != nil {
			fmt.Printf("skipped (%v)\n", err)
		} else if err := os.Remove(exe); err != nil {
			fmt.Printf("skipped (%v)\n", err)
		} else {
			fmt.Println("done")
		}

		fmt.Println("\nslim has been completely removed.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
}
