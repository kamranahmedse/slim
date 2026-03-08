package doctor

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kamranahmedse/slim/internal/config"
	"github.com/kamranahmedse/slim/internal/daemon"
	"github.com/kamranahmedse/slim/internal/system"
)

func setupDoctorTest(t *testing.T) func() {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := config.Init(); err != nil {
		t.Fatalf("config.Init: %v", err)
	}

	prevReadFile := readFileFn
	prevDaemonRunning := daemonIsRunningFn
	prevDaemonIPC := daemonSendIPCFn
	prevPortFwd := newPortFwdFn
	prevConfigLoad := configLoadFn
	prevDialTimeout := dialTimeoutFn

	return func() {
		readFileFn = prevReadFile
		daemonIsRunningFn = prevDaemonRunning
		daemonSendIPCFn = prevDaemonIPC
		newPortFwdFn = prevPortFwd
		configLoadFn = prevConfigLoad
		dialTimeoutFn = prevDialTimeout
	}
}

type mockPortFwd struct {
	enabled bool
	loaded  bool
}

func (m *mockPortFwd) Enable() error       { return nil }
func (m *mockPortFwd) Disable() error      { return nil }
func (m *mockPortFwd) IsEnabled() bool     { return m.enabled }
func (m *mockPortFwd) IsLoaded() bool      { return m.loaded }
func (m *mockPortFwd) EnsureLoaded() error { return nil }

func TestCheckPortForwarding(t *testing.T) {
	restore := setupDoctorTest(t)
	defer restore()

	daemonIsRunningFn = func() bool { return true }
	dialTimeoutFn = func(network, address string, timeout time.Duration) (net.Conn, error) {
		client, server := net.Pipe()
		_ = server.Close()
		return client, nil
	}

	newPortFwdFn = func() system.PortForwarder { return &mockPortFwd{enabled: true, loaded: true} }
	r := checkPortForwarding()
	if r.Status != Pass {
		t.Fatalf("expected Pass, got %v: %s", r.Status, r.Message)
	}

	newPortFwdFn = func() system.PortForwarder { return &mockPortFwd{enabled: false} }
	r = checkPortForwarding()
	if r.Status != Warn {
		t.Fatalf("expected Warn, got %v: %s", r.Status, r.Message)
	}

	newPortFwdFn = func() system.PortForwarder { return &mockPortFwd{enabled: true, loaded: false} }
	r = checkPortForwarding()
	if r.Status != Fail {
		t.Fatalf("expected Fail for enabled but not loaded, got %v: %s", r.Status, r.Message)
	}

	daemonIsRunningFn = func() bool { return false }
	r = checkPortForwarding()
	if r.Status != Warn {
		t.Fatalf("expected Warn for enabled but not loaded when daemon is stopped, got %v: %s", r.Status, r.Message)
	}

	daemonIsRunningFn = func() bool { return true }
	newPortFwdFn = func() system.PortForwarder { return &mockPortFwd{enabled: true, loaded: true} }
	dialTimeoutFn = func(network, address string, timeout time.Duration) (net.Conn, error) {
		return nil, errors.New("connection refused")
	}
	r = checkPortForwarding()
	if r.Status != Fail {
		t.Fatalf("expected Fail when ingress ports are unreachable, got %v: %s", r.Status, r.Message)
	}
}

func TestCheckHostsFile(t *testing.T) {
	restore := setupDoctorTest(t)
	defer restore()

	readFileFn = func(path string) ([]byte, error) {
		return []byte("127.0.0.1 myapp.test # slim\n"), nil
	}
	r := checkHostsFile("myapp.test")
	if r.Status != Pass {
		t.Fatalf("expected Pass, got %v: %s", r.Status, r.Message)
	}

	readFileFn = func(path string) ([]byte, error) {
		return []byte("127.0.0.1 localhost\n"), nil
	}
	r = checkHostsFile("myapp.test")
	if r.Status != Fail {
		t.Fatalf("expected Fail, got %v: %s", r.Status, r.Message)
	}
}

