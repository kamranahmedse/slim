package cmd

import (
	"strings"
	"testing"
)

func TestValidateLogsFlags(t *testing.T) {
	tests := []struct {
		name     string
		flush    bool
		follow   bool
		argCount int
		wantErr  string
	}{
		{
			name:     "flush with follow",
			flush:    true,
			follow:   true,
			argCount: 0,
			wantErr:  "--flush cannot be used with --follow",
		},
		{
			name:     "flush with filter arg",
			flush:    true,
			follow:   false,
			argCount: 1,
			wantErr:  "--flush does not support domain filter",
		},
		{
			name:     "flush valid",
			flush:    true,
			follow:   false,
			argCount: 0,
		},
		{
			name:     "not flushing ignores follow and args",
			flush:    false,
			follow:   true,
			argCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLogsFlags(tt.flush, tt.follow, tt.argCount)
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
