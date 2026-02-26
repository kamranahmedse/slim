package log

import (
	"fmt"
	"os"
	"sync"
	"time"
)

const (
	Reset   = "\033[0m"
	Dim     = "\033[2m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Red     = "\033[31m"
	Cyan    = "\033[36m"
	Magenta = "\033[35m"
)

const maxLogSize = 10 << 20 // 10 MB

var (
	logFile *os.File
	mu      sync.Mutex
)

func SetOutput(path string) error {
	mu.Lock()
	defer mu.Unlock()

	if info, err := os.Stat(path); err == nil && info.Size() > maxLogSize {
		os.Truncate(path, 0)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	logFile = f
	return nil
}

func Close() {
	mu.Lock()
	defer mu.Unlock()

	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
}

func ColorForStatus(code int) string {
	switch {
	case code >= 500:
		return Red
	case code >= 400:
		return Yellow
	case code >= 300:
		return Cyan
	default:
		return Green
	}
}

func Request(domain string, method string, path string, upstream int, status int, duration time.Duration) {
	ts := time.Now().Format("15:04:05")
	dur := formatDuration(duration)

	mu.Lock()
	if logFile != nil {
		fmt.Fprintf(logFile, "%s\t%s\t%s\t%s\t%d\t%d\t%s\n",
			ts, domain, method, path, upstream, status, dur)
	}
	mu.Unlock()
}

func Info(format string, args ...interface{}) {
	fmt.Printf("%s[localname]%s %s\n", Cyan, Reset, fmt.Sprintf(format, args...))
}

func Error(format string, args ...interface{}) {
	fmt.Printf("%s[localname]%s %s\n", Red, Reset, fmt.Sprintf(format, args...))
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
