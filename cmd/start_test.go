package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/kamranahmedse/slim/internal/config"
)

func TestParseRouteFlags(t *testing.T) {
	tests := []struct {
		name    string
		flags   []string
		want    []config.Route
		wantErr string
	}{
		{
			name:  "empty",
			flags: nil,
			want:  nil,
		},
		{
			name:  "single route",
			flags: []string{"/api=8080"},
			want:  []config.Route{{Path: "/api", Port: 8080}},
		},
		{
			name:  "multiple routes",
			flags: []string{"/api=8080", "/ws=9000"},
			want:  []config.Route{{Path: "/api", Port: 8080}, {Path: "/ws", Port: 9000}},
		},
		{
			name:    "missing equals",
			flags:   []string{"/api8080"},
			wantErr: "expected path=port",
		},
		{
			name:    "invalid port",
			flags:   []string{"/api=notaport"},
			wantErr: "invalid route port",
		},
		{
			name:    "missing leading slash",
			flags:   []string{"api=8080"},
			wantErr: "must start with /",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRouteFlags(tt.flags)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("expected %d routes, got %d", len(tt.want), len(got))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("route[%d]: got %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

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