func TestCheckDaemon(t *testing.T) {
	restore := setupDoctorTest(t)
	defer restore()

	daemonIsRunningFn = func() bool { return false }
	r := checkDaemon()
	if r.Status != Warn {
		t.Fatalf("expected Warn, got %v: %s", r.Status, r.Message)
	}

	daemonIsRunningFn = func() bool { return true }
	daemonSendIPCFn = func(req daemon.Request) (*daemon.Response, error) {
		return &daemon.Response{OK: true}, nil
	}
	r = checkDaemon()
	if r.Status != Pass {
		t.Fatalf("expected Pass, got %v: %s", r.Status, r.Message)
	}
}

func TestCheckCACert(t *testing.T) {
	restore := setupDoctorTest(t)
	defer restore()

	readFileFn = func(path string) ([]byte, error) {
		return nil, os.ErrNotExist
	}
	r := checkCACert()
	if r.Status != Fail {
		t.Fatalf("expected Fail for missing cert, got %v: %s", r.Status, r.Message)
	}

	certPEM := generateTestCertPEM(t, time.Now().Add(365*24*time.Hour))
	readFileFn = func(path string) ([]byte, error) { return certPEM, nil }
	r = checkCACert()
	if r.Status != Pass {
		t.Fatalf("expected Pass for valid cert, got %v: %s", r.Status, r.Message)
	}

	expiringPEM := generateTestCertPEM(t, time.Now().Add(10*24*time.Hour))
	readFileFn = func(path string) ([]byte, error) { return expiringPEM, nil }
	r = checkCACert()
	if r.Status != Warn {
		t.Fatalf("expected Warn for expiring cert, got %v: %s", r.Status, r.Message)
	}

	expiredPEM := generateTestCertPEM(t, time.Now().Add(-1*time.Hour))
	readFileFn = func(path string) ([]byte, error) { return expiredPEM, nil }
	r = checkCACert()
	if r.Status != Fail {
		t.Fatalf("expected Fail for expired cert, got %v: %s", r.Status, r.Message)
	}
}

func TestCheckLeafCert(t *testing.T) {
	restore := setupDoctorTest(t)
	defer restore()

	certPEM := generateTestCertPEM(t, time.Now().Add(365*24*time.Hour))
	readFileFn = func(path string) ([]byte, error) { return certPEM, nil }
	r := checkLeafCert("myapp.test")
	if r.Status != Pass {
		t.Fatalf("expected Pass, got %v: %s", r.Status, r.Message)
	}

	readFileFn = func(path string) ([]byte, error) { return nil, os.ErrNotExist }
	r = checkLeafCert("myapp.test")
	if r.Status != Fail {
		t.Fatalf("expected Fail for missing cert, got %v: %s", r.Status, r.Message)
	}
}

func TestRun(t *testing.T) {
	restore := setupDoctorTest(t)
	defer restore()

	cfg := &config.Config{
		Domains: []config.Domain{{Name: "myapp", Port: 3000}},
	}
	configLoadFn = func() (*config.Config, error) { return cfg, nil }
	readFileFn = func(path string) ([]byte, error) { return nil, os.ErrNotExist }
	daemonIsRunningFn = func() bool { return false }
	newPortFwdFn = func() system.PortForwarder { return &mockPortFwd{enabled: false, loaded: false} }

	report := Run()
	if len(report.Results) == 0 {
		t.Fatal("expected at least one check result")
	}
}

func generateTestCertPEM(t *testing.T, notAfter time.Time) []byte {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     notAfter,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
}

func TestCheckCACertWriteAndRead(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := config.Init(); err != nil {
		t.Fatalf("config.Init: %v", err)
	}

	caDir := filepath.Join(home, ".slim", "ca")
	if err := os.MkdirAll(caDir, 0700); err != nil {
		t.Fatal(err)
	}

	certPEM := generateTestCertPEM(t, time.Now().Add(365*24*time.Hour))
	if err := os.WriteFile(filepath.Join(caDir, "rootCA.pem"), certPEM, 0644); err != nil {
		t.Fatal(err)
	}

	prevReadFile := readFileFn
	defer func() { readFileFn = prevReadFile }()
	readFileFn = os.ReadFile

	r := checkCACert()
	if r.Status != Pass {
		t.Fatalf("expected Pass, got %v: %s", r.Status, r.Message)
	}
}
