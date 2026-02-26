package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/kamranahmedse/localname/internal/config"
	"github.com/kamranahmedse/localname/internal/log"
	"github.com/spf13/cobra"
)

var logsFollow bool

var logsCmd = &cobra.Command{
	Use:   "logs [name]",
	Short: "Show request logs",
	Long: `Tail the access log. Optionally filter by domain name.

  localname logs             # all domains
  localname logs myapp       # only myapp.local
  localname logs -f          # follow (like tail -f)`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		logPath := config.LogPath()

		f, err := os.Open(logPath)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("No logs yet. Start a domain first with 'localname start'.")
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

func formatLogLine(line string) string {
	parts := strings.Split(line, "\t")
	if len(parts) == 4 {
		ts := parts[0]
		domain := parts[1]
		status := parts[2]
		duration := parts[3]

		statusColor := log.Green
		if len(status) > 0 {
			switch status[0] {
			case '5':
				statusColor = log.Red
			case '4':
				statusColor = log.Yellow
			case '3':
				statusColor = log.Cyan
			}
		}

		return fmt.Sprintf("%s%s%s %s%s%s %s%s%s %s%s%s",
			log.Dim, ts, log.Reset,
			log.Magenta, domain, log.Reset,
			statusColor, status, log.Reset,
			log.Dim, duration, log.Reset,
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

	statusColor := log.Green
	if len(status) > 0 {
		switch status[0] {
		case '5':
			statusColor = log.Red
		case '4':
			statusColor = log.Yellow
		case '3':
			statusColor = log.Cyan
		}
	}

	return fmt.Sprintf("%s%s%s %s%s%s %s %s → %s:%s%s %s%s%s %s%s%s",
		log.Dim, ts, log.Reset,
		log.Magenta, domain, log.Reset,
		method,
		path,
		log.Dim, upstream, log.Reset,
		statusColor, status, log.Reset,
		log.Dim, duration, log.Reset,
	)
}

func init() {
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output")
	rootCmd.AddCommand(logsCmd)
}
