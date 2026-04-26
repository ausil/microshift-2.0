package certs

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/ausil/microshift-2.0/pkg/config"
)

// CertificateAuthority holds a CA key pair and its PEM-encoded form.
type CertificateAuthority struct {
	Key     *rsa.PrivateKey
	Cert    *x509.Certificate
	KeyPEM  []byte
	CertPEM []byte
}

// Certificate holds a signed certificate and its PEM-encoded form.
type Certificate struct {
	Key     *rsa.PrivateKey
	Cert    *x509.Certificate
	KeyPEM  []byte
	CertPEM []byte
}

// ClusterCerts contains all certificates needed for a MicroShift cluster.
type ClusterCerts struct {
	RootCA                     *CertificateAuthority
	APIServerCert              *Certificate
	APIServerKubeletClientCert *Certificate
	ControllerManagerCert      *Certificate
	SchedulerCert              *Certificate
	AdminCert                  *Certificate
	FrontProxyCA               *CertificateAuthority
	FrontProxyCert             *Certificate
	ServiceAccountKey          struct {
		PublicKeyPEM  []byte
		PrivateKeyPEM []byte
	}
}

// NewSelfSignedCA generates a new self-signed CA with a 2048-bit RSA key.
func NewSelfSignedCA(commonName string, validFor time.Duration) (*CertificateAuthority, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generating CA key: %w", err)
	}

	serial, err := randomSerial()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: commonName,
		},
		NotBefore:             now,
		NotAfter:              now.Add(validFor),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("creating CA certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, fmt.Errorf("parsing CA certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	return &CertificateAuthority{
		Key:     key,
		Cert:    cert,
		KeyPEM:  keyPEM,
		CertPEM: certPEM,
	}, nil
}

// NewSignedCert creates a certificate signed by the given CA. If isServer is
// true, the certificate will have server auth extended key usage; otherwise it
// will have client auth.
func NewSignedCert(ca *CertificateAuthority, commonName string, orgs []string, dnsNames []string, ips []net.IP, validFor time.Duration, isServer bool) (*Certificate, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generating key: %w", err)
	}

	serial, err := randomSerial()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: orgs,
		},
		NotBefore:             now,
		NotAfter:              now.Add(validFor),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		DNSNames:              dnsNames,
		IPAddresses:           ips,
	}

	if isServer {
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	} else {
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, ca.Cert, &key.PublicKey, ca.Key)
	if err != nil {
		return nil, fmt.Errorf("creating certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, fmt.Errorf("parsing certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	return &Certificate{
		Key:     key,
		Cert:    cert,
		KeyPEM:  keyPEM,
		CertPEM: certPEM,
	}, nil
}

// NewServiceAccountKeyPair generates an RSA 2048-bit key pair for service
// account token signing.
func NewServiceAccountKeyPair() (publicKeyPEM, privateKeyPEM []byte, err error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("generating service account key: %w", err)
	}

	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("marshaling public key: %w", err)
	}

	publicKeyPEM = pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	privateKeyPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	return publicKeyPEM, privateKeyPEM, nil
}

// WriteCertAndKey writes PEM-encoded certificate and key files to the given
// directory. The certificate is written as name.crt (mode 0644) and the key
// as name.key (mode 0600).
func WriteCertAndKey(dir, name string, certPEM, keyPEM []byte) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	certPath := filepath.Join(dir, name+".crt")
	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		return fmt.Errorf("writing cert %s: %w", certPath, err)
	}

	keyPath := filepath.Join(dir, name+".key")
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return fmt.Errorf("writing key %s: %w", keyPath, err)
	}

	return nil
}

