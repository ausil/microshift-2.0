package services

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ausil/microshift-2.0/pkg/config"
)

// GenerateKubeletConfig writes the kubelet configuration files:
//   - kubelet-config.yaml — KubeletConfiguration
//   - kubelet.env         — KUBELET_ARGS with required flags
func GenerateKubeletConfig(cfg *config.Config) error {
	configDir := cfg.ComponentConfigDir()
	certDir := cfg.CertDir()
	kubeconfigDir := cfg.KubeconfigDir()

	// kubelet-config.yaml
	kubeletConfigYAML := fmt.Sprintf(`apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
authentication:
  x509:
    clientCAFile: %s
  webhook:
    enabled: true
authorization:
  mode: Webhook
clusterDNS:
  - %s
clusterDomain: cluster.local
containerRuntimeEndpoint: unix:///var/run/crio/crio.sock
cgroupDriver: systemd
`, filepath.Join(certDir, "ca.crt"), cfg.DNSServiceIP())

	kubeletConfigPath := filepath.Join(configDir, "kubelet-config.yaml")
	if err := os.WriteFile(kubeletConfigPath, []byte(kubeletConfigYAML), 0644); err != nil {
		return fmt.Errorf("writing kubelet-config.yaml: %w", err)
	}

	// kubelet.env
	args := fmt.Sprintf(
		"--config=%s "+
			"--kubeconfig=%s "+
			"--hostname-override=%s "+
			"--node-ip=%s "+
			"--container-runtime-endpoint=unix:///var/run/crio/crio.sock",
		kubeletConfigPath,
		filepath.Join(kubeconfigDir, "kubelet.kubeconfig"),
		cfg.NodeIP,
		cfg.NodeIP,
	)

	env := fmt.Sprintf("KUBELET_ARGS=\"%s\"\n", args)

	if err := os.WriteFile(filepath.Join(configDir, "kubelet.env"), []byte(env), 0644); err != nil {
		return fmt.Errorf("writing kubelet.env: %w", err)
	}

	return nil
}
