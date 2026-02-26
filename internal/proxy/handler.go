package proxy

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/kamranahmedse/slim/internal/log"
)

type domainRoute struct {
	port  int
	proxy *httputil.ReverseProxy
}

func buildHandler(s *Server) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := normalizeHost(r.Host)
		name, ok := localDomainFromHost(host)
		if !ok {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, notFoundPage, host)
			return
		}

		s.cfgMu.RLock()
		route, found := s.routes[name]
		s.cfgMu.RUnlock()
		if !found {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, notFoundPage, host)
			return
		}

		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: 200}
		route.proxy.ServeHTTP(recorder, r)

		log.Request(host, r.Method, r.URL.RequestURI(), route.port, recorder.status, time.Since(start))
	})
}

func newDomainProxy(port int, transport *http.Transport) *httputil.ReverseProxy {
	target := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("localhost:%d", port),
	}

	return &httputil.ReverseProxy{
		Transport: transport,
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			pr.Out.Host = pr.In.Host
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			host := normalizeHost(r.Host)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadGateway)
			fmt.Fprintf(w, upstreamDownPage, host, port, port)
		},
	}
}

func normalizeHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	host = strings.TrimSuffix(host, ".")

	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	} else if strings.Count(host, ":") == 1 {
		if idx := strings.LastIndex(host, ":"); idx != -1 {
			host = host[:idx]
		}
	}

	host = strings.Trim(host, "[]")
	return strings.TrimSuffix(host, ".")
}

func localDomainFromHost(host string) (string, bool) {
	host = normalizeHost(host)
	if !strings.HasSuffix(host, ".local") {
		return "", false
	}

	name := strings.TrimSuffix(host, ".local")
	return name, name != ""
}

type statusRecorder struct {
	http.ResponseWriter
	status  int
	written bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if !r.written {
		r.status = code
		r.written = true
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := r.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, fmt.Errorf("upstream ResponseWriter does not support hijacking")
}

const notFoundPage = `<!DOCTYPE html>
<html>
<head><title>slim - Not Found</title>
<style>
  body { font-family: -apple-system, system-ui, sans-serif; display: flex; justify-content: center; align-items: center; min-height: 100vh; margin: 0; background: #0a0a0a; color: #e5e5e5; }
  .container { text-align: center; max-width: 480px; }
  h1 { font-size: 1.5rem; font-weight: 600; }
  p { color: #888; line-height: 1.6; }
  code { background: #1a1a1a; padding: 2px 8px; border-radius: 4px; font-size: 0.9em; }
</style>
</head>
<body>
<div class="container">
  <h1>%s</h1>
  <p>This domain isn't configured in slim.<br>
  Run <code>slim start &lt;name&gt; --port &lt;port&gt;</code> to set it up.</p>
</div>
</body>
</html>`

const upstreamDownPage = `<!DOCTYPE html>
<html>
<head><title>slim - Waiting for server</title>
<meta http-equiv="refresh" content="2">
<style>
  body { font-family: -apple-system, system-ui, sans-serif; display: flex; justify-content: center; align-items: center; min-height: 100vh; margin: 0; background: #0a0a0a; color: #e5e5e5; }
  .container { text-align: center; max-width: 480px; }
  h1 { font-size: 1.5rem; font-weight: 600; }
  p { color: #888; line-height: 1.6; }
  .spinner { display: inline-block; width: 20px; height: 20px; border: 2px solid #333; border-top-color: #fff; border-radius: 50%%; animation: spin 0.8s linear infinite; margin-bottom: 1rem; }
  @keyframes spin { to { transform: rotate(360deg); } }
</style>
</head>
<body>
<div class="container">
  <div class="spinner"></div>
  <h1>Waiting for %s</h1>
  <p>The dev server on port %d doesn't seem to be running.<br>
  Start your server and this page will auto-refresh.</p>
  <p style="font-size: 0.85em; color: #555">Expecting a server on localhost:%d</p>
</div>
</body>
</html>`
