package proxy

import (
	"crypto/tls"
	"errors"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kamranahmedse/slim/internal/config"
)

func TestGetCertificateRejectsUnknownSNI(t *testing.T) {
	s := &Server{
		cfg:          &config.Config{},
		knownDomains: map[string]struct{}{"myapp": {}},
		certCache:    map[string]*tls.Certificate{},
	}

	_, err := s.getCertificate(&tls.ClientHelloInfo{ServerName: "other.local"})
	if err == nil {
		t.Fatal("expected error for unknown SNI")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("expected not-configured error, got %v", err)
	}
}

func TestGetCertificateUsesDefaultDomainWhenSNIEmpty(t *testing.T) {
	cert := &tls.Certificate{}
	s := &Server{
		cfg:           &config.Config{},
		defaultDomain: "myapp",
		knownDomains:  map[string]struct{}{"myapp": {}},
		certCache:     map[string]*tls.Certificate{"myapp": cert},
	}

	got, err := s.getCertificate(&tls.ClientHelloInfo{})
	if err != nil {
		t.Fatalf("getCertificate: %v", err)
	}
	if got != cert {
		t.Fatal("expected cached default certificate")
	}
}

func TestGetCertificateUsesSingleflightOnCacheMiss(t *testing.T) {
	origEnsure := ensureLeafCertFn
	origLoad := loadLeafTLSFn
	defer func() {
		ensureLeafCertFn = origEnsure
		loadLeafTLSFn = origLoad
	}()

	var ensureCalls int32
	var loadCalls int32
	cert := &tls.Certificate{}

	ensureLeafCertFn = func(name string) error {
		if name != "myapp" {
			return errors.New("unexpected name")
		}
		atomic.AddInt32(&ensureCalls, 1)
		time.Sleep(50 * time.Millisecond)
		return nil
	}
	loadLeafTLSFn = func(name string) (*tls.Certificate, error) {
		if name != "myapp" {
			return nil, errors.New("unexpected name")
		}
		atomic.AddInt32(&loadCalls, 1)
		return cert, nil
	}

	s := &Server{
		cfg:          &config.Config{},
		knownDomains: map[string]struct{}{"myapp": {}},
		certCache:    map[string]*tls.Certificate{},
	}

	const workers = 20
	var wg sync.WaitGroup
	errCh := make(chan error, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := s.getCertificate(&tls.ClientHelloInfo{ServerName: "myapp.local"})
			if err != nil {
				errCh <- err
				return
			}
			if got != cert {
				errCh <- errors.New("unexpected certificate pointer")
			}
		}()
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("getCertificate concurrent error: %v", err)
		}
	}

	if got := atomic.LoadInt32(&ensureCalls); got != 1 {
		t.Fatalf("expected ensureLeafCertFn once, got %d", got)
	}
	if got := atomic.LoadInt32(&loadCalls); got != 1 {
		t.Fatalf("expected loadLeafTLSFn once, got %d", got)
	}
}

func freeTCPPort(t *testing.T) int {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	return ln.Addr().(*net.TCPAddr).Port
}
