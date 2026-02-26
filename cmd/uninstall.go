package cmd

import (
	"fmt"
	"os"

	"github.com/kamranahmedse/localname/internal/cert"
	"github.com/kamranahmedse/localname/internal/config"
	"github.com/kamranahmedse/localname/internal/daemon"
	"github.com/kamranahmedse/localname/internal/hostfile"
	"github.com/kamranahmedse/localname/internal/portfwd"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove all localname data and configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Uninstalling localname...")

		if daemon.IsRunning() {
			fmt.Print("  Stopping daemon... ")
			daemon.SendIPC(daemon.Request{Type: daemon.MsgShutdown})
			fmt.Println("done")
		}

		fmt.Print("  Removing CA from trust store... ")
		if err := cert.UntrustCA(); err != nil {
			fmt.Printf("skipped (%v)\n", err)
		} else {
			fmt.Println("done")
		}

		fmt.Print("  Removing port forwarding rules... ")
		pf := portfwd.New()
		pf.Disable()
		fmt.Println("done")

		fmt.Print("  Cleaning /etc/hosts... ")
		if err := hostfile.RemoveAll(); err != nil {
			fmt.Printf("skipped (%v)\n", err)
		} else {
			fmt.Println("done")
		}

		fmt.Print("  Removing ~/.localname/... ")
		os.RemoveAll(config.Dir())
		fmt.Println("done")

		fmt.Println("\nlocalname has been completely removed.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
}
