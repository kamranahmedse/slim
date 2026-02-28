package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/kamranahmedse/slim/internal/config"
	"github.com/kamranahmedse/slim/internal/term"
	"github.com/spf13/cobra"
)

var logsFollow bool
var logsFlush bool

var logsCmd = &cobra.Command{
	Use:   "logs [name]",
	Short: "Show request logs",
	Long: `Tail the access log. Optionally filter by domain name.

  slim logs             # all domains
  slim logs myapp       # only myapp.local
  slim logs -f          # follow (like tail -f)
  slim logs --flush     # clear log file`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		logPath := config.LogPath()
		if logsFlush {
			if err := validateLogsFlags(logsFlush, logsFollow, len(args)); err != nil {
				return err
			}

			if err := os.Truncate(logPath, 0); err != nil {
				if os.IsNotExist(err) {
					fmt.Println("No logs to clear.")
					return nil
				}
				return fmt.Errorf("clearing logs: %w", err)
			}
			fmt.Println("Cleared access logs.")
			return nil
		}

		f, err := os.Open(logPath)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("No logs yet. Start a domain first with 'slim start'.")
				return nil
			}
			return err
		}
		defer f.Close()

		filter := ""
		if len(args) > 0 {
			filter = normalizeName(args[0]) + ".local"
		}

		if logsFollow {
			f.Seek(0, io.SeekEnd)
		}

		reader := bufio.NewReader(f)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					if !logsFollow {
						break
					}
					time.Sleep(100 * time.Millisecond)
					continue
				}
				return err
			}

			line = strings.TrimRight(line, "\n")
			if filter != "" && !strings.Contains(line, filter) {
				continue
			}

			fmt.Println(formatLogLine(line))
		}

		return nil
	},
}

func validateLogsFlags(flush bool, follow bool, argCount int) error {
	if !flush {
		return nil
	}
	if follow {
		return fmt.Errorf("--flush cannot be used with --follow")
	}
	if argCount > 0 {
		return fmt.Errorf("--flush does not support domain filter")
	}
	return nil
}

func formatLogLine(line string) string {
	parts := strings.Split(line, "\t")
	if len(parts) == 4 {
		ts := parts[0]
		domain := parts[1]
		status := parts[2]
		duration := parts[3]

		statusColor := term.Green
		if len(status) > 0 {
			switch status[0] {
			case '5':
				statusColor = term.Red
			case '4':
				statusColor = term.Yellow
			case '3':
				statusColor = term.Cyan
			}
		}

		return fmt.Sprintf("%s%s%s %s%s%s %s%s%s %s%s%s",
			term.Dim, ts, term.Reset,
			term.Magenta, domain, term.Reset,
			statusColor, status, term.Reset,
			term.Dim, duration, term.Reset,
		)
	}

	if len(parts) < 7 {
		return line
	}

	ts := parts[0]
	domain := parts[1]
	method := parts[2]
	path := parts[3]
	upstream := parts[4]
	status := parts[5]
	duration := parts[6]

	statusColor := term.Green
	if len(status) > 0 {
		switch status[0] {
		case '5':
			statusColor = term.Red
		case '4':
			statusColor = term.Yellow
		case '3':
			statusColor = term.Cyan
		}
	}

	return fmt.Sprintf("%s%s%s %s%s%s %s %s → %s:%s%s %s%s%s %s%s%s",
		term.Dim, ts, term.Reset,
		term.Magenta, domain, term.Reset,
		method,
		path,
		term.Dim, upstream, term.Reset,
		statusColor, status, term.Reset,
		term.Dim, duration, term.Reset,
	)
}

func init() {
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output")
	logsCmd.Flags().BoolVar(&logsFlush, "flush", false, "Clear the access log file")
	rootCmd.AddCommand(logsCmd)
}
