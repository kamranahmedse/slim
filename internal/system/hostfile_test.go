package system

import "testing"

func TestLineHasHost(t *testing.T) {
	tests := []struct {
		line     string
		hostname string
		want     bool
	}{
		{"127.0.0.1 myapp.local # localname", "myapp.local", true},
		{"127.0.0.1 other.local # localname", "myapp.local", false},
		{"127.0.0.1 myapp.local.extra # localname", "myapp.local", false},
		{"# comment", "myapp.local", false},
		{"", "myapp.local", false},
		{"127.0.0.1\tmyapp.local\t# localname", "myapp.local", true},
	}

	for _, tt := range tests {
		got := lineHasHost(tt.line, tt.hostname)
		if got != tt.want {
			t.Errorf("lineHasHost(%q, %q) = %v, want %v", tt.line, tt.hostname, got, tt.want)
		}
	}
}

func TestHasMarkedEntry(t *testing.T) {
	content := "127.0.0.1 localhost\n127.0.0.1 myapp.local # localname\n"

	if !hasMarkedEntry(content, "myapp.local") {
		t.Error("expected to find marked entry for myapp.local")
	}
	if hasMarkedEntry(content, "other.local") {
		t.Error("did not expect to find marked entry for other.local")
	}
	if hasMarkedEntry("", "myapp.local") {
		t.Error("did not expect to find entry in empty content")
	}
}
