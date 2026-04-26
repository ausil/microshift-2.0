package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ausil/microshift-2.0/pkg/config"
)

// setupTestConfig creates a temporary directory structure and returns a
// config.Config whose DataDir points at it.  The caller does not need to
// clean up — t.TempDir handles that.
func setupTestConfig(t *testing.T) *config.Config {
	t.Helper()
	dir := t.TempDir()

	cfg := config.NewDefaultConfig()
	cfg.DataDir = dir
	cfg.NodeIP = "192.168.1.100"

	// Create the subdirectories the generators write into.
	for _, sub := range []string{cfg.ComponentConfigDir(), cfg.CertDir(), cfg.KubeconfigDir()} {
		if err := os.MkdirAll(sub, 0755); err != nil {
			t.Fatalf("creating dir %s: %v", sub, err)
		}
	}

	return cfg
}

// fileContains is a small helper that reads a file and asserts it contains
// every string in wants.
func fileContains(t *testing.T, path string, wants ...string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	content := string(data)
	for _, w := range wants {
		if !strings.Contains(content, w) {
			t.Errorf("%s: expected to contain %q, got:\n%s", filepath.Base(path), w, content)
		}
	}
}

// ---------- etcd ----------

func TestGenerateEtcdConfig(t *testing.T) {
	cfg := setupTestConfig(t)

	if err := GenerateEtcdConfig(cfg); err != nil {
		t.Fatalf("GenerateEtcdConfig: %v", err)
	}

	configDir := cfg.ComponentConfigDir()

	// etcd.yaml must exist and contain key settings.
	etcdYAML := filepath.Join(configDir, "etcd.yaml")
	fileContains(t, etcdYAML,
		"name: microshift",
		"data-dir:",
		"listen-client-urls: https://127.0.0.1:2379",
		"advertise-client-urls: https://127.0.0.1:2379",
		"listen-peer-urls: https://127.0.0.1:2380",
	)

	// etcd.env must exist and contain cert variables.
	etcdEnv := filepath.Join(configDir, "etcd.env")
	fileContains(t, etcdEnv,
		"ETCD_CERT_FILE=",
		"ETCD_KEY_FILE=",
		"ETCD_TRUSTED_CA_FILE=",
		"ETCD_CLIENT_CERT_AUTH=\"true\"",
		"ETCD_PEER_CERT_FILE=",
		"ETCD_PEER_KEY_FILE=",
		"ETCD_PEER_TRUSTED_CA_FILE=",
		"ETCD_PEER_CLIENT_CERT_AUTH=\"true\"",
		"apiserver.crt",
		"apiserver.key",
		"ca.crt",
	)
}

// ---------- apiserver ----------

func TestGenerateAPIServerConfig(t *testing.T) {
	cfg := setupTestConfig(t)

	if err := GenerateAPIServerConfig(cfg); err != nil {
		t.Fatalf("GenerateAPIServerConfig: %v", err)
	}

	envPath := filepath.Join(cfg.ComponentConfigDir(), "apiserver.env")
	fileContains(t, envPath,
		"KUBE_APISERVER_ARGS=",
		"--etcd-servers=https://127.0.0.1:2379",
		"--etcd-cafile=",
		"--etcd-certfile=",
		"--etcd-keyfile=",
		"--tls-cert-file=",
		"--tls-private-key-file=",
		"--client-ca-file=",
		"--service-account-key-file=",
		"--service-account-signing-key-file=",
		"--service-account-issuer=",
		"--service-cluster-ip-range=10.96.0.0/16",
		"--secure-port=6443",
		"--bind-address=0.0.0.0",
		"--authorization-mode=Node,RBAC",
		"--enable-admission-plugins=NodeRestriction",
		"--kubelet-client-certificate=",
		"--kubelet-client-key=",
		"--requestheader-client-ca-file=",
		"--allow-privileged=true",
		"apiserver.crt",
		"apiserver.key",
		"ca.crt",
	)
}

// ---------- controller-manager ----------

func TestGenerateControllerManagerConfig(t *testing.T) {
	cfg := setupTestConfig(t)

	if err := GenerateControllerManagerConfig(cfg); err != nil {
		t.Fatalf("GenerateControllerManagerConfig: %v", err)
	}

	envPath := filepath.Join(cfg.ComponentConfigDir(), "controller-manager.env")
	fileContains(t, envPath,
		"KUBE_CONTROLLER_MANAGER_ARGS=",
		"--kubeconfig=",
		"controller-manager.kubeconfig",
		"--cluster-cidr=10.42.0.0/16",
		"--service-cluster-ip-range=10.96.0.0/16",
		"--cluster-signing-cert-file=",
		"--cluster-signing-key-file=",
		"--root-ca-file=",
		"--service-account-private-key-file=",
		"--use-service-account-credentials=true",
		"--controllers=*,bootstrapsigner,tokencleaner",
		"ca.crt",
		"ca.key",
		"sa.key",
	)
}

// ---------- scheduler ----------

func TestGenerateSchedulerConfig(t *testing.T) {
	cfg := setupTestConfig(t)

	if err := GenerateSchedulerConfig(cfg); err != nil {
		t.Fatalf("GenerateSchedulerConfig: %v", err)
	}

	envPath := filepath.Join(cfg.ComponentConfigDir(), "scheduler.env")
	fileContains(t, envPath,
		"KUBE_SCHEDULER_ARGS=",
		"--kubeconfig=",
		"scheduler.kubeconfig",
	)
}

// ---------- kubelet ----------

func TestGenerateKubeletConfig(t *testing.T) {
	cfg := setupTestConfig(t)

	if err := GenerateKubeletConfig(cfg); err != nil {
		t.Fatalf("GenerateKubeletConfig: %v", err)
	}

	configDir := cfg.ComponentConfigDir()

	// kubelet-config.yaml
	kubeletCfg := filepath.Join(configDir, "kubelet-config.yaml")
	fileContains(t, kubeletCfg,
		"apiVersion: kubelet.config.k8s.io/v1beta1",
		"kind: KubeletConfiguration",
		"clientCAFile:",
		"ca.crt",
		"webhook:",
		"enabled: true",
		"mode: Webhook",
		"clusterDNS:",
		"10.96.0.10",
		"clusterDomain: cluster.local",
		"containerRuntimeEndpoint: unix:///var/run/crio/crio.sock",
		"cgroupDriver: systemd",
	)

	// kubelet.env
	envPath := filepath.Join(configDir, "kubelet.env")
	fileContains(t, envPath,
		"KUBELET_ARGS=",
		"--config=",
		"kubelet-config.yaml",
		"--kubeconfig=",
		"kubelet.kubeconfig",
		"--hostname-override=192.168.1.100",
		"--node-ip=192.168.1.100",
		"--container-runtime-endpoint=unix:///var/run/crio/crio.sock",
	)
}
