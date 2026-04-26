package services

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ausil/microshift-2.0/pkg/config"
)

// GenerateControllerManagerConfig writes the controller manager environment file:
//   - controller-manager.env — KUBE_CONTROLLER_MANAGER_ARGS with all required flags
func GenerateControllerManagerConfig(cfg *config.Config) error {
	configDir := cfg.ComponentConfigDir()
	certDir := cfg.CertDir()
	kubeconfigDir := cfg.KubeconfigDir()

	args := fmt.Sprintf(
		"--kubeconfig=%s "+
			"--cluster-cidr=%s "+
			"--service-cluster-ip-range=%s "+
			"--cluster-signing-cert-file=%s "+
			"--cluster-signing-key-file=%s "+
			"--root-ca-file=%s "+
			"--service-account-private-key-file=%s "+
			"--use-service-account-credentials=true "+
			"--controllers=*,bootstrapsigner,tokencleaner",
		filepath.Join(kubeconfigDir, "controller-manager.kubeconfig"),
		cfg.ClusterCIDR,
		cfg.ServiceCIDR,
		filepath.Join(certDir, "ca.crt"),
		filepath.Join(certDir, "ca.key"),
		filepath.Join(certDir, "ca.crt"),
		filepath.Join(certDir, "sa.key"),
	)

	env := fmt.Sprintf("KUBE_CONTROLLER_MANAGER_ARGS=\"%s\"\n", args)

	if err := os.WriteFile(filepath.Join(configDir, "controller-manager.env"), []byte(env), 0644); err != nil {
		return fmt.Errorf("writing controller-manager.env: %w", err)
	}

	return nil
}
