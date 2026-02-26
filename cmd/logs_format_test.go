package cmd

import (
	"strings"
	"testing"

	"github.com/kamranahmedse/slim/internal/log"
)

func TestFormatLogLineMinimal(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		colorPrefix string
	}{
		{
			name:        "5xx status is red",
			line:        "12:00:00\tmyapp.local\t500\t10ms",
			colorPrefix: log.Red,
		},
		{
			name:        "4xx status is yellow",
			line:        "12:00:00\tmyapp.local\t404\t10ms",
			colorPrefix: log.Yellow,
		},
		{
			name:        "3xx status is cyan",
			line:        "12:00:00\tmyapp.local\t301\t10ms",
			colorPrefix: log.Cyan,
		},
		{
			name:        "2xx status is green",
			line:        "12:00:00\tmyapp.local\t200\t10ms",
			colorPrefix: log.Green,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatLogLine(tt.line)
			if !strings.Contains(got, "myapp.local") {
				t.Fatalf("expected domain in output, got: %q", got)
			}
			if !strings.Contains(got, tt.colorPrefix) {
				t.Fatalf("expected status color %q in output, got: %q", tt.colorPrefix, got)
			}
		})
	}
}

func TestFormatLogLineFull(t *testing.T) {
	line := "12:00:00\tmyapp.local\tGET\t/api/health\t3000\t200\t12ms"
	got := formatLogLine(line)

	if !strings.Contains(got, "GET") {
		t.Fatalf("expected method in output, got: %q", got)
	}
	if !strings.Contains(got, "/api/health") {
		t.Fatalf("expected path in output, got: %q", got)
	}
	if !strings.Contains(got, "3000") {
		t.Fatalf("expected upstream port in output, got: %q", got)
	}
}

func TestFormatLogLineMalformedPassthrough(t *testing.T) {
	line := "malformed"
	got := formatLogLine(line)
	if got != line {
		t.Fatalf("expected passthrough for malformed line, got: %q", got)
	}
}
