package healthcheck

import (
	"testing"
)

func TestNewHealthChecker(t *testing.T) {
	hc := NewHealthChecker("/var/lib/microshift/kubeconfig/admin.kubeconfig", "/var/lib/microshift/certs")
	if hc.kubeconfigPath != "/var/lib/microshift/kubeconfig/admin.kubeconfig" {
		t.Errorf("unexpected kubeconfig path: %s", hc.kubeconfigPath)
	}
	if hc.certDir != "/var/lib/microshift/certs" {
		t.Errorf("unexpected cert dir: %s", hc.certDir)
	}
}

func TestCheckAPIServerNoServer(t *testing.T) {
	hc := NewHealthChecker("", "")
	err := hc.CheckAPIServer()
	if err == nil {
		t.Error("expected error when no API server is running")
	}
}
