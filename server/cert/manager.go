// server/cert/manager.go
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
)

type CertManager struct {
	CaKey         *rsa.PrivateKey
	CaCert        *x509.Certificate
	CertDirectory string
	CertPool      *x509.CertPool
}

func NewCertManager(certDir string) (*CertManager, error) {
	if err := os.MkdirAll(certDir, 0755); err != nil {
		return nil, fmt.Errorf("creating cert directory: %v", err)
	}

	cm := &CertManager{
		CertDirectory: certDir,
		CertPool:      x509.NewCertPool(),
	}

	// Check if CA exists, if not create it
	if err := cm.initializeCA(); err != nil {
		return nil, err
	}

	return cm, nil
}

func (cm *CertManager) initializeCA() error {
	caKeyPath := filepath.Join(cm.CertDirectory, "ca-key.pem")
	caCertPath := filepath.Join(cm.CertDirectory, "ca-cert.pem")

	// Check if CA files exist
	if _, err := os.Stat(caKeyPath); os.IsNotExist(err) {
		// Generate new CA
		key, cert, err := cm.generateCA()
		if err != nil {
			return fmt.Errorf("generating CA: %v", err)
		}

		cm.CaKey = key
		cm.CaCert = cert

		// Save CA files
		if err := cm.savePEMKey(caKeyPath, key); err != nil {
			return fmt.Errorf("saving CA key: %v", err)
		}
		if err := cm.savePEMCert(caCertPath, cert); err != nil {
			return fmt.Errorf("saving CA cert: %v", err)
		}
	} else {
		// Load existing CA
		key, err := cm.loadPEMKey(caKeyPath)
		if err != nil {
			return fmt.Errorf("loading CA key: %v", err)
		}
		cert, err := cm.loadPEMCert(caCertPath)
		if err != nil {
			return fmt.Errorf("loading CA cert: %v", err)
		}

		cm.CaKey = key
		cm.CaCert = cert
	}

	// Add CA cert to pool
	if !cm.CertPool.AppendCertsFromPEM(cm.pemEncodeCert(cm.CaCert)) {
		return fmt.Errorf("failed to add CA cert to pool")
	}

	return nil
}

func (cm *CertManager) GenerateClientCert(id string) (string, string, error) {
	// Generate key
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", fmt.Errorf("generating key: %v", err)
	}

	// Generate certificate
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			CommonName: id,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour * 365),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, cm.CaCert, &key.PublicKey, cm.CaKey)
	if err != nil {
		return "", "", fmt.Errorf("creating certificate: %v", err)
	}

	// Save files
	certPath := filepath.Join(cm.CertDirectory, fmt.Sprintf("%s-cert.pem", id))
	keyPath := filepath.Join(cm.CertDirectory, fmt.Sprintf("%s-key.pem", id))

	if err := cm.savePEMKey(keyPath, key); err != nil {
		return "", "", fmt.Errorf("saving key: %v", err)
	}

	if err := cm.savePEMBytes(certPath, "CERTIFICATE", certBytes); err != nil {
		return "", "", fmt.Errorf("saving certificate: %v", err)
	}

	return certPath, keyPath, nil
}

func (cm *CertManager) GetCertPool() *x509.CertPool {
	return cm.CertPool
}

// Helper methods for PEM encoding/decoding
func (cm *CertManager) savePEMKey(path string, key *rsa.PrivateKey) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return pem.Encode(file, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
}

func (cm *CertManager) savePEMCert(path string, cert *x509.Certificate) error {
	return cm.savePEMBytes(path, "CERTIFICATE", cert.Raw)
}

func (cm *CertManager) savePEMBytes(path, pemType string, bytes []byte) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return pem.Encode(file, &pem.Block{
		Type:  pemType,
		Bytes: bytes,
	})
}

func (cm *CertManager) loadPEMKey(path string) (*rsa.PrivateKey, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(bytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func (cm *CertManager) loadPEMCert(path string) (*x509.Certificate, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(bytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	return x509.ParseCertificate(block.Bytes)
}

func (cm *CertManager) pemEncodeCert(cert *x509.Certificate) []byte {
	block := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}
	return pem.EncodeToMemory(block)
}

func (cm *CertManager) generateCA() (*rsa.PrivateKey, *x509.Certificate, error) {
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "Remote Management CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, nil, err
	}

	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, nil, err
	}

	return key, cert, nil
}
