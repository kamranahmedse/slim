package cmd

import (
	"fmt"

	"github.com/kamranahmedse/slim/internal/doctor"
	"github.com/kamranahmedse/slim/internal/term"
	"github.com/spf13/cobra"
)

var doctorRunFn = doctor.Run

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose setup issues",
	Long: `Run diagnostic checks and print a pass/fail/warn checklist.

  slim doctor`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		report := doctorRunFn()
		printReport(report)
		return nil
	},
}

func printReport(report doctor.Report) {
	for _, r := range report.Results {
		icon := statusIcon(r.Status)
		fmt.Printf("  %s  %-22s %s\n", icon, r.Name, r.Message)
	}
}

func statusIcon(s doctor.Status) string {
	switch s {
	case doctor.Pass:
		return term.CheckMark
	case doctor.Warn:
		return term.WarnMark
	case doctor.Fail:
		return term.CrossMark
	default:
		return "?"
	}
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
