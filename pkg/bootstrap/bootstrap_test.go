package bootstrap

import (
	"os"
	"strings"
	"testing"

	"github.com/ausil/microshift-2.0/pkg/config"
)

func TestTemplateManifest(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.NodeIP = "192.168.1.100"

	input := []byte(`clusterIP: "{{.DNSAddress}}"
podCIDR: "{{.ClusterCIDR}}"
serviceCIDR: "{{.ServiceCIDR}}"
nodeIP: "{{.NodeIP}}"`)

	rendered, err := templateManifest(input, cfg)
	if err != nil {
		t.Fatal(err)
	}

	result := string(rendered)
	if !strings.Contains(result, "10.96.0.10") {
		t.Errorf("expected DNS address in output, got: %s", result)
	}
	if !strings.Contains(result, "10.42.0.0/16") {
		t.Errorf("expected cluster CIDR in output, got: %s", result)
	}
	if !strings.Contains(result, "10.96.0.0/16") {
		t.Errorf("expected service CIDR in output, got: %s", result)
	}
	if !strings.Contains(result, "192.168.1.100") {
		t.Errorf("expected node IP in output, got: %s", result)
	}
}

func TestAssetsDirEnvVar(t *testing.T) {
	t.Setenv("MICROSHIFT_ASSETS_DIR", "/custom/assets")
	if got := AssetsDir(); got != "/custom/assets" {
		t.Errorf("expected /custom/assets, got %s", got)
	}
}

func TestAssetsDirFallback(t *testing.T) {
	t.Setenv("MICROSHIFT_ASSETS_DIR", "")
	dir := AssetsDir()
	// Should fall back to ./assets or /usr/share/microshift/assets
	if dir != "./assets" && dir != "/usr/share/microshift/assets" {
		t.Errorf("unexpected assets dir: %s", dir)
	}
}

func TestTemplateManifestInvalidTemplate(t *testing.T) {
	cfg := config.NewDefaultConfig()
	input := []byte(`{{.InvalidField}}`)
	_, err := templateManifest(input, cfg)
	if err == nil {
		t.Error("expected error for invalid template field")
	}
}

func TestTemplateManifestStorageVars(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.Storage.Driver = "nfs"
	cfg.Storage.NFS.Server = "nfs.example.com"
	cfg.Storage.NFS.Path = "/exports/data"
	cfg.Storage.LVMS.VolumeGroup = "test-vg"

	input := []byte(`server: "{{.NFSServer}}"
path: "{{.NFSPath}}"
storagePath: "{{.StoragePath}}"
vg: "{{.VolumeGroup}}"`)

	rendered, err := templateManifest(input, cfg)
	if err != nil {
		t.Fatal(err)
	}

	result := string(rendered)
	if !strings.Contains(result, "nfs.example.com") {
		t.Errorf("expected NFS server in output, got: %s", result)
	}
	if !strings.Contains(result, "/exports/data") {
		t.Errorf("expected NFS path in output, got: %s", result)
	}
	if !strings.Contains(result, "/var/lib/microshift/storage") {
		t.Errorf("expected storage path in output, got: %s", result)
	}
	if !strings.Contains(result, "test-vg") {
		t.Errorf("expected volume group in output, got: %s", result)
	}
}

