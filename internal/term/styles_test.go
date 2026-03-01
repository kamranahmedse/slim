package term

import (
	"testing"

	"charm.land/lipgloss/v2"
)

func TestStyleForStatus(t *testing.T) {
	tests := []struct {
		code int
		want lipgloss.Style
	}{
		{code: 200, want: Green},
		{code: 302, want: Cyan},
		{code: 404, want: Yellow},
		{code: 500, want: Red},
	}

	for _, tt := range tests {
		got := StyleForStatus(tt.code)
		if got.GetForeground() != tt.want.GetForeground() {
			t.Fatalf("StyleForStatus(%d) foreground = %v, want %v", tt.code, got.GetForeground(), tt.want.GetForeground())
		}
	}
}
