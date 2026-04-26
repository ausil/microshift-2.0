package services

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ausil/microshift-2.0/pkg/config"
)

// GenerateAPIServerConfig writes the API server environment file:
//   - apiserver.env — KUBE_APISERVER_ARGS with all required flags
func GenerateAPIServerConfig(cfg *config.Config) error {
	configDir := cfg.ComponentConfigDir()
	certDir := cfg.CertDir()

	args := fmt.Sprintf(
		"--etcd-servers=https://127.0.0.1:2379 "+
			"--etcd-cafile=%s "+
			"--etcd-certfile=%s "+
			"--etcd-keyfile=%s "+
			"--tls-cert-file=%s "+
			"--tls-private-key-file=%s "+
			"--client-ca-file=%s "+
			"--service-account-key-file=%s "+
			"--service-account-signing-key-file=%s "+
			"--service-account-issuer=https://kubernetes.default.svc "+
			"--service-cluster-ip-range=%s "+
			"--secure-port=6443 "+
			"--bind-address=0.0.0.0 "+
			"--authorization-mode=Node,RBAC "+
			"--enable-admission-plugins=NodeRestriction "+
			"--kubelet-client-certificate=%s "+
			"--kubelet-client-key=%s "+
			"--requestheader-client-ca-file=%s "+
			"--allow-privileged=true",
		filepath.Join(certDir, "ca.crt"),
		filepath.Join(certDir, "apiserver.crt"),
		filepath.Join(certDir, "apiserver.key"),
		filepath.Join(certDir, "apiserver.crt"),
		filepath.Join(certDir, "apiserver.key"),
		filepath.Join(certDir, "ca.crt"),
		filepath.Join(certDir, "sa.pub"),
		filepath.Join(certDir, "sa.key"),
		cfg.ServiceCIDR,
		filepath.Join(certDir, "apiserver-kubelet-client.crt"),
		filepath.Join(certDir, "apiserver-kubelet-client.key"),
		filepath.Join(certDir, "front-proxy-ca.crt"),
	)

	env := fmt.Sprintf("KUBE_APISERVER_ARGS=\"%s\"\n", args)

	if err := os.WriteFile(filepath.Join(configDir, "apiserver.env"), []byte(env), 0644); err != nil {
		return fmt.Errorf("writing apiserver.env: %w", err)
	}

	return nil
}
