package log

import (
	"bufio"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/kamranahmedse/slim/internal/term"
)

const maxLogSize = 10 << 20 // 10 MB

const (
	logModeFull    = "full"
	logModeMinimal = "minimal"
	logModeOff     = "off"

	logBufferSize  = 4096
	logFlushPeriod = 250 * time.Millisecond
)

var (
	logFile    *os.File
	logMode    = logModeFull
	logCh      chan string
	stopWriter chan struct{}
	writerWG   sync.WaitGroup
	mu         sync.RWMutex
)

func SetOutput(path string, mode string) error {
	mu.Lock()
	defer mu.Unlock()

	shutdownWriterLocked()
	logMode = normalizeMode(mode)
	if logMode == logModeOff {
		return nil
	}

	if info, err := os.Stat(path); err == nil && info.Size() > maxLogSize {
		_ = os.Truncate(path, 0)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	logFile = f
	logCh = make(chan string, logBufferSize)
	stopWriter = make(chan struct{})

	writerWG.Add(1)
	go writerLoop(f, logCh, stopWriter)
	return nil
}

func Close() {
	mu.Lock()
	defer mu.Unlock()

	shutdownWriterLocked()
}

func Request(domain string, method string, path string, upstream int, status int, duration time.Duration) {
	mu.RLock()
	mode := logMode
	ch := logCh
	mu.RUnlock()

	if mode == logModeOff || ch == nil {
		return
	}

	ts := time.Now().Format("15:04:05")
	dur := FormatDuration(duration)

	line := fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%d\t%s\n",
		ts, domain, method, path, upstream, status, dur)
	if mode == logModeMinimal {
		line = fmt.Sprintf("%s\t%s\t%d\t%s\n", ts, domain, status, dur)
	}

	select {
	case ch <- line:
	default:
	}
}

func Info(format string, args ...interface{}) {
	fmt.Printf("%s[slim]%s %s\n", term.Cyan, term.Reset, fmt.Sprintf(format, args...))
}

func Error(format string, args ...interface{}) {
	fmt.Printf("%s[slim]%s %s\n", term.Red, term.Reset, fmt.Sprintf(format, args...))
}

func FormatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

func normalizeMode(mode string) string {
	switch mode {
	case logModeFull, "":
		return logModeFull
	case logModeMinimal:
		return logModeMinimal
	case logModeOff:
		return logModeOff
	default:
		return logModeFull
	}
}

func writerLoop(file *os.File, entries <-chan string, stop <-chan struct{}) {
	defer writerWG.Done()

	buffered := bufio.NewWriterSize(file, 64*1024)
	ticker := time.NewTicker(logFlushPeriod)
	defer ticker.Stop()

	flushAndClose := func() {
		_ = buffered.Flush()
		_ = file.Close()
	}

	for {
		select {
		case line := <-entries:
			if _, err := buffered.WriteString(line); err != nil {
				flushAndClose()
				return
			}
		case <-ticker.C:
			_ = buffered.Flush()
		case <-stop:
			for {
				select {
				case line := <-entries:
					if _, err := buffered.WriteString(line); err != nil {
						flushAndClose()
						return
					}
				default:
					flushAndClose()
					return
				}
			}
		}
	}
}

func shutdownWriterLocked() {
	if stopWriter != nil {
		close(stopWriter)
		writerWG.Wait()
		stopWriter = nil
	}
	logCh = nil
	logFile = nil
}
