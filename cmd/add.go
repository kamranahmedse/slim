package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/kamrify/localname/internal/config"
	"github.com/kamrify/localname/internal/daemon"
	"github.com/kamrify/localname/internal/hostfile"
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

		if err := hostfile.Add(name); err != nil {
			fmt.Printf("Warning: could not update /etc/hosts: %v\n", err)
			fmt.Println("  Run with sudo to enable local hostname resolution.")
		}

		if daemon.IsRunning() {
			data, _ := json.Marshal(daemon.DomainData{Name: name, Port: addPort})
			daemon.SendIPC(daemon.Request{Type: daemon.MsgReload, Data: data})
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
