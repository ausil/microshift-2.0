package services

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ausil/microshift-2.0/pkg/config"
)

// GenerateEtcdConfig writes the etcd configuration files:
//   - etcd.yaml  — etcd configuration
//   - etcd.env   — environment file with TLS cert paths
func GenerateEtcdConfig(cfg *config.Config) error {
	configDir := cfg.ComponentConfigDir()
	certDir := cfg.CertDir()

	// etcd.yaml
	etcdYAML := fmt.Sprintf(`name: %s
data-dir: %s
listen-client-urls: https://127.0.0.1:2379
advertise-client-urls: https://127.0.0.1:2379
listen-peer-urls: https://127.0.0.1:2380
`, cfg.ClusterName, cfg.EtcdDataDir())

	if err := os.WriteFile(filepath.Join(configDir, "etcd.yaml"), []byte(etcdYAML), 0644); err != nil {
		return fmt.Errorf("writing etcd.yaml: %w", err)
	}

	// etcd.env — environment file consumed by the systemd unit
	// Single-node: reuse apiserver cert for etcd TLS
	etcdEnv := fmt.Sprintf(`ETCD_CONFIG_FILE="%s"
ETCD_CERT_FILE="%s"
ETCD_KEY_FILE="%s"
ETCD_TRUSTED_CA_FILE="%s"
ETCD_CLIENT_CERT_AUTH="true"
ETCD_PEER_CERT_FILE="%s"
ETCD_PEER_KEY_FILE="%s"
ETCD_PEER_TRUSTED_CA_FILE="%s"
ETCD_PEER_CLIENT_CERT_AUTH="true"
`,
		filepath.Join(configDir, "etcd.yaml"),
		filepath.Join(certDir, "apiserver.crt"),
		filepath.Join(certDir, "apiserver.key"),
		filepath.Join(certDir, "ca.crt"),
		filepath.Join(certDir, "apiserver.crt"),
		filepath.Join(certDir, "apiserver.key"),
		filepath.Join(certDir, "ca.crt"),
	)

	if err := os.WriteFile(filepath.Join(configDir, "etcd.env"), []byte(etcdEnv), 0644); err != nil {
		return fmt.Errorf("writing etcd.env: %w", err)
	}

	return nil
}
