//go:build darwin

package system

import "testing"

func TestIsPFAlreadyEnabledOutput(t *testing.T) {
	tests := []struct {
		name string
		out  string
		want bool
	}{
		{
			name: "pf already enabled",
			out:  "No ALTQ support in kernel\npfctl: pf already enabled",
			want: true,
		},
		{
			name: "case insensitive",
			out:  "PF Already Enabled",
			want: true,
		},
		{
			name: "different error",
			out:  "pfctl: syntax error",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPFAlreadyEnabledOutput(tt.out)
			if got != tt.want {
				t.Fatalf("isPFAlreadyEnabledOutput(%q) = %v, want %v", tt.out, got, tt.want)
			}
		})
	}
}
