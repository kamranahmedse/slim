package log

import (
	"testing"
	"time"
)

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
		got := FormatDuration(tt.in)
		if got != tt.want {
			t.Fatalf("FormatDuration(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
