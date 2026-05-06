package kubeconfig

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/ausil/microshift-2.0/pkg/certs"
	"github.com/ausil/microshift-2.0/pkg/config"
	"gopkg.in/yaml.v3"
)

func TestGenerateKubeconfig(t *testing.T) {
	caCert := []byte("---CA CERT PEM---")
	clientCert := []byte("---CLIENT CERT PEM---")
	clientKey := []byte("---CLIENT KEY PEM---")

	data, err := GenerateKubeconfig("test-cluster", "https://127.0.0.1:6443", caCert, clientCert, clientKey)
	if err != nil {
		t.Fatalf("GenerateKubeconfig failed: %v", err)
	}

	// Verify it parses as valid YAML.
	var kc KubeconfigData
	if err := yaml.Unmarshal(data, &kc); err != nil {
		t.Fatalf("failed to unmarshal kubeconfig YAML: %v", err)
	}

	if kc.APIVersion != "v1" {
		t.Errorf("expected apiVersion=v1, got %q", kc.APIVersion)
	}
	if kc.Kind != "Config" {
		t.Errorf("expected kind=Config, got %q", kc.Kind)
	}
	if kc.CurrentContext != "test-cluster" {
		t.Errorf("expected current-context=test-cluster, got %q", kc.CurrentContext)
	}

	// Verify clusters.
	if len(kc.Clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(kc.Clusters))
	}
	cluster := kc.Clusters[0]
	if cluster.Name != "test-cluster" {
		t.Errorf("expected cluster name=test-cluster, got %q", cluster.Name)
	}
	if cluster.Cluster.Server != "https://127.0.0.1:6443" {
		t.Errorf("expected server=https://127.0.0.1:6443, got %q", cluster.Cluster.Server)
	}

	// Verify the CA data decodes back to the original.
	decoded, err := base64.StdEncoding.DecodeString(cluster.Cluster.CertificateAuthorityData)
	if err != nil {
		t.Fatalf("failed to decode CA data: %v", err)
	}
	if string(decoded) != "---CA CERT PEM---" {
		t.Errorf("CA data mismatch: %q", string(decoded))
	}

	// Verify users.
	if len(kc.Users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(kc.Users))
	}
	user := kc.Users[0]
	if user.Name != "test-cluster-user" {
		t.Errorf("expected user name=test-cluster-user, got %q", user.Name)
	}

	decodedCert, err := base64.StdEncoding.DecodeString(user.User.ClientCertificateData)
	if err != nil {
		t.Fatalf("failed to decode client cert data: %v", err)
	}
	if string(decodedCert) != "---CLIENT CERT PEM---" {
		t.Errorf("client cert data mismatch: %q", string(decodedCert))
	}

	decodedKey, err := base64.StdEncoding.DecodeString(user.User.ClientKeyData)
	if err != nil {
		t.Fatalf("failed to decode client key data: %v", err)
	}
	if string(decodedKey) != "---CLIENT KEY PEM---" {
		t.Errorf("client key data mismatch: %q", string(decodedKey))
	}

	// Verify contexts.
	if len(kc.Contexts) != 1 {
		t.Fatalf("expected 1 context, got %d", len(kc.Contexts))
	}
	ctx := kc.Contexts[0]
	if ctx.Name != "test-cluster" {
		t.Errorf("expected context name=test-cluster, got %q", ctx.Name)
	}
	if ctx.Context.Cluster != "test-cluster" {
		t.Errorf("expected context cluster=test-cluster, got %q", ctx.Context.Cluster)
	}
	if ctx.Context.User != "test-cluster-user" {
		t.Errorf("expected context user=test-cluster-user, got %q", ctx.Context.User)
	}
}

func TestGenerateKubeconfigCustomServer(t *testing.T) {
	data, err := GenerateKubeconfig("edge", "https://10.0.0.1:8443", []byte("ca"), []byte("cert"), []byte("key"))
	if err != nil {
		t.Fatalf("GenerateKubeconfig failed: %v", err)
	}

	var kc KubeconfigData
	if err := yaml.Unmarshal(data, &kc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if kc.Clusters[0].Cluster.Server != "https://10.0.0.1:8443" {
		t.Errorf("expected server=https://10.0.0.1:8443, got %q", kc.Clusters[0].Cluster.Server)
	}
}

func TestGenerateKubeconfigContextUserNaming(t *testing.T) {
	tests := []struct {
		cluster      string
		expectedUser string
	}{
		{"foo", "foo-user"},
		{"my-cluster", "my-cluster-user"},
		{"", "-user"},
	}
	for _, tt := range tests {
		data, err := GenerateKubeconfig(tt.cluster, "https://127.0.0.1:6443", []byte("ca"), []byte("cert"), []byte("key"))
		if err != nil {
			t.Fatalf("GenerateKubeconfig(%q): %v", tt.cluster, err)
		}
		var kc KubeconfigData
		if err := yaml.Unmarshal(data, &kc); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if kc.Users[0].Name != tt.expectedUser {
			t.Errorf("cluster=%q: expected user=%q, got %q", tt.cluster, tt.expectedUser, kc.Users[0].Name)
		}
	}
}

func TestGenerateKubeconfigYAMLValidity(t *testing.T) {
	data, err := GenerateKubeconfig("cluster", "https://1.2.3.4:6443", []byte("ca"), []byte("cert"), []byte("key"))
	if err != nil {
		t.Fatalf("GenerateKubeconfig failed: %v", err)
	}

	// Verify it's valid YAML by unmarshaling into a generic map.
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("kubeconfig is not valid YAML: %v", err)
	}

	// Check all expected top-level keys are present.
	for _, key := range []string{"apiVersion", "kind", "clusters", "users", "contexts", "current-context"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("missing top-level key %q", key)
		}
	}
}

