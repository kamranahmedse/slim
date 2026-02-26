package cmd

import "testing"

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "myapp", want: "myapp"},
		{input: "myapp.local", want: "myapp"},
		{input: "myapp.local.", want: "myapp"},
		{input: "MYAPP.LOCAL", want: "myapp"},
		{input: "  myapp.local  ", want: "myapp"},
		{input: "my-app", want: "my-app"},
	}

	for _, tt := range tests {
		got := normalizeName(tt.input)
		if got != tt.want {
			t.Fatalf("normalizeName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
