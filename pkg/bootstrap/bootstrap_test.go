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

func init() {
	// Ensure MICROSHIFT_ASSETS_DIR doesn't leak between tests
	os.Unsetenv("MICROSHIFT_ASSETS_DIR")
}