// GenerateAllCerts generates all TLS certificates needed for the cluster.
func GenerateAllCerts(cfg *config.Config) (*ClusterCerts, error) {
	validFor := 365 * 24 * time.Hour // 1 year

	cc := &ClusterCerts{}

	// Root CA
	rootCA, err := NewSelfSignedCA("microshift-root-ca", validFor)
	if err != nil {
		return nil, fmt.Errorf("generating root CA: %w", err)
	}
	cc.RootCA = rootCA

	// Compute the first IP of ServiceCIDR for API server SANs.
	_, svcNet, err := net.ParseCIDR(cfg.ServiceCIDR)
	if err != nil {
		return nil, fmt.Errorf("parsing ServiceCIDR: %w", err)
	}
	svcFirstIP := make(net.IP, len(svcNet.IP))
	copy(svcFirstIP, svcNet.IP)
	svcFirstIP = svcFirstIP.To4()
	if svcFirstIP != nil {
		svcFirstIP[3]++
	}

	// API server SANs
	apiServerDNS := []string{
		"kubernetes",
		"kubernetes.default",
		"kubernetes.default.svc",
		"localhost",
	}
	apiServerIPs := []net.IP{
		net.ParseIP("127.0.0.1"),
		svcFirstIP,
	}
	if cfg.NodeIP != "" {
		apiServerIPs = append(apiServerIPs, net.ParseIP(cfg.NodeIP))
	}

	// API server cert (server)
	apiServerCert, err := NewSignedCert(rootCA, "kube-apiserver", nil, apiServerDNS, apiServerIPs, validFor, true)
	if err != nil {
		return nil, fmt.Errorf("generating apiserver cert: %w", err)
	}
	cc.APIServerCert = apiServerCert

	// API server kubelet client cert (client)
	kubeletClientCert, err := NewSignedCert(rootCA, "kube-apiserver-kubelet-client", []string{"system:masters"}, nil, nil, validFor, false)
	if err != nil {
		return nil, fmt.Errorf("generating apiserver-kubelet-client cert: %w", err)
	}
	cc.APIServerKubeletClientCert = kubeletClientCert

	// Controller manager cert (client)
	cmCert, err := NewSignedCert(rootCA, "system:kube-controller-manager", nil, nil, nil, validFor, false)
	if err != nil {
		return nil, fmt.Errorf("generating controller-manager cert: %w", err)
	}
	cc.ControllerManagerCert = cmCert

	// Scheduler cert (client)
	schedCert, err := NewSignedCert(rootCA, "system:kube-scheduler", nil, nil, nil, validFor, false)
	if err != nil {
		return nil, fmt.Errorf("generating scheduler cert: %w", err)
	}
	cc.SchedulerCert = schedCert

	// Admin cert (client)
	adminCert, err := NewSignedCert(rootCA, "kubernetes-admin", []string{"system:masters"}, nil, nil, validFor, false)
	if err != nil {
		return nil, fmt.Errorf("generating admin cert: %w", err)
	}
	cc.AdminCert = adminCert

	// Front proxy CA
	fpCA, err := NewSelfSignedCA("microshift-front-proxy-ca", validFor)
	if err != nil {
		return nil, fmt.Errorf("generating front-proxy CA: %w", err)
	}
	cc.FrontProxyCA = fpCA

	// Front proxy cert (client)
	fpCert, err := NewSignedCert(fpCA, "front-proxy-client", nil, nil, nil, validFor, false)
	if err != nil {
		return nil, fmt.Errorf("generating front-proxy cert: %w", err)
	}
	cc.FrontProxyCert = fpCert

	// Service account key pair
	pubPEM, privPEM, err := NewServiceAccountKeyPair()
	if err != nil {
		return nil, fmt.Errorf("generating service account keys: %w", err)
	}
	cc.ServiceAccountKey.PublicKeyPEM = pubPEM
	cc.ServiceAccountKey.PrivateKeyPEM = privPEM

	return cc, nil
}

// WriteAllCerts writes all cluster certificates to the cert directory on disk.
func WriteAllCerts(cfg *config.Config, cc *ClusterCerts) error {
	certDir := cfg.CertDir()

	if err := WriteCertAndKey(certDir, "ca", cc.RootCA.CertPEM, cc.RootCA.KeyPEM); err != nil {
		return err
	}
	if err := WriteCertAndKey(certDir, "apiserver", cc.APIServerCert.CertPEM, cc.APIServerCert.KeyPEM); err != nil {
		return err
	}
	if err := WriteCertAndKey(certDir, "apiserver-kubelet-client", cc.APIServerKubeletClientCert.CertPEM, cc.APIServerKubeletClientCert.KeyPEM); err != nil {
		return err
	}
	if err := WriteCertAndKey(certDir, "controller-manager", cc.ControllerManagerCert.CertPEM, cc.ControllerManagerCert.KeyPEM); err != nil {
		return err
	}
	if err := WriteCertAndKey(certDir, "scheduler", cc.SchedulerCert.CertPEM, cc.SchedulerCert.KeyPEM); err != nil {
		return err
	}
	if err := WriteCertAndKey(certDir, "admin", cc.AdminCert.CertPEM, cc.AdminCert.KeyPEM); err != nil {
		return err
	}
	if err := WriteCertAndKey(certDir, "front-proxy-ca", cc.FrontProxyCA.CertPEM, cc.FrontProxyCA.KeyPEM); err != nil {
		return err
	}
	if err := WriteCertAndKey(certDir, "front-proxy-client", cc.FrontProxyCert.CertPEM, cc.FrontProxyCert.KeyPEM); err != nil {
		return err
	}

	if err := os.MkdirAll(certDir, 0755); err != nil {
		return fmt.Errorf("creating cert dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(certDir, "sa.pub"), cc.ServiceAccountKey.PublicKeyPEM, 0644); err != nil {
		return fmt.Errorf("writing sa.pub: %w", err)
	}
	if err := os.WriteFile(filepath.Join(certDir, "sa.key"), cc.ServiceAccountKey.PrivateKeyPEM, 0600); err != nil {
		return fmt.Errorf("writing sa.key: %w", err)
	}

	return nil
}

func randomSerial() (*big.Int, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("generating serial number: %w", err)
	}
	return serial, nil
}
