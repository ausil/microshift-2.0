package bootstrap

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/ausil/microshift-2.0/pkg/config"
)

func Bootstrap(cfg *config.Config) error {
	kubeconfigPath := filepath.Join(cfg.KubeconfigDir(), "admin.kubeconfig")

	if err := waitForAPIServer(kubeconfigPath, 120*time.Second); err != nil {
		return fmt.Errorf("waiting for API server: %w", err)
	}

	if cfg.Storage.Driver == "lvms" {
		if err := ensureLVMVolumeGroup(cfg); err != nil {
			return fmt.Errorf("setting up LVM: %w", err)
		}
	}

	if cfg.Storage.Driver == "local-path" {
		if err := os.MkdirAll(cfg.Storage.LocalPath.StoragePath, 0755); err != nil {
			return fmt.Errorf("creating local-path storage dir: %w", err)
		}
	}

	assetsDir := AssetsDir()
	if err := applyManifests(kubeconfigPath, cfg, assetsDir); err != nil {
		return fmt.Errorf("applying bootstrap manifests: %w", err)
	}

	return nil
}

func waitForAPIServer(kubeconfigPath string, timeout time.Duration) error {
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get("https://127.0.0.1:6443/healthz")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("API server did not become healthy within %s", timeout)
}

func storageManifestDir(driver string) string {
	switch driver {
	case "local-path":
		return "storage/local-path"
	case "lvms":
		return "storage/lvms"
	case "nfs":
		return "storage/nfs"
	default:
		return ""
	}
}

func applyManifests(kubeconfigPath string, cfg *config.Config, assetsDir string) error {
	dirs := []string{"rbac", "kindnet", "coredns"}

	if storageDir := storageManifestDir(cfg.Storage.Driver); storageDir != "" {
		dirs = append(dirs, storageDir)
	}

	for _, dir := range dirs {
		manifestDir := filepath.Join(assetsDir, dir)
		entries, err := os.ReadDir(manifestDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("reading manifest dir %s: %w", dir, err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
				continue
			}

			data, err := os.ReadFile(filepath.Join(manifestDir, entry.Name()))
			if err != nil {
				return fmt.Errorf("reading manifest %s/%s: %w", dir, entry.Name(), err)
			}

			rendered, err := templateManifest(data, cfg)
			if err != nil {
				return fmt.Errorf("templating manifest %s/%s: %w", dir, entry.Name(), err)
			}

			if err := kubectlApply(kubeconfigPath, rendered); err != nil {
				return fmt.Errorf("applying manifest %s/%s: %w", dir, entry.Name(), err)
			}
		}
	}

	return nil
}

func templateManifest(data []byte, cfg *config.Config) ([]byte, error) {
	tmpl, err := template.New("manifest").Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	vars := struct {
		ClusterCIDR string
		ServiceCIDR string
		DNSAddress  string
		NodeIP      string
		StoragePath string
		VolumeGroup string
		NFSServer   string
		NFSPath     string
	}{
		ClusterCIDR: cfg.ClusterCIDR,
		ServiceCIDR: cfg.ServiceCIDR,
		DNSAddress:  cfg.DNSServiceIP(),
		NodeIP:      cfg.NodeIP,
		StoragePath: cfg.Storage.LocalPath.StoragePath,
		VolumeGroup: cfg.Storage.LVMS.VolumeGroup,
		NFSServer:   cfg.Storage.NFS.Server,
		NFSPath:     cfg.Storage.NFS.Path,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	return buf.Bytes(), nil
}

func kubectlApply(kubeconfigPath string, manifest []byte) error {
	cmd := exec.Command("kubectl", "apply", "--kubeconfig", kubeconfigPath, "-f", "-")
	cmd.Stdin = bytes.NewReader(manifest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func ensureLVMVolumeGroup(cfg *config.Config) error {
	vgName := cfg.Storage.LVMS.VolumeGroup

	// Check if VG already exists
	cmd := exec.Command("vgs", "--noheadings", "-o", "vg_name", vgName)
	if err := cmd.Run(); err == nil {
		return nil // VG already exists
	}

	// VG doesn't exist — if a device is configured, create it
	device := cfg.Storage.LVMS.Device
	if device == "" {
		return fmt.Errorf("volume group %q does not exist and no device configured to create it", vgName)
	}

	// Create PV on the device
	pvcreate := exec.Command("pvcreate", device)
	pvcreate.Stdout = os.Stdout
	pvcreate.Stderr = os.Stderr
	if err := pvcreate.Run(); err != nil {
		return fmt.Errorf("pvcreate %s: %w", device, err)
	}

	// Create the VG
	vgcreate := exec.Command("vgcreate", vgName, device)
	vgcreate.Stdout = os.Stdout
	vgcreate.Stderr = os.Stderr
	if err := vgcreate.Run(); err != nil {
		return fmt.Errorf("vgcreate %s %s: %w", vgName, device, err)
	}

	return nil
}

func AssetsDir() string {
	if dir := os.Getenv("MICROSHIFT_ASSETS_DIR"); dir != "" {
		return dir
	}
	if info, err := os.Stat("/usr/share/microshift/assets"); err == nil && info.IsDir() {
		return "/usr/share/microshift/assets"
	}
	return "./assets"
}
