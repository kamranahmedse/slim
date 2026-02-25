package log

import (
	"fmt"
	"time"
)

const (
	reset   = "\033[0m"
	dim     = "\033[2m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	red     = "\033[31m"
	cyan    = "\033[36m"
	magenta = "\033[35m"
)

func colorForStatus(code int) string {
	switch {
	case code >= 500:
		return red
	case code >= 400:
		return yellow
	case code >= 300:
		return cyan
	default:
		return green
	}
}

func Request(domain string, method string, path string, upstream int, status int, duration time.Duration) {
	ts := time.Now().Format("15:04:05")
	statusColor := colorForStatus(status)

	fmt.Printf("%s%s%s %s%s%s %s %s → %s:%d%s %s%d%s %s%s%s\n",
		dim, ts, reset,
		magenta, domain, reset,
		method,
		path,
		dim, upstream, reset,
		statusColor, status, reset,
		dim, formatDuration(duration), reset,
	)
}

func Info(format string, args ...interface{}) {
	fmt.Printf("%s[localname]%s %s\n", cyan, reset, fmt.Sprintf(format, args...))
}

func Error(format string, args ...interface{}) {
	fmt.Printf("%s[localname]%s %s\n", red, reset, fmt.Sprintf(format, args...))
}

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}
