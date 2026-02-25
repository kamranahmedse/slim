package cmd

import (
	"fmt"

	"github.com/kamrify/localname/internal/daemon"
	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop the background proxy",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !daemon.IsRunning() {
			fmt.Println("localname is not running.")
			return nil
		}

		resp, err := daemon.SendIPC(daemon.Request{Type: daemon.MsgShutdown})
		if err != nil {
			return err
		}

		if !resp.OK {
			return fmt.Errorf("shutdown failed: %s", resp.Error)
		}

		fmt.Println("localname stopped.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(downCmd)
}
