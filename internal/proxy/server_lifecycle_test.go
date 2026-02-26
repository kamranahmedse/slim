package proxy

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/kamranahmedse/slim/internal/config"
)

func TestApplyConfigBuildsRoutesAndDefaults(t *testing.T) {
	restore := snapshotProxyCertHooks()
	defer restore()

	cert := &tls.Certificate{}
	ensureLeafCertFn = func(string) error { return nil }
	loadLeafTLSFn = func(string) (*tls.Certificate, error) { return cert, nil }

	s := NewServer(&config.Config{})
	cfg := &config.Config{
		Domains: []config.Domain{
			{Name: "myapp", Port: 3000},
			{Name: "api", Port: 8080},
		},
	}

	if err := s.applyConfig(cfg); err != nil {
		t.Fatalf("applyConfig: %v", err)
	}
	if s.defaultDomain != "myapp" {
		t.Fatalf("expected default domain myapp, got %q", s.defaultDomain)
	}
	if len(s.routes) != 2 || len(s.knownDomains) != 2 || len(s.certCache) != 2 {
		t.Fatalf("expected 2 routes/known/certs, got routes=%d known=%d certs=%d", len(s.routes), len(s.knownDomains), len(s.certCache))
	}
}

func TestApplyConfigPropagatesEnsureError(t *testing.T) {
	restore := snapshotProxyCertHooks()
	defer restore()

	ensureLeafCertFn = func(name string) error { return errors.New("ensure failed: " + name) }
	loadLeafTLSFn = func(string) (*tls.Certificate, error) { return &tls.Certificate{}, nil }

	s := NewServer(&config.Config{})
	err := s.applyConfig(&config.Config{Domains: []config.Domain{{Name: "myapp", Port: 3000}}})
	if err == nil {
		t.Fatal("expected applyConfig to fail when ensureLeafCert fails")
	}
	if !strings.Contains(err.Error(), "ensure") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplyConfigPropagatesLoadError(t *testing.T) {
	restore := snapshotProxyCertHooks()
	defer restore()

	ensureLeafCertFn = func(string) error { return nil }
	loadLeafTLSFn = func(string) (*tls.Certificate, error) { return nil, errors.New("load failed") }

	s := NewServer(&config.Config{})
	err := s.applyConfig(&config.Config{Domains: []config.Domain{{Name: "myapp", Port: 3000}}})
	if err == nil {
		t.Fatal("expected applyConfig to fail when loadLeafTLS fails")
	}
	if !strings.Contains(err.Error(), "loading cert") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReloadConfigLoadsFromDisk(t *testing.T) {
	restore := snapshotProxyCertHooks()
	defer restore()
	initProxyTestConfig(t)

	ensureLeafCertFn = func(string) error { return nil }
	loadLeafTLSFn = func(string) (*tls.Certificate, error) { return &tls.Certificate{}, nil }

	fileCfg := &config.Config{Domains: []config.Domain{{Name: "myapp", Port: 3000}}}
	if err := fileCfg.Save(); err != nil {
		t.Fatalf("Save config: %v", err)
	}

	s := NewServer(&config.Config{})
	loaded, err := s.ReloadConfig()
	if err != nil {
		t.Fatalf("ReloadConfig: %v", err)
	}
	if len(loaded.Domains) != 1 || loaded.Domains[0].Name != "myapp" {
		t.Fatalf("unexpected loaded config: %+v", loaded.Domains)
	}
	if !s.isKnownDomain("myapp") {
		t.Fatalf("expected myapp to be known after reload")
	}
}

func TestStartFailsWhenHTTPPortUnavailable(t *testing.T) {
	s := NewServer(&config.Config{})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen busy http port: %v", err)
	}
	defer ln.Close()

	s.httpAddr = ln.Addr().String()
	s.httpsAddr = "127.0.0.1:0"

	err = s.Start()
	if err == nil {
		t.Fatal("expected Start to fail when HTTP port is in use")
	}
	if !strings.Contains(err.Error(), "listening on "+s.httpAddr) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStartFailsWhenHTTPSPortUnavailableAndClosesHTTPListener(t *testing.T) {
	s := NewServer(&config.Config{})

	httpPort := freeTCPPort(t)
	s.httpAddr = fmt.Sprintf("127.0.0.1:%d", httpPort)

	lnTLS, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen busy https port: %v", err)
	}
	defer lnTLS.Close()
	s.httpsAddr = lnTLS.Addr().String()

	err = s.Start()
	if err == nil {
		t.Fatal("expected Start to fail when HTTPS port is in use")
	}
	if !strings.Contains(err.Error(), "listening on "+s.httpsAddr) {
		t.Fatalf("unexpected error: %v", err)
	}

	// HTTP listener should be closed after HTTPS bind failure.
	lnCheck, err := net.Listen("tcp", s.httpAddr)
	if err != nil {
		t.Fatalf("expected HTTP port to be released, got: %v", err)
	}
	_ = lnCheck.Close()
}

func snapshotProxyCertHooks() func() {
	prevEnsure := ensureLeafCertFn
	prevLoad := loadLeafTLSFn
	return func() {
		ensureLeafCertFn = prevEnsure
		loadLeafTLSFn = prevLoad
	}
}

func initProxyTestConfig(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := config.Init(); err != nil {
		t.Fatalf("config.Init: %v", err)
	}
}
