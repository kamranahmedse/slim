package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/kamrify/localname/internal/daemon"
	"github.com/spf13/cobra"
)

var statusJSON bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the running state and domain health",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !daemon.IsRunning() {
			if statusJSON {
				fmt.Println(`{"running": false}`)
			} else {
				fmt.Println("localname is not running.")
			}
			return nil
		}

		resp, err := daemon.SendIPC(daemon.Request{Type: daemon.MsgStatus})
		if err != nil {
			return err
		}

		if !resp.OK {
			return fmt.Errorf("status failed: %s", resp.Error)
		}

		var status daemon.StatusData
		json.Unmarshal(resp.Data, &status)

		if statusJSON {
			data, _ := json.MarshalIndent(status, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("localname is running (PID %d)\n\n", status.PID)

		if len(status.Domains) == 0 {
			fmt.Println("No domains configured.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "DOMAIN\tPORT\tSTATUS")
		for _, d := range status.Domains {
			health := "\033[32m●\033[0m reachable"
			if !d.Healthy {
				health = "\033[31m●\033[0m unreachable"
			}
			fmt.Fprintf(w, "%s.local\t%d\t%s\n", d.Name, d.Port, health)
		}
		w.Flush()

		return nil
	},
}

func init() {
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "Output as JSON")
	rootCmd.AddCommand(statusCmd)
}
