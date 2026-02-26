package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/kamrify/localname/internal/config"
)

func CADir() string {
	return filepath.Join(config.Dir(), "ca")
}

func CACertPath() string {
	return filepath.Join(CADir(), "rootCA.pem")
}

func CAKeyPath() string {
	return filepath.Join(CADir(), "rootCA-key.pem")
}

func CAExists() bool {
	_, certErr := os.Stat(CACertPath())
	_, keyErr := os.Stat(CAKeyPath())
	return certErr == nil && keyErr == nil
}

func GenerateCA() error {
	if err := os.MkdirAll(CADir(), 0700); err != nil {
		return fmt.Errorf("creating CA dir: %w", err)
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("generating CA key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("generating serial: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"localname"},
			CommonName:   "localname Root CA",
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return fmt.Errorf("creating CA cert: %w", err)
	}

	certFile, err := os.OpenFile(CACertPath(), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer certFile.Close()
	pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	keyFile, err := os.OpenFile(CAKeyPath(), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer keyFile.Close()
	pem.Encode(keyFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	return nil
}

func LoadCA() (*x509.Certificate, *rsa.PrivateKey, error) {
	certPEM, err := os.ReadFile(CACertPath())
	if err != nil {
		return nil, nil, fmt.Errorf("reading CA cert: %w", err)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, nil, fmt.Errorf("invalid CA cert PEM")
	}

	caCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing CA cert: %w", err)
	}

	keyPEM, err := os.ReadFile(CAKeyPath())
	if err != nil {
		return nil, nil, fmt.Errorf("reading CA key: %w", err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, nil, fmt.Errorf("invalid CA key PEM")
	}

	caKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing CA key: %w", err)
	}

	return caCert, caKey, nil
}
