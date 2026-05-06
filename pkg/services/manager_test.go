package services

import (
	"path/filepath"
	"testing"

	"github.com/ausil/microshift-2.0/internal/testutil"
)

var setupTestConfig = testutil.NewTestConfig
var fileContains = testutil.FileContains

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

func TestGenerateEtcdConfigPaths(t *testing.T) {
	cfg := setupTestConfig(t)

	if err := GenerateEtcdConfig(cfg); err != nil {
		t.Fatalf("GenerateEtcdConfig: %v", err)
	}

	// Verify the etcd data dir path is included.
	etcdYAML := filepath.Join(cfg.ComponentConfigDir(), "etcd.yaml")
	fileContains(t, etcdYAML, cfg.EtcdDataDir())

	// Verify cert paths reference the config's cert dir.
	etcdEnv := filepath.Join(cfg.ComponentConfigDir(), "etcd.env")
	fileContains(t, etcdEnv, cfg.CertDir())
}

func TestGenerateAPIServerConfigCustomCIDR(t *testing.T) {
	cfg := setupTestConfig(t)
	cfg.ServiceCIDR = "10.200.0.0/16"

	if err := GenerateAPIServerConfig(cfg); err != nil {
		t.Fatalf("GenerateAPIServerConfig: %v", err)
	}

	envPath := filepath.Join(cfg.ComponentConfigDir(), "apiserver.env")
	fileContains(t, envPath, "--service-cluster-ip-range=10.200.0.0/16")
}

func TestGenerateControllerManagerConfigCustomCIDR(t *testing.T) {
	cfg := setupTestConfig(t)
	cfg.ClusterCIDR = "10.200.0.0/16"
	cfg.ServiceCIDR = "10.100.0.0/16"

	if err := GenerateControllerManagerConfig(cfg); err != nil {
		t.Fatalf("GenerateControllerManagerConfig: %v", err)
	}

	envPath := filepath.Join(cfg.ComponentConfigDir(), "controller-manager.env")
	fileContains(t, envPath,
		"--cluster-cidr=10.200.0.0/16",
		"--service-cluster-ip-range=10.100.0.0/16",
	)
}

func TestGenerateKubeletConfigCustomNodeIP(t *testing.T) {
	cfg := setupTestConfig(t)
	cfg.NodeIP = "10.0.0.5"

	if err := GenerateKubeletConfig(cfg); err != nil {
		t.Fatalf("GenerateKubeletConfig: %v", err)
	}

	envPath := filepath.Join(cfg.ComponentConfigDir(), "kubelet.env")
	fileContains(t, envPath,
		"--hostname-override=10.0.0.5",
		"--node-ip=10.0.0.5",
	)

	// DNS address should be based on ServiceCIDR.
	configPath := filepath.Join(cfg.ComponentConfigDir(), "kubelet-config.yaml")
	fileContains(t, configPath, cfg.DNSServiceIP())
}

func TestGenerateSchedulerConfigKubeconfig(t *testing.T) {
	cfg := setupTestConfig(t)

	if err := GenerateSchedulerConfig(cfg); err != nil {
		t.Fatalf("GenerateSchedulerConfig: %v", err)
	}

	envPath := filepath.Join(cfg.ComponentConfigDir(), "scheduler.env")
	fileContains(t, envPath, cfg.KubeconfigDir())
}

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