func TestStorageManifestDir(t *testing.T) {
	tests := []struct {
		driver   string
		expected string
	}{
		{"local-path", "storage/local-path"},
		{"lvms", "storage/lvms"},
		{"nfs", "storage/nfs"},
		{"none", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := storageManifestDir(tt.driver)
		if got != tt.expected {
			t.Errorf("storageManifestDir(%q) = %q, want %q", tt.driver, got, tt.expected)
		}
	}
}

func TestTemplateManifestNoTemplateVars(t *testing.T) {
	cfg := config.NewDefaultConfig()
	input := []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: test`)

	rendered, err := templateManifest(input, cfg)
	if err != nil {
		t.Fatal(err)
	}

	if string(rendered) != string(input) {
		t.Errorf("manifest without template vars should be unchanged")
	}
}

func TestApplyManifestsWithAssets(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.NodeIP = "192.168.1.100"

	assetsDir := t.TempDir()

	// Create a minimal RBAC manifest with no template vars.
	rbacDir := assetsDir + "/rbac"
	if err := os.MkdirAll(rbacDir, 0755); err != nil {
		t.Fatal(err)
	}
	rbacManifest := `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: test-role
`
	if err := os.WriteFile(rbacDir+"/role.yaml", []byte(rbacManifest), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a storage/local-path manifest with template vars.
	storageDir := assetsDir + "/storage/local-path"
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		t.Fatal(err)
	}
	storageManifest := `apiVersion: v1
kind: ConfigMap
metadata:
  name: local-path-config
data:
  path: "{{.StoragePath}}"
  dns: "{{.DNSAddress}}"
`
	if err := os.WriteFile(storageDir+"/local-path.yaml", []byte(storageManifest), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a non-YAML file that should be skipped.
	if err := os.WriteFile(rbacDir+"/README.md", []byte("skip me"), 0644); err != nil {
		t.Fatal(err)
	}

	// applyManifests will call kubectlApply which will fail because there's
	// no API server, but we can verify the template rendering worked by
	// checking the error message doesn't mention template errors.
	err := applyManifests("/tmp/nonexistent.kubeconfig", cfg, assetsDir)
	if err == nil {
		t.Fatal("expected error (no kubectl/API server), got nil")
	}
	// The error should be about kubectl failing, not template parsing.
	if strings.Contains(err.Error(), "template") {
		t.Errorf("unexpected template error: %v", err)
	}
}

func TestApplyManifestsNonexistentDir(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.NodeIP = "192.168.1.100"

	// Create an assets dir with only the required dirs missing.
	assetsDir := t.TempDir()
	// No rbac, kindnet, or coredns dirs — should skip them gracefully.
	err := applyManifests("/tmp/nonexistent.kubeconfig", cfg, assetsDir)
	if err != nil {
		t.Errorf("expected nil for nonexistent manifest dirs, got: %v", err)
	}
}

func TestApplyManifestsStorageNone(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.Storage.Driver = "none"
	cfg.NodeIP = "192.168.1.100"

	assetsDir := t.TempDir()
	// With driver "none", no storage dir should be appended.
	err := applyManifests("/tmp/nonexistent.kubeconfig", cfg, assetsDir)
	if err != nil {
		t.Errorf("expected nil for driver=none with no manifest dirs, got: %v", err)
	}
}

func TestTemplateManifestAllVars(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.NodeIP = "10.0.0.5"
	cfg.ClusterCIDR = "10.200.0.0/16"
	cfg.ServiceCIDR = "10.100.0.0/16"
	cfg.Storage.LocalPath.StoragePath = "/custom/storage"
	cfg.Storage.LVMS.VolumeGroup = "my-vg"
	cfg.Storage.NFS.Server = "nfs.corp.com"
	cfg.Storage.NFS.Path = "/data"

	input := []byte(`{{.ClusterCIDR}} {{.ServiceCIDR}} {{.DNSAddress}} {{.NodeIP}} {{.StoragePath}} {{.VolumeGroup}} {{.NFSServer}} {{.NFSPath}}`)

	rendered, err := templateManifest(input, cfg)
	if err != nil {
		t.Fatal(err)
	}

	result := string(rendered)
	for _, want := range []string{
		"10.200.0.0/16", "10.100.0.0/16", "10.100.0.10", "10.0.0.5",
		"/custom/storage", "my-vg", "nfs.corp.com", "/data",
	} {
		if !strings.Contains(result, want) {
			t.Errorf("expected %q in output, got: %s", want, result)
		}
	}
}

func init() {
	// Ensure MICROSHIFT_ASSETS_DIR doesn't leak between tests
	os.Unsetenv("MICROSHIFT_ASSETS_DIR")
}
