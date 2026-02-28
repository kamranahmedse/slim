package proxy

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/kamranahmedse/slim/internal/config"
)

func TestBuildHandlerRoutesKnownDomain(t *testing.T) {
	hostCh := make(chan string, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hostCh <- r.Host
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	port := mustPortFromURL(t, upstream.URL)
	s := &Server{
		cfg:    &config.Config{},
		routes: map[string]*domainRoute{"myapp": {port: port, proxy: newDomainProxy(port, newUpstreamTransport())}},
	}

	req := httptest.NewRequest(http.MethodGet, "https://myapp.local/health?x=1", nil)
	req.Host = "myapp.local"
	rr := httptest.NewRecorder()

	buildHandler(s).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d", http.StatusCreated, rr.Code)
	}
	if body := rr.Body.String(); body != "ok" {
		t.Fatalf("expected body %q, got %q", "ok", body)
	}

	select {
	case gotHost := <-hostCh:
		if gotHost != "myapp.local" {
			t.Fatalf("expected upstream host %q, got %q", "myapp.local", gotHost)
		}
	default:
		t.Fatal("upstream request was not observed")
	}
}

func TestBuildHandlerUnknownDomainReturnsNotFound(t *testing.T) {
	s := &Server{
		cfg:    &config.Config{},
		routes: map[string]*domainRoute{},
	}

	req := httptest.NewRequest(http.MethodGet, "https://unknown.local/", nil)
	req.Host = "unknown.local"
	rr := httptest.NewRecorder()

	buildHandler(s).ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "404") {
		t.Fatalf("expected 404 response, got %q", rr.Body.String())
	}
}

func TestBuildHandlerUpstreamDownReturnsBadGateway(t *testing.T) {
	port := freeTCPPort(t)
	s := &Server{
		cfg:    &config.Config{},
		routes: map[string]*domainRoute{"myapp": {port: port, proxy: newDomainProxy(port, newUpstreamTransport())}},
	}

	req := httptest.NewRequest(http.MethodGet, "https://myapp.local/", nil)
	req.Host = "myapp.local"
	rr := httptest.NewRecorder()

	buildHandler(s).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("expected %d, got %d", http.StatusBadGateway, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Waiting for myapp.local") {
		t.Fatalf("expected upstream-down page content, got %q", rr.Body.String())
	}
}

func mustPortFromURL(t *testing.T, raw string) int {
	t.Helper()

	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}

	var port int
	if _, err := fmt.Sscanf(u.Host, "127.0.0.1:%d", &port); err != nil {
		if _, err := fmt.Sscanf(u.Host, "localhost:%d", &port); err != nil {
			t.Fatalf("extract port from %q: %v", u.Host, err)
		}
	}
	return port
}
