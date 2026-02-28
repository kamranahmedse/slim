package term

import "testing"

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
