package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/kamrify/localname/internal/cert"
	"github.com/kamrify/localname/internal/config"
	"github.com/kamrify/localname/internal/log"
)

type Server struct {
	cfg        *config.Config
	httpAddr   string
	httpsAddr  string
	httpServer *http.Server
	tlsServer  *http.Server
	certCache  map[string]*tls.Certificate
	certMu     sync.RWMutex
}

func NewServer(cfg *config.Config, httpAddr, httpsAddr string) *Server {
	return &Server{
		cfg:       cfg,
		httpAddr:  httpAddr,
		httpsAddr: httpsAddr,
		certCache: make(map[string]*tls.Certificate),
	}
}

func (s *Server) getCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	name := strings.TrimSuffix(hello.ServerName, ".local")

	s.certMu.RLock()
	if c, ok := s.certCache[name]; ok {
		s.certMu.RUnlock()
		return c, nil
	}
	s.certMu.RUnlock()

	if !cert.LeafExists(name) {
		log.Info("Generating certificate for %s.local", name)
		if err := cert.GenerateLeafCert(name); err != nil {
			return nil, fmt.Errorf("generating cert for %s: %w", name, err)
		}
	}

	tlsCert, err := cert.LoadLeafTLS(name)
	if err != nil {
		return nil, err
	}

	s.certMu.Lock()
	s.certCache[name] = tlsCert
	s.certMu.Unlock()

	return tlsCert, nil
}

func (s *Server) Start() error {
	handler := buildHandler(s.cfg)

	// Generate certs for all configured domains upfront
	for _, d := range s.cfg.Domains {
		if !cert.LeafExists(d.Name) {
			log.Info("Generating certificate for %s.local", d.Name)
			if err := cert.GenerateLeafCert(d.Name); err != nil {
				return fmt.Errorf("generating cert for %s: %w", d.Name, err)
			}
		}
	}

	// HTTP server redirects to HTTPS
	s.httpServer = &http.Server{
		Addr: s.httpAddr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			target := "https://" + r.Host + r.URL.RequestURI()
			http.Redirect(w, r, target, http.StatusMovedPermanently)
		}),
	}

	// HTTPS server with dynamic TLS
	s.tlsServer = &http.Server{
		Addr:    s.httpsAddr,
		Handler: handler,
		TLSConfig: &tls.Config{
			GetCertificate: s.getCertificate,
		},
	}

	httpLn, err := net.Listen("tcp", s.httpAddr)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", s.httpAddr, err)
	}

	tlsLn, err := net.Listen("tcp", s.httpsAddr)
	if err != nil {
		httpLn.Close()
		return fmt.Errorf("listening on %s: %w", s.httpsAddr, err)
	}

	tlsLn = tls.NewListener(tlsLn, s.tlsServer.TLSConfig)

	log.Info("HTTP  listening on %s (redirects to HTTPS)", s.httpAddr)
	log.Info("HTTPS listening on %s", s.httpsAddr)

	for _, d := range s.cfg.Domains {
		log.Info("  %s.local → localhost:%d", d.Name, d.Port)
	}

	errCh := make(chan error, 2)
	go func() { errCh <- s.httpServer.Serve(httpLn) }()
	go func() { errCh <- s.tlsServer.Serve(tlsLn) }()

	err = <-errCh
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	var firstErr error

	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if s.tlsServer != nil {
		if err := s.tlsServer.Shutdown(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func (s *Server) ReloadConfig() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	for _, d := range cfg.Domains {
		if !cert.LeafExists(d.Name) {
			log.Info("Generating certificate for %s.local", d.Name)
			if err := cert.GenerateLeafCert(d.Name); err != nil {
				return err
			}
		}
	}

	s.cfg = cfg

	// Clear cert cache to pick up any new domains
	s.certMu.Lock()
	s.certCache = make(map[string]*tls.Certificate)
	s.certMu.Unlock()

	return nil
}
