package cmd

import (
	"strings"
	"testing"
)

func TestFormatLogLineMinimal(t *testing.T) {
	tests := []struct {
		name   string
		line   string
		status string
	}{
		{
			name:   "5xx status",
			line:   "12:00:00\tmyapp.local\t500\t10ms",
			status: "500",
		},
		{
			name:   "4xx status",
			line:   "12:00:00\tmyapp.local\t404\t10ms",
			status: "404",
		},
		{
			name:   "3xx status",
			line:   "12:00:00\tmyapp.local\t301\t10ms",
			status: "301",
		},
		{
			name:   "2xx status",
			line:   "12:00:00\tmyapp.local\t200\t10ms",
			status: "200",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatLogLine(tt.line)
			if !strings.Contains(got, "myapp.local") {
				t.Fatalf("expected domain in output, got: %q", got)
			}
			if !strings.Contains(got, tt.status) {
				t.Fatalf("expected status %q in output, got: %q", tt.status, got)
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
