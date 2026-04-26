package daemon

import (
	"os"
	"testing"

	"github.com/ausil/microshift-2.0/pkg/config"
)

func TestNew(t *testing.T) {
	cfg := config.NewDefaultConfig()
	d := New(cfg)
	if d == nil {
		t.Fatal("New returned nil")
	}
	if d.cfg != cfg {
		t.Error("daemon config doesn't match")
	}
	if d.stopCh == nil {
		t.Error("stopCh is nil")
	}
}

func TestGenerateComponentConfigs(t *testing.T) {
	cfg := testConfig(t)

	d := New(cfg)
	if err := d.generateComponentConfigs(); err != nil {
		t.Fatalf("generateComponentConfigs: %v", err)
	}

	expectedFiles := []string{
		"etcd.yaml", "etcd.env",
		"apiserver.env",
		"controller-manager.env",
		"scheduler.env",
		"kubelet-config.yaml", "kubelet.env",
	}

	for _, f := range expectedFiles {
		path := cfg.ComponentConfigDir() + "/" + f
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected file %s: %v", f, err)
		}
	}
}

func TestComponentServices(t *testing.T) {
	expected := []string{
		"microshift-etcd.service",
		"microshift-apiserver.service",
		"microshift-controller-manager.service",
		"microshift-scheduler.service",
		"microshift-kubelet.service",
	}
	if len(componentServices) != len(expected) {
		t.Fatalf("expected %d services, got %d", len(expected), len(componentServices))
	}
	for i, svc := range expected {
		if componentServices[i] != svc {
			t.Errorf("service[%d]: expected %s, got %s", i, svc, componentServices[i])
		}
	}
}

func TestStop(t *testing.T) {
	cfg := config.NewDefaultConfig()
	d := New(cfg)

	// Stop should close the channel without panic
	d.Stop()

	// Verify channel is closed
	select {
	case _, ok := <-d.stopCh:
		if ok {
			t.Error("stopCh should be closed")
		}
	default:
		t.Error("stopCh should be readable after Stop")
	}
}

func TestStopComponentsNilManager(t *testing.T) {
	cfg := config.NewDefaultConfig()
	d := New(cfg)
	// Should not panic with nil svcMgr
	d.stopComponents()
}

func TestSdNotifyNoSocket(t *testing.T) {
	t.Setenv("NOTIFY_SOCKET", "")
	// Should return silently without error when no socket is set.
	sdNotify("READY=1")
}

func TestSdNotifyInvalidSocket(t *testing.T) {
	t.Setenv("NOTIFY_SOCKET", "/tmp/nonexistent-microshift-test.sock")
	// Should not panic even with an invalid socket path.
	sdNotify("READY=1")
}

func TestNewInitializesStopCh(t *testing.T) {
	cfg := config.NewDefaultConfig()
	d := New(cfg)

	// stopCh should be open (not closed).
	select {
	case <-d.stopCh:
		t.Error("stopCh should not be closed on a new daemon")
	default:
		// expected
	}
}

func TestGenerateComponentConfigsCreatesAllFiles(t *testing.T) {
	cfg := testConfig(t)
	d := New(cfg)

	if err := d.generateComponentConfigs(); err != nil {
		t.Fatalf("generateComponentConfigs: %v", err)
	}

	// Verify file contents are non-empty.
	files := map[string]string{
		"etcd.yaml":              "name:",
		"etcd.env":               "ETCD_",
		"apiserver.env":          "KUBE_APISERVER_ARGS",
		"controller-manager.env": "KUBE_CONTROLLER_MANAGER_ARGS",
		"scheduler.env":          "KUBE_SCHEDULER_ARGS",
		"kubelet-config.yaml":    "KubeletConfiguration",
		"kubelet.env":            "KUBELET_ARGS",
	}

	for f, expected := range files {
		path := cfg.ComponentConfigDir() + "/" + f
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("reading %s: %v", f, err)
			continue
		}
		content := string(data)
		if len(content) == 0 {
			t.Errorf("%s is empty", f)
		}
		if expected != "" && !contains(content, expected) {
			t.Errorf("%s: expected to contain %q", f, expected)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestComponentServicesOrder(t *testing.T) {
	// Verify etcd comes first (must start before apiserver).
	if componentServices[0] != "microshift-etcd.service" {
		t.Errorf("expected etcd first, got %s", componentServices[0])
	}
	// Verify apiserver comes second.
	if componentServices[1] != "microshift-apiserver.service" {
		t.Errorf("expected apiserver second, got %s", componentServices[1])
	}
	// Verify kubelet is last.
	last := componentServices[len(componentServices)-1]
	if last != "microshift-kubelet.service" {
		t.Errorf("expected kubelet last, got %s", last)
	}
}

func testConfig(t *testing.T) *config.Config {
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