func TestGenerateAllKubeconfigs(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.NodeIP = "192.168.1.100"
	cfg.DataDir = t.TempDir()

	cc, err := certs.GenerateAllCerts(cfg)
	if err != nil {
		t.Fatalf("GenerateAllCerts failed: %v", err)
	}

	if err := GenerateAllKubeconfigs(cfg, cc); err != nil {
		t.Fatalf("GenerateAllKubeconfigs failed: %v", err)
	}

	expectedFiles := []string{
		"admin.kubeconfig",
		"controller-manager.kubeconfig",
		"scheduler.kubeconfig",
		"kubelet.kubeconfig",
	}

	dir := cfg.KubeconfigDir()
	for _, name := range expectedFiles {
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("failed to read %s: %v", name, err)
			continue
		}

		// Verify each kubeconfig is valid YAML.
		var kc KubeconfigData
		if err := yaml.Unmarshal(data, &kc); err != nil {
			t.Errorf("%s is not valid kubeconfig YAML: %v", name, err)
			continue
		}
		if kc.APIVersion != "v1" {
			t.Errorf("%s: expected apiVersion=v1, got %q", name, kc.APIVersion)
		}
		if kc.CurrentContext != cfg.ClusterName {
			t.Errorf("%s: expected current-context=%s, got %q", name, cfg.ClusterName, kc.CurrentContext)
		}

		// Verify file permissions.
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("stat %s: %v", name, err)
			continue
		}
		if perm := info.Mode().Perm(); perm != 0600 {
			t.Errorf("%s: expected mode 0600, got %o", name, perm)
		}
	}
}

func TestGenerateAllKubeconfigsFileCount(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.NodeIP = "192.168.1.100"
	cfg.DataDir = t.TempDir()

	cc, err := certs.GenerateAllCerts(cfg)
	if err != nil {
		t.Fatalf("GenerateAllCerts: %v", err)
	}

	if err := GenerateAllKubeconfigs(cfg, cc); err != nil {
		t.Fatalf("GenerateAllKubeconfigs: %v", err)
	}

	entries, err := os.ReadDir(cfg.KubeconfigDir())
	if err != nil {
		t.Fatalf("reading kubeconfig dir: %v", err)
	}

	if len(entries) != 4 {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Errorf("expected 4 kubeconfig files, got %d: %v", len(entries), names)
	}
}

func TestGenerateAllKubeconfigsEachFileContent(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.NodeIP = "192.168.1.100"
	cfg.DataDir = t.TempDir()

	cc, err := certs.GenerateAllCerts(cfg)
	if err != nil {
		t.Fatalf("GenerateAllCerts: %v", err)
	}

	if err := GenerateAllKubeconfigs(cfg, cc); err != nil {
		t.Fatalf("GenerateAllKubeconfigs: %v", err)
	}

	files := []string{"admin", "controller-manager", "scheduler", "kubelet"}
	for _, name := range files {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(cfg.KubeconfigDir(), name+".kubeconfig")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading %s: %v", name, err)
			}

			var kc KubeconfigData
			if err := yaml.Unmarshal(data, &kc); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			if kc.CurrentContext != cfg.ClusterName {
				t.Errorf("current-context: expected %q, got %q", cfg.ClusterName, kc.CurrentContext)
			}
			if kc.Clusters[0].Cluster.Server != "https://127.0.0.1:6443" {
				t.Errorf("server: expected https://127.0.0.1:6443, got %q", kc.Clusters[0].Cluster.Server)
			}
			if kc.Contexts[0].Context.User != cfg.ClusterName+"-user" {
				t.Errorf("user: expected %s-user, got %q", cfg.ClusterName, kc.Contexts[0].Context.User)
			}
		})
	}
}
