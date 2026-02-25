package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/kamrify/localname/internal/config"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all domain mappings",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if len(cfg.Domains) == 0 {
			fmt.Println("No domains configured. Use 'localname add' to add one.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "DOMAIN\tPORT")
		for _, d := range cfg.Domains {
			fmt.Fprintf(w, "%s.local\t%d\n", d.Name, d.Port)
		}
		w.Flush()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
