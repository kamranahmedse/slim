package cert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kamranahmedse/slim/internal/config"
)

func TestCAAndLeafPathHelpers(t *testing.T) {
	home := initCertTestConfig(t)

	if got, want := CADir(), filepath.Join(home, ".slim", "ca"); got != want {
		t.Fatalf("CADir() = %q, want %q", got, want)
	}
	if got, want := CertsDir(), filepath.Join(home, ".slim", "certs"); got != want {
		t.Fatalf("CertsDir() = %q, want %q", got, want)
	}
	if got, want := CACertPath(), filepath.Join(home, ".slim", "ca", "rootCA.pem"); got != want {
		t.Fatalf("CACertPath() = %q, want %q", got, want)
	}
	if got, want := CAKeyPath(), filepath.Join(home, ".slim", "ca", "rootCA-key.pem"); got != want {
		t.Fatalf("CAKeyPath() = %q, want %q", got, want)
	}
	if got, want := LeafCertPath("myapp"), filepath.Join(home, ".slim", "certs", "myapp.pem"); got != want {
		t.Fatalf("LeafCertPath() = %q, want %q", got, want)
	}
	if got, want := LeafKeyPath("myapp"), filepath.Join(home, ".slim", "certs", "myapp-key.pem"); got != want {
		t.Fatalf("LeafKeyPath() = %q, want %q", got, want)
	}
}

func TestCAExistsAndLeafExists(t *testing.T) {
	initCertTestConfig(t)

	if CAExists() {
		t.Fatal("expected CAExists false before files are created")
	}
	if LeafExists("myapp") {
		t.Fatal("expected LeafExists false before files are created")
	}

	if err := os.MkdirAll(CADir(), 0700); err != nil {
		t.Fatalf("MkdirAll CADir: %v", err)
	}
	if err := os.WriteFile(CACertPath(), []byte("cert"), 0644); err != nil {
		t.Fatalf("WriteFile CACertPath: %v", err)
	}
	if err := os.WriteFile(CAKeyPath(), []byte("key"), 0600); err != nil {
		t.Fatalf("WriteFile CAKeyPath: %v", err)
	}
	if !CAExists() {
		t.Fatal("expected CAExists true when cert and key files exist")
	}

	if err := os.MkdirAll(CertsDir(), 0700); err != nil {
		t.Fatalf("MkdirAll CertsDir: %v", err)
	}
	if err := os.WriteFile(LeafCertPath("myapp"), []byte("cert"), 0644); err != nil {
		t.Fatalf("WriteFile LeafCertPath: %v", err)
	}
	if err := os.WriteFile(LeafKeyPath("myapp"), []byte("key"), 0600); err != nil {
		t.Fatalf("WriteFile LeafKeyPath: %v", err)
	}
	if !LeafExists("myapp") {
		t.Fatal("expected LeafExists true when cert and key files exist")
	}
}

func TestGenerateCAAndLoadCA(t *testing.T) {
	initCertTestConfig(t)

	if err := GenerateCA(); err != nil {
		t.Fatalf("GenerateCA: %v", err)
	}
	if !CAExists() {
		t.Fatal("expected CAExists true after GenerateCA")
	}

	cert, key, err := LoadCA()
	if err != nil {
		t.Fatalf("LoadCA: %v", err)
	}
	if cert == nil || key == nil {
		t.Fatal("expected non-nil cert and key")
	}
	if !cert.IsCA {
		t.Fatal("expected loaded certificate to be a CA cert")
	}
}

func TestEnsureLeafCertAndLoadLeafTLS(t *testing.T) {
	initCertTestConfig(t)

	if err := GenerateCA(); err != nil {
		t.Fatalf("GenerateCA: %v", err)
	}

	if err := EnsureLeafCert("myapp"); err != nil {
		t.Fatalf("EnsureLeafCert: %v", err)
	}
	if !LeafExists("myapp") {
		t.Fatal("expected leaf cert+key files to exist")
	}

	tlsCert, err := LoadLeafTLS("myapp")
	if err != nil {
		t.Fatalf("LoadLeafTLS: %v", err)
	}
	if tlsCert == nil || len(tlsCert.Certificate) == 0 {
		t.Fatal("expected loaded TLS certificate data")
	}

	cert, err := x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}
	if cert.Subject.CommonName != "myapp.local" {
		t.Fatalf("expected CN myapp.local, got %q", cert.Subject.CommonName)
	}
}

func TestLeafNeedsRenewal(t *testing.T) {
	initCertTestConfig(t)
	name := "renewal"

	if !leafNeedsRenewal(name) {
		t.Fatal("expected renewal=true when cert file is missing")
	}

	if err := os.MkdirAll(CertsDir(), 0700); err != nil {
		t.Fatalf("MkdirAll CertsDir: %v", err)
	}

	if err := os.WriteFile(LeafCertPath(name), []byte("not pem"), 0644); err != nil {
		t.Fatalf("WriteFile invalid PEM: %v", err)
	}
	if !leafNeedsRenewal(name) {
		t.Fatal("expected renewal=true for invalid PEM cert")
	}

	if err := writeLeafCertPEM(name, "rsa", time.Now().Add(90*24*time.Hour)); err != nil {
		t.Fatalf("writeLeafCertPEM rsa: %v", err)
	}
	if !leafNeedsRenewal(name) {
		t.Fatal("expected renewal=true for non-ECDSA cert")
	}

	if err := writeLeafCertPEM(name, "ecdsa", time.Now().Add(10*24*time.Hour)); err != nil {
		t.Fatalf("writeLeafCertPEM ecdsa expiring: %v", err)
	}
	if !leafNeedsRenewal(name) {
		t.Fatal("expected renewal=true for cert expiring soon")
	}

	if err := writeLeafCertPEM(name, "ecdsa", time.Now().Add(90*24*time.Hour)); err != nil {
		t.Fatalf("writeLeafCertPEM ecdsa healthy: %v", err)
	}
	if leafNeedsRenewal(name) {
		t.Fatal("expected renewal=false for valid ECDSA cert with sufficient lifetime")
	}
}

func initCertTestConfig(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := config.Init(); err != nil {
		t.Fatalf("config.Init: %v", err)
	}
	return home
}

func writeLeafCertPEM(name string, keyType string, notAfter time.Time) error {
	var (
		publicKey  any
		privateKey any
	)

	switch keyType {
	case "rsa":
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return err
		}
		privateKey = key
		publicKey = &key.PublicKey
	case "ecdsa":
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return err
		}
		privateKey = key
		publicKey = &key.PublicKey
	default:
		return fmt.Errorf("unknown key type: %s", keyType)
	}

	serial, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		return err
	}
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: name + ".local",
		},
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter:  notAfter,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, publicKey, privateKey)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(CertsDir(), 0700); err != nil {
		return err
	}

	f, err := os.OpenFile(LeafCertPath(name), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	return pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: der})
}
