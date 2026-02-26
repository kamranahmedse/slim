package cmd

import (
	"strings"
	"testing"
	"time"
)

func TestValidateStartWaitFlags(t *testing.T) {
	tests := []struct {
		name           string
		timeoutChanged bool
		wait           bool
		timeout        time.Duration
		wantErr        string
	}{
		{
			name:           "timeout without wait",
			timeoutChanged: true,
			wait:           false,
			timeout:        30 * time.Second,
			wantErr:        "--timeout requires --wait",
		},
		{
			name:           "wait with non-positive timeout",
			timeoutChanged: false,
			wait:           true,
			timeout:        0,
			wantErr:        "--timeout must be greater than 0",
		},
		{
			name:           "valid wait flags",
			timeoutChanged: true,
			wait:           true,
			timeout:        30 * time.Second,
		},
		{
			name:           "default no wait",
			timeoutChanged: false,
			wait:           false,
			timeout:        30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStartWaitFlags(tt.timeoutChanged, tt.wait, tt.timeout)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
			}
		})
	}
}
