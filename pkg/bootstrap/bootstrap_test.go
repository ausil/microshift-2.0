package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

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

	if err := os.WriteFile(rbacDir+"/README.md", []byte("skip me"), 0644); err != nil {
		t.Fatal(err)
	}

	err := applyManifests("/tmp/nonexistent.kubeconfig", cfg, assetsDir)
	if err == nil {
		t.Fatal("expected error (no kubectl/API server), got nil")
	}
	if strings.Contains(err.Error(), "template") {
		t.Errorf("unexpected template error: %v", err)
	}
}

func TestApplyManifestsNonexistentDir(t *testing.T) {
	cfg := config.NewDefaultConfig()
	cfg.NodeIP = "192.168.1.100"

	assetsDir := t.TempDir()
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

// --- New tests ---

func TestWaitForAPIServerSuccess(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	if err := waitForAPIServer(srv.URL, 5*time.Second); err != nil {
		t.Errorf("expected success, got error: %v", err)
	}
}

func TestWaitForAPIServerTimeout(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	err := waitForAPIServer(srv.URL, 3*time.Second)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "did not become healthy") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestKubectlApplyArgs(t *testing.T) {
	var capturedArgs []string
	origExecCommand := execCommand
	execCommand = func(name string, args ...string) *exec.Cmd {
		capturedArgs = append([]string{name}, args...)
		return exec.Command("true")
	}
	t.Cleanup(func() { execCommand = origExecCommand })

	manifest := []byte("apiVersion: v1\nkind: ConfigMap\n")
	if err := kubectlApply("/tmp/test.kubeconfig", manifest); err != nil {
		t.Fatalf("kubectlApply: %v", err)
	}

	if len(capturedArgs) < 5 {
		t.Fatalf("expected at least 5 args, got %v", capturedArgs)
	}
	if capturedArgs[0] != "kubectl" {
		t.Errorf("expected kubectl, got %s", capturedArgs[0])
	}
	if capturedArgs[1] != "apply" {
		t.Errorf("expected apply, got %s", capturedArgs[1])
	}
	if capturedArgs[3] != "/tmp/test.kubeconfig" {
		t.Errorf("expected kubeconfig path, got %s", capturedArgs[3])
	}
}

func TestEnsureLVMVolumeGroupExists(t *testing.T) {
	origExecCommand := execCommand
	execCommand = func(name string, args ...string) *exec.Cmd {
		if name == "vgs" {
			return exec.Command("true")
		}
		return exec.Command("false")
	}
	t.Cleanup(func() { execCommand = origExecCommand })

	cfg := config.NewDefaultConfig()
	cfg.Storage.Driver = "lvms"
	cfg.Storage.LVMS.VolumeGroup = "test-vg"

	if err := ensureLVMVolumeGroup(cfg); err != nil {
		t.Errorf("expected nil when VG exists, got: %v", err)
	}
}

func TestEnsureLVMVolumeGroupCreateNew(t *testing.T) {
	var cmds []string
	origExecCommand := execCommand
	execCommand = func(name string, args ...string) *exec.Cmd {
		cmds = append(cmds, name)
		if name == "vgs" {
			return exec.Command("false")
		}
		return exec.Command("true")
	}
	t.Cleanup(func() { execCommand = origExecCommand })

	cfg := config.NewDefaultConfig()
	cfg.Storage.Driver = "lvms"
	cfg.Storage.LVMS.VolumeGroup = "new-vg"
	cfg.Storage.LVMS.Device = "/dev/sdb"

	if err := ensureLVMVolumeGroup(cfg); err != nil {
		t.Errorf("expected success, got: %v", err)
	}

	if len(cmds) != 3 {
		t.Fatalf("expected 3 commands (vgs, pvcreate, vgcreate), got %v", cmds)
	}
	if cmds[0] != "vgs" || cmds[1] != "pvcreate" || cmds[2] != "vgcreate" {
		t.Errorf("unexpected command sequence: %v", cmds)
	}
}

func TestEnsureLVMVolumeGroupNoDevice(t *testing.T) {
	origExecCommand := execCommand
	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	}
	t.Cleanup(func() { execCommand = origExecCommand })

	cfg := config.NewDefaultConfig()
	cfg.Storage.Driver = "lvms"
	cfg.Storage.LVMS.VolumeGroup = "missing-vg"
	cfg.Storage.LVMS.Device = ""

	err := ensureLVMVolumeGroup(cfg)
	if err == nil {
		t.Fatal("expected error when VG missing and no device configured")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("unexpected error: %v", err)
	}
}

func init() {
	os.Unsetenv("MICROSHIFT_ASSETS_DIR")
}
