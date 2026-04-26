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

func applyManifests(kubeconfigPath string, cfg *config.Config, assetsDir string) error {
	dirs := []string{"rbac", "kindnet", "coredns"}

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
	}{
		ClusterCIDR: cfg.ClusterCIDR,
		ServiceCIDR: cfg.ServiceCIDR,
		DNSAddress:  cfg.DNSServiceIP(),
		NodeIP:      cfg.NodeIP,
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

func AssetsDir() string {
	if dir := os.Getenv("MICROSHIFT_ASSETS_DIR"); dir != "" {
		return dir
	}
	if info, err := os.Stat("/usr/share/microshift/assets"); err == nil && info.IsDir() {
		return "/usr/share/microshift/assets"
	}
	return "./assets"
}
