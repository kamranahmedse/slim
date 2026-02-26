package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/kamranahmedse/localname/internal/config"
)

func CertsDir() string {
	return filepath.Join(config.Dir(), "certs")
}

func LeafCertPath(name string) string {
	return filepath.Join(CertsDir(), name+".pem")
}

func LeafKeyPath(name string) string {
	return filepath.Join(CertsDir(), name+"-key.pem")
}

func LeafExists(name string) bool {
	_, certErr := os.Stat(LeafCertPath(name))
	_, keyErr := os.Stat(LeafKeyPath(name))
	return certErr == nil && keyErr == nil
}

func GenerateLeafCert(name string) error {
	caCert, caKey, err := LoadCA()
	if err != nil {
		return fmt.Errorf("loading CA: %w", err)
	}

	if err := os.MkdirAll(CertsDir(), 0700); err != nil {
		return fmt.Errorf("creating certs dir: %w", err)
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("generating leaf key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("generating serial: %w", err)
	}

	hostname := name + ".local"
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: hostname,
		},
		DNSNames:    []string{hostname},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
		NotBefore:   time.Now().Add(-1 * time.Hour),
		NotAfter:    time.Now().Add(825 * 24 * time.Hour), // ~2 years, under Apple's limit
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, caCert, &key.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("creating leaf cert: %w", err)
	}

	certFile, err := os.OpenFile(LeafCertPath(name), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer certFile.Close()
	if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return fmt.Errorf("writing leaf cert: %w", err)
	}

	keyFile, err := os.OpenFile(LeafKeyPath(name), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer keyFile.Close()
	if err := pem.Encode(keyFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}); err != nil {
		return fmt.Errorf("writing leaf key: %w", err)
	}

	return nil
}

func LoadLeafTLS(name string) (*tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(LeafCertPath(name), LeafKeyPath(name))
	if err != nil {
		return nil, fmt.Errorf("loading cert for %s: %w", name, err)
	}
	return &cert, nil
}

func EnsureLeafCert(name string) error {
	if LeafExists(name) && !leafExpiringSoon(name) {
		return nil
	}
	return GenerateLeafCert(name)
}

func leafExpiringSoon(name string) bool {
	data, err := os.ReadFile(LeafCertPath(name))
	if err != nil {
		return true
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return true
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return true
	}

	return time.Until(cert.NotAfter) < 30*24*time.Hour
}
