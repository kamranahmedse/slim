package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/kamrify/localname/internal/config"
	"github.com/kamrify/localname/internal/log"
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
			// Show last chunk then follow
			seekToTail(f, 50)
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

func seekToTail(f *os.File, lines int) {
	info, err := f.Stat()
	if err != nil || info.Size() == 0 {
		return
	}

	// Read last 8KB to find recent lines
	bufSize := int64(8192)
	if bufSize > info.Size() {
		bufSize = info.Size()
	}

	f.Seek(-bufSize, io.SeekEnd)

	buf := make([]byte, bufSize)
	n, _ := f.Read(buf)
	buf = buf[:n]

	// Find the offset of the Nth-to-last newline
	count := 0
	for i := len(buf) - 1; i >= 0; i-- {
		if buf[i] == '\n' {
			count++
			if count > lines {
				f.Seek(-bufSize+int64(i)+1, io.SeekEnd)
				return
			}
		}
	}

	// Less than N lines in buffer, just show from start of buffer
	f.Seek(-bufSize, io.SeekEnd)
}

func formatLogLine(line string) string {
	// Format: timestamp\tdomain\tmethod\tpath\tupstream\tstatus\tduration
	parts := strings.Split(line, "\t")
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
