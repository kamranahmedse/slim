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
		routes: map[string]*domainRouter{"myapp.test": {defaultPort: port, defaultHandler: newDomainProxy(port, newUpstreamTransport(), false)}},
	}

	req := httptest.NewRequest(http.MethodGet, "https://myapp.test/health?x=1", nil)
	req.Host = "myapp.test"
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
		if gotHost != "myapp.test" {
			t.Fatalf("expected upstream host %q, got %q", "myapp.test", gotHost)
		}
	default:
		t.Fatal("upstream request was not observed")
	}
}

func TestBuildHandlerRoutesCustomHostname(t *testing.T) {
	hostCh := make(chan string, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hostCh <- r.Host
		w.WriteHeader(http.StatusAccepted)
	}))
	defer upstream.Close()

	port := mustPortFromURL(t, upstream.URL)
	s := &Server{
		cfg:    &config.Config{},
		routes: map[string]*domainRouter{"branch.local.example.com": {defaultPort: port, defaultHandler: newDomainProxy(port, newUpstreamTransport(), false)}},
	}

	req := httptest.NewRequest(http.MethodGet, "https://branch.local.example.com/health", nil)
	req.Host = "branch.local.example.com"
	rr := httptest.NewRecorder()

	buildHandler(s).ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected %d, got %d", http.StatusAccepted, rr.Code)
	}

	select {
	case gotHost := <-hostCh:
		if gotHost != "branch.local.example.com" {
			t.Fatalf("expected upstream host %q, got %q", "branch.local.example.com", gotHost)
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

	req := httptest.NewRequest(http.MethodGet, "https://unknown.test/", nil)
	req.Host = "unknown.test"
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
		routes: map[string]*domainRouter{"myapp.test": {defaultPort: port, defaultHandler: newDomainProxy(port, newUpstreamTransport(), false)}},
	}

	req := httptest.NewRequest(http.MethodGet, "https://myapp.test/", nil)
	req.Host = "myapp.test"
	rr := httptest.NewRecorder()

	buildHandler(s).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("expected %d, got %d", http.StatusBadGateway, rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Waiting for myapp.test") {
		t.Fatalf("expected upstream-down page content, got %q", rr.Body.String())
	}
}

func TestDomainRouterMatch(t *testing.T) {
	transport := newUpstreamTransport()
	proxy := newDomainProxy(3000, transport, false)
	router := &domainRouter{
		defaultPort:    3000,
		defaultHandler: proxy,
		pathRoutes: []pathRoute{
			{prefix: "/api/v2", port: 9090, handler: http.StripPrefix("/api/v2", newDomainProxy(9090, transport, false))},
			{prefix: "/api", port: 8080, handler: http.StripPrefix("/api", newDomainProxy(8080, transport, false))},
			{prefix: "/ws", port: 9000, handler: http.StripPrefix("/ws", newDomainProxy(9000, transport, false))},
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
			"myapp.test": {
				defaultPort:    3000,
				defaultHandler: newDomainProxy(3000, newUpstreamTransport(), false),
				pathRoutes: []pathRoute{
					{prefix: "/api", port: apiPort, handler: http.StripPrefix("/api", newDomainProxy(apiPort, newUpstreamTransport(), false))},
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
		req := httptest.NewRequest(http.MethodGet, "https://myapp.test"+tt.reqPath, nil)
		req.Host = "myapp.test"
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

func TestCORSHeadersNotAddedByDefault(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://example.com")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	port := mustPortFromURL(t, upstream.URL)
	s := &Server{
		cfg:    &config.Config{},
		routes: map[string]*domainRouter{"myapp.test": {defaultPort: port, defaultHandler: newDomainProxy(port, newUpstreamTransport(), false)}},
	}

	req := httptest.NewRequest(http.MethodGet, "https://myapp.test/api", nil)
	req.Host = "myapp.test"
	req.Header.Set("Origin", "http://example.com")
	rr := httptest.NewRecorder()

	buildHandler(s).ServeHTTP(rr, req)

	origins := rr.Result().Header.Values("Access-Control-Allow-Origin")
	if len(origins) != 1 {
		t.Fatalf("expected 1 Access-Control-Allow-Origin header, got %d: %v", len(origins), origins)
	}
	if origins[0] != "http://example.com" {
		t.Fatalf("expected origin %q, got %q", "http://example.com", origins[0])
	}
}

func TestCORSEnabledStripsUpstreamHeaders(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://example.com")
		w.Header().Set("Access-Control-Allow-Methods", "GET")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	port := mustPortFromURL(t, upstream.URL)
	s := &Server{
		cfg:    &config.Config{Cors: true},
		routes: map[string]*domainRouter{"myapp.test": {defaultPort: port, defaultHandler: newDomainProxy(port, newUpstreamTransport(), true)}},
	}

	req := httptest.NewRequest(http.MethodGet, "https://myapp.test/api", nil)
	req.Host = "myapp.test"
	req.Header.Set("Origin", "http://example.com")
	rr := httptest.NewRecorder()

	buildHandler(s).ServeHTTP(rr, req)

	origins := rr.Result().Header.Values("Access-Control-Allow-Origin")
	if len(origins) != 1 {
		t.Fatalf("expected 1 Access-Control-Allow-Origin header, got %d: %v", len(origins), origins)
	}

	methods := rr.Result().Header.Values("Access-Control-Allow-Methods")
	if len(methods) != 1 {
		t.Fatalf("expected 1 Access-Control-Allow-Methods header, got %d: %v", len(methods), methods)
	}

	creds := rr.Result().Header.Values("Access-Control-Allow-Credentials")
	if len(creds) != 1 {
		t.Fatalf("expected 1 Access-Control-Allow-Credentials header, got %d: %v", len(creds), creds)
	}
}

func TestCORSEnabledHandlesPreflight(t *testing.T) {
	s := &Server{
		cfg:    &config.Config{Cors: true},
		routes: map[string]*domainRouter{"myapp.test": {defaultPort: 3000, defaultHandler: newDomainProxy(3000, newUpstreamTransport(), true)}},
	}

	req := httptest.NewRequest(http.MethodOptions, "https://myapp.test/api", nil)
	req.Host = "myapp.test"
	req.Header.Set("Origin", "http://example.com")
	rr := httptest.NewRecorder()

	buildHandler(s).ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected %d for OPTIONS preflight, got %d", http.StatusNoContent, rr.Code)
	}
	if rr.Result().Header.Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Fatalf("expected CORS origin header on preflight response")
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
