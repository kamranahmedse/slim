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
		routes: map[string]*domainRouter{"myapp": {defaultPort: port, defaultHandler: newDomainProxy(port, newUpstreamTransport())}},
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
		routes: map[string]*domainRouter{},
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
		routes: map[string]*domainRouter{"myapp": {defaultPort: port, defaultHandler: newDomainProxy(port, newUpstreamTransport())}},
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

func TestDomainRouterMatch(t *testing.T) {
	transport := newUpstreamTransport()
	proxy := newDomainProxy(3000, transport)
	router := &domainRouter{
		defaultPort:    3000,
		defaultHandler: proxy,
		pathRoutes: []pathRoute{
			{prefix: "/api/v2", port: 9090, handler: http.StripPrefix("/api/v2", newDomainProxy(9090, transport))},
			{prefix: "/api", port: 8080, handler: http.StripPrefix("/api", newDomainProxy(8080, transport))},
			{prefix: "/ws", port: 9000, handler: http.StripPrefix("/ws", newDomainProxy(9000, transport))},
		},
	}

	tests := []struct {
		path     string
		wantPort int
	}{
		{"/", 3000},
		{"/about", 3000},
		{"/api", 8080},
		{"/api/users", 8080},
		{"/api/v2", 9090},
		{"/api/v2/items", 9090},
		{"/apikeys", 3000},
		{"/ws", 9000},
		{"/ws/chat", 9000},
		{"/other", 3000},
	}

	for _, tt := range tests {
		port, _ := router.match(tt.path)
		if port != tt.wantPort {
			t.Errorf("match(%q) = %d, want %d", tt.path, port, tt.wantPort)
		}
	}
}

func TestPathRouteStripsPrefix(t *testing.T) {
	pathCh := make(chan string, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pathCh <- r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	apiPort := mustPortFromURL(t, upstream.URL)
	s := &Server{
		cfg: &config.Config{},
		routes: map[string]*domainRouter{
			"myapp": {
				defaultPort:    3000,
				defaultHandler: newDomainProxy(3000, newUpstreamTransport()),
				pathRoutes: []pathRoute{
					{prefix: "/api", port: apiPort, handler: http.StripPrefix("/api", newDomainProxy(apiPort, newUpstreamTransport()))},
				},
			},
		},
	}

	tests := []struct {
		reqPath  string
		wantPath string
	}{
		{"/api/v1/health", "/v1/health"},
		{"/api/users/123", "/users/123"},
		{"/api", "/"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, "https://myapp.local"+tt.reqPath, nil)
		req.Host = "myapp.local"
		rr := httptest.NewRecorder()

		buildHandler(s).ServeHTTP(rr, req)

		select {
		case gotPath := <-pathCh:
			if gotPath != tt.wantPath {
				t.Errorf("request %s: upstream got path %q, want %q", tt.reqPath, gotPath, tt.wantPath)
			}
		default:
			t.Errorf("request %s: upstream was not called", tt.reqPath)
		}
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
