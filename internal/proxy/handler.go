package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/kamrify/localname/internal/config"
	"github.com/kamrify/localname/internal/log"
)

func buildHandler(cfg *config.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if idx := strings.IndexByte(host, ':'); idx != -1 {
			host = host[:idx]
		}

		name := strings.TrimSuffix(host, ".local")
		domain, _ := cfg.FindDomain(name)
		if domain == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, notFoundPage, host)
			return
		}

		target := &url.URL{
			Scheme: "http",
			Host:   fmt.Sprintf("localhost:%d", domain.Port),
		}

		proxy := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = target.Scheme
				req.URL.Host = target.Host
				req.Host = r.Host
			},
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusBadGateway)
				fmt.Fprintf(w, upstreamDownPage, host, domain.Port, domain.Port)
			},
		}

		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: 200}
		proxy.ServeHTTP(recorder, r)

		log.Request(host, r.Method, r.URL.Path, domain.Port, recorder.status, time.Since(start))
	})
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

func (r *statusRecorder) Hijack() (c interface{}, buf interface{}, err error) {
	return nil, nil, fmt.Errorf("hijack not supported")
}

const notFoundPage = `<!DOCTYPE html>
<html>
<head><title>localname - Not Found</title>
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
  <p>This domain isn't configured in localname.<br>
  Run <code>localname add &lt;name&gt; --port &lt;port&gt;</code> to set it up.</p>
</div>
</body>
</html>`

const upstreamDownPage = `<!DOCTYPE html>
<html>
<head><title>localname - Waiting for server</title>
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
