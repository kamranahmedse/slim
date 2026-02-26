package log

import (
	"testing"
	"time"
)

func TestColorForStatus(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{code: 200, want: Green},
		{code: 302, want: Cyan},
		{code: 404, want: Yellow},
		{code: 500, want: Red},
	}

	for _, tt := range tests {
		got := ColorForStatus(tt.code)
		if got != tt.want {
			t.Fatalf("ColorForStatus(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		in   time.Duration
		want string
	}{
		{in: 800 * time.Microsecond, want: "800µs"},
		{in: 1250 * time.Microsecond, want: "1ms"},
		{in: 125 * time.Millisecond, want: "125ms"},
		{in: 1500 * time.Millisecond, want: "1.5s"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.in)
		if got != tt.want {
			t.Fatalf("formatDuration(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
