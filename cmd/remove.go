package cmd

import (
	"fmt"

	"github.com/kamrify/localname/internal/config"
	"github.com/kamrify/localname/internal/daemon"
	"github.com/kamrify/localname/internal/hostfile"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:     "remove [name]",
	Aliases: []string{"rm"},
	Short:   "Remove a domain mapping",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if err := cfg.RemoveDomain(name); err != nil {
			return err
		}

		hostfile.Remove(name)

		if daemon.IsRunning() {
			daemon.SendIPC(daemon.Request{Type: daemon.MsgReload})
		}

		fmt.Printf("Removed %s.local\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
