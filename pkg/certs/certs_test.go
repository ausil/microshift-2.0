package certs

import (
	"crypto/x509"
	"encoding/pem"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ausil/microshift-2.0/pkg/config"
)

func TestNewSelfSignedCA(t *testing.T) {
	ca, err := NewSelfSignedCA("test-ca", 24*time.Hour)
	if err != nil {
		t.Fatalf("NewSelfSignedCA failed: %v", err)
	}

	if ca.Key == nil {
		t.Fatal("CA key is nil")
	}
	if ca.Cert == nil {
		t.Fatal("CA cert is nil")
	}
	if !ca.Cert.IsCA {
		t.Error("CA cert is not marked as CA")
	}
	if ca.Cert.Subject.CommonName != "test-ca" {
		t.Errorf("expected CN=test-ca, got %q", ca.Cert.Subject.CommonName)
	}
	if len(ca.CertPEM) == 0 {
		t.Error("CA CertPEM is empty")
	}
	if len(ca.KeyPEM) == 0 {
		t.Error("CA KeyPEM is empty")
	}

	// Verify the PEM can be decoded.
	block, _ := pem.Decode(ca.CertPEM)
	if block == nil {
		t.Fatal("failed to decode CA CertPEM")
	}
	if block.Type != "CERTIFICATE" {
		t.Errorf("expected PEM type CERTIFICATE, got %q", block.Type)
	}
}

func TestNewSignedCertServer(t *testing.T) {
	ca, err := NewSelfSignedCA("test-ca", 24*time.Hour)
	if err != nil {
		t.Fatalf("NewSelfSignedCA failed: %v", err)
	}

	dnsNames := []string{"kubernetes", "kubernetes.default"}
	ips := []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("10.96.0.1")}

	cert, err := NewSignedCert(ca, "kube-apiserver", nil, dnsNames, ips, 24*time.Hour, true)
	if err != nil {
		t.Fatalf("NewSignedCert failed: %v", err)
	}

	if cert.Cert.Subject.CommonName != "kube-apiserver" {
		t.Errorf("expected CN=kube-apiserver, got %q", cert.Cert.Subject.CommonName)
	}
	if len(cert.Cert.DNSNames) != 2 {
		t.Errorf("expected 2 DNS names, got %d", len(cert.Cert.DNSNames))
	}
	if len(cert.Cert.IPAddresses) != 2 {
		t.Errorf("expected 2 IP addresses, got %d", len(cert.Cert.IPAddresses))
	}

	// Should have server auth key usage.
	hasServerAuth := false
	for _, u := range cert.Cert.ExtKeyUsage {
		if u == x509.ExtKeyUsageServerAuth {
			hasServerAuth = true
		}
	}
	if !hasServerAuth {
		t.Error("server cert missing ExtKeyUsageServerAuth")
	}

	// Verify the cert is signed by the CA.
	pool := x509.NewCertPool()
	pool.AddCert(ca.Cert)
	opts := x509.VerifyOptions{
		Roots:     pool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	if _, err := cert.Cert.Verify(opts); err != nil {
		t.Errorf("cert verification failed: %v", err)
	}
}

func TestNewSignedCertClient(t *testing.T) {
	ca, err := NewSelfSignedCA("test-ca", 24*time.Hour)
	if err != nil {
		t.Fatalf("NewSelfSignedCA failed: %v", err)
	}

	cert, err := NewSignedCert(ca, "system:kube-controller-manager", nil, nil, nil, 24*time.Hour, false)
	if err != nil {
		t.Fatalf("NewSignedCert failed: %v", err)
	}

	// Should have client auth key usage.
	hasClientAuth := false
	for _, u := range cert.Cert.ExtKeyUsage {
		if u == x509.ExtKeyUsageClientAuth {
			hasClientAuth = true
		}
	}
	if !hasClientAuth {
		t.Error("client cert missing ExtKeyUsageClientAuth")
	}

	// Verify the cert is signed by the CA.
	pool := x509.NewCertPool()
	pool.AddCert(ca.Cert)
	opts := x509.VerifyOptions{
		Roots:     pool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	if _, err := cert.Cert.Verify(opts); err != nil {
		t.Errorf("cert verification failed: %v", err)
	}
}

func TestNewSignedCertWithOrgs(t *testing.T) {
	ca, err := NewSelfSignedCA("test-ca", 24*time.Hour)
	if err != nil {
		t.Fatalf("NewSelfSignedCA failed: %v", err)
	}

	cert, err := NewSignedCert(ca, "kubernetes-admin", []string{"system:masters"}, nil, nil, 24*time.Hour, false)
	if err != nil {
		t.Fatalf("NewSignedCert failed: %v", err)
	}

	if len(cert.Cert.Subject.Organization) != 1 || cert.Cert.Subject.Organization[0] != "system:masters" {
		t.Errorf("expected Org=[system:masters], got %v", cert.Cert.Subject.Organization)
	}
}

func TestNewServiceAccountKeyPair(t *testing.T) {
	pubPEM, privPEM, err := NewServiceAccountKeyPair()
	if err != nil {
		t.Fatalf("NewServiceAccountKeyPair failed: %v", err)
	}

	if len(pubPEM) == 0 {
		t.Error("public key PEM is empty")
	}
	if len(privPEM) == 0 {
		t.Error("private key PEM is empty")
	}

	pubBlock, _ := pem.Decode(pubPEM)
	if pubBlock == nil {
		t.Fatal("failed to decode public key PEM")
	}
	if pubBlock.Type != "PUBLIC KEY" {
		t.Errorf("expected PEM type PUBLIC KEY, got %q", pubBlock.Type)
	}

	privBlock, _ := pem.Decode(privPEM)
	if privBlock == nil {
		t.Fatal("failed to decode private key PEM")
	}
	if privBlock.Type != "RSA PRIVATE KEY" {
		t.Errorf("expected PEM type RSA PRIVATE KEY, got %q", privBlock.Type)
	}
}

func TestWriteCertAndKey(t *testing.T) {
	dir := t.TempDir()

	certPEM := []byte("---CERT---")
	keyPEM := []byte("---KEY---")

	if err := WriteCertAndKey(dir, "test", certPEM, keyPEM); err != nil {
		t.Fatalf("WriteCertAndKey failed: %v", err)
	}

	certPath := filepath.Join(dir, "test.crt")
	keyPath := filepath.Join(dir, "test.key")

	certData, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("reading cert file: %v", err)
	}
	if string(certData) != "---CERT---" {
		t.Errorf("cert content mismatch: %q", string(certData))
	}

	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("reading key file: %v", err)
	}
	if string(keyData) != "---KEY---" {
		t.Errorf("key content mismatch: %q", string(keyData))
	}

	// Verify key file permissions.
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("stat key file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("expected key file mode 0600, got %o", perm)
	}

	// Verify cert file permissions.
	info, err = os.Stat(certPath)
	if err != nil {
		t.Fatalf("stat cert file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0644 {
		t.Errorf("expected cert file mode 0644, got %o", perm)
	}
}

func TestWriteCertAndKeyCreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sub", "dir")

	if err := WriteCertAndKey(dir, "test", []byte("cert"), []byte("key")); err != nil {
		t.Fatalf("WriteCertAndKey failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "test.crt")); err != nil {
		t.Errorf("cert file not created: %v", err)
	}
}

func TestGenerateAllCerts(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.NodeIP = "192.168.1.100"

	cc, err := GenerateAllCerts(cfg)
	if err != nil {
		t.Fatalf("GenerateAllCerts failed: %v", err)
	}

	if cc.RootCA == nil {
		t.Error("RootCA is nil")
	}
	if cc.APIServerCert == nil {
		t.Error("APIServerCert is nil")
	}
	if cc.APIServerKubeletClientCert == nil {
		t.Error("APIServerKubeletClientCert is nil")
	}
	if cc.ControllerManagerCert == nil {
		t.Error("ControllerManagerCert is nil")
	}
	if cc.SchedulerCert == nil {
		t.Error("SchedulerCert is nil")
	}
	if cc.AdminCert == nil {
		t.Error("AdminCert is nil")
	}
	if cc.FrontProxyCA == nil {
		t.Error("FrontProxyCA is nil")
	}
	if cc.FrontProxyCert == nil {
		t.Error("FrontProxyCert is nil")
	}
	if len(cc.ServiceAccountKey.PublicKeyPEM) == 0 {
		t.Error("ServiceAccountKey public key is empty")
	}
	if len(cc.ServiceAccountKey.PrivateKeyPEM) == 0 {
		t.Error("ServiceAccountKey private key is empty")
	}

	// Verify API server cert has the expected SANs.
	apiCert := cc.APIServerCert.Cert
	expectedDNS := map[string]bool{
		"kubernetes":             false,
		"kubernetes.default":     false,
		"kubernetes.default.svc": false,
		"localhost":              false,
	}
	for _, dns := range apiCert.DNSNames {
		if _, ok := expectedDNS[dns]; ok {
			expectedDNS[dns] = true
		}
	}
	for dns, found := range expectedDNS {
		if !found {
			t.Errorf("API server cert missing DNS SAN: %s", dns)
		}
	}

	// Check IP SANs include 127.0.0.1 and the NodeIP.
	foundLoopback := false
	foundNodeIP := false
	for _, ip := range apiCert.IPAddresses {
		if ip.Equal(net.ParseIP("127.0.0.1")) {
			foundLoopback = true
		}
		if ip.Equal(net.ParseIP("192.168.1.100")) {
			foundNodeIP = true
		}
	}
	if !foundLoopback {
		t.Error("API server cert missing 127.0.0.1 IP SAN")
	}
	if !foundNodeIP {
		t.Error("API server cert missing NodeIP SAN")
	}

	// Verify kubelet client cert.
	kc := cc.APIServerKubeletClientCert.Cert
	if kc.Subject.CommonName != "kube-apiserver-kubelet-client" {
		t.Errorf("kubelet client CN = %q", kc.Subject.CommonName)
	}
	if len(kc.Subject.Organization) == 0 || kc.Subject.Organization[0] != "system:masters" {
		t.Errorf("kubelet client Org = %v", kc.Subject.Organization)
	}

	// Verify admin cert.
	ac := cc.AdminCert.Cert
	if ac.Subject.CommonName != "kubernetes-admin" {
		t.Errorf("admin CN = %q", ac.Subject.CommonName)
	}
}
