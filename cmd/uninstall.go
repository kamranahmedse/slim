package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/kamranahmedse/slim/internal/cert"
	"github.com/kamranahmedse/slim/internal/config"
	"github.com/kamranahmedse/slim/internal/daemon"
	"github.com/kamranahmedse/slim/internal/system"
	"github.com/kamranahmedse/slim/internal/term"
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

		steps := []term.Step{
			{
				Name: "Stopping daemon",
				Run: func() (string, error) {
					if !daemon.IsRunning() {
						return "skipped (not running)", nil
					}
					if _, err := daemon.SendIPC(daemon.Request{Type: daemon.MsgShutdown}); err != nil {
						return fmt.Sprintf("skipped (%v)", err), nil
					}
					return "done", nil
				},
			},
			{
				Name: "Removing CA from trust store",
				Run: func() (string, error) {
					if err := cert.UntrustCA(); err != nil {
						return fmt.Sprintf("skipped (%v)", err), nil
					}
					return "done", nil
				},
			},
			{
				Name: "Removing port forwarding rules",
				Run: func() (string, error) {
					pf := system.NewPortForwarder()
					if err := pf.Disable(); err != nil {
						return fmt.Sprintf("skipped (%v)", err), nil
					}
					return "done", nil
				},
			},
			{
				Name: "Cleaning /etc/hosts",
				Run: func() (string, error) {
					if err := system.RemoveAllHosts(); err != nil {
						return fmt.Sprintf("skipped (%v)", err), nil
					}
					return "done", nil
				},
			},
			{
				Name: "Removing ~/.slim/",
				Run: func() (string, error) {
					os.RemoveAll(config.Dir())
					return "done", nil
				},
			},
			{
				Name: "Removing slim binary",
				Run: func() (string, error) {
					exe, err := os.Executable()
					if err != nil {
						return fmt.Sprintf("skipped (%v)", err), nil
					}
					if err := os.Remove(exe); err != nil {
						return fmt.Sprintf("skipped (%v)", err), nil
					}
					return "done", nil
				},
			},
		}

		if err := term.RunSteps(steps); err != nil {
			return err
		}

		fmt.Println("\nslim has been completely removed.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
}
