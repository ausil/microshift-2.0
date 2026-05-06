package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ausil/microshift-2.0/pkg/config"
)

func NewTestConfig(t *testing.T) *config.Config {
	t.Helper()
	dir := t.TempDir()
	cfg := config.NewDefaultConfig()
	cfg.DataDir = dir
	cfg.NodeIP = "192.168.1.100"

	for _, sub := range []string{
		cfg.CertDir(),
		cfg.KubeconfigDir(),
		cfg.ComponentConfigDir(),
		cfg.EtcdDataDir(),
	} {
		if err := os.MkdirAll(sub, 0755); err != nil {
			t.Fatalf("creating %s: %v", sub, err)
		}
	}
	return cfg
}

func FileContains(t *testing.T, path string, wants ...string) {
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

func FileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file %s to exist: %v", path, err)
	}
}
