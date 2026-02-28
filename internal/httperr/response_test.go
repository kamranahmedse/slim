package httperr

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestFromResponse_JSONErrorField(t *testing.T) {
	resp := &http.Response{
		StatusCode: 400,
		Body:       io.NopCloser(strings.NewReader(`{"error":"invalid email"}`)),
	}
	err := FromResponse(resp)
	if err.Error() != "server error: invalid email (HTTP 400)" {
		t.Fatalf("got %q", err)
	}
}

func TestFromResponse_JSONMessageField(t *testing.T) {
	resp := &http.Response{
		StatusCode: 422,
		Body:       io.NopCloser(strings.NewReader(`{"message":"email is required"}`)),
	}
	err := FromResponse(resp)
	if err.Error() != "server error: email is required (HTTP 422)" {
		t.Fatalf("got %q", err)
	}
}

func TestFromResponse_StatusHint(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{521, "server returned HTTP 521 — the server is temporarily unavailable, please try again later"},
		{500, "server returned HTTP 500 — internal server error, please try again later"},
		{429, "server returned HTTP 429 — too many requests, please wait a moment and try again"},
		{401, "server returned HTTP 401 — unauthorized, please try logging in again"},
		{418, "server returned HTTP 418"},
	}

	for _, tt := range tests {
		resp := &http.Response{
			StatusCode: tt.code,
			Body:       io.NopCloser(strings.NewReader("")),
		}
		err := FromResponse(resp)
		if err.Error() != tt.want {
			t.Errorf("code %d: got %q, want %q", tt.code, err, tt.want)
		}
	}
}

func TestFromResponse_JSONErrorTakesPrecedence(t *testing.T) {
	resp := &http.Response{
		StatusCode: 500,
		Body:       io.NopCloser(strings.NewReader(`{"error":"database connection failed"}`)),
	}
	err := FromResponse(resp)
	want := "server error: database connection failed (HTTP 500)"
	if err.Error() != want {
		t.Fatalf("got %q, want %q", err, want)
	}
}

func TestStatusHint_5xxFallback(t *testing.T) {
	hint := StatusHint(599)
	if hint != "server error, please try again later" {
		t.Fatalf("got %q", hint)
	}
}

func TestStatusHint_4xxNoHint(t *testing.T) {
	hint := StatusHint(418)
	if hint != "" {
		t.Fatalf("expected empty, got %q", hint)
	}
}
