package healthcheck

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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
	if hc.apiServerURL != "https://127.0.0.1:6443" {
		t.Errorf("unexpected default apiServerURL: %s", hc.apiServerURL)
	}
	if hc.etcdURL != "https://127.0.0.1:2379" {
		t.Errorf("unexpected default etcdURL: %s", hc.etcdURL)
	}
	if hc.cmdRunner == nil {
		t.Error("cmdRunner should not be nil")
	}
}

func TestCheckAPIServerNoServer(t *testing.T) {
	hc := NewHealthChecker("", "")
	err := hc.CheckAPIServer()
	if err == nil {
		t.Error("expected error when no API server is running")
	}
}

func TestCheckAPIServerHealthy(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	hc := &HealthChecker{
		apiServerURL: srv.URL,
		cmdRunner:    defaultCmdRunner,
	}

	if err := hc.CheckAPIServer(); err != nil {
		t.Errorf("expected healthy, got error: %v", err)
	}
}

func TestCheckAPIServerUnhealthy(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	hc := &HealthChecker{
		apiServerURL: srv.URL,
		cmdRunner:    defaultCmdRunner,
	}

	err := hc.CheckAPIServer()
	if err == nil {
		t.Fatal("expected error for unhealthy API server")
	}
	if !strings.Contains(err.Error(), "unhealthy: status 503") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckEtcdNoServer(t *testing.T) {
	hc := NewHealthChecker("", "")
	err := hc.CheckEtcd()
	if err == nil {
		t.Error("expected error when no etcd is running")
	}
	if !strings.Contains(err.Error(), "etcd health check failed") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCheckEtcdHealthy(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"health":"true"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	hc := &HealthChecker{
		etcdURL:   srv.URL,
		cmdRunner: defaultCmdRunner,
	}

	if err := hc.CheckEtcd(); err != nil {
		t.Errorf("expected healthy, got error: %v", err)
	}
}

func TestCheckEtcdUnhealthy(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	hc := &HealthChecker{
		etcdURL:   srv.URL,
		cmdRunner: defaultCmdRunner,
	}

	err := hc.CheckEtcd()
	if err == nil {
		t.Fatal("expected error for unhealthy etcd")
	}
	if !strings.Contains(err.Error(), "unhealthy: status 503") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckNodeReadyTrue(t *testing.T) {
	hc := &HealthChecker{
		kubeconfigPath: "/tmp/fake",
		cmdRunner: func(name string, args ...string) ([]byte, error) {
			return []byte("True"), nil
		},
	}

	if err := hc.CheckNodeReady(); err != nil {
		t.Errorf("expected node ready, got error: %v", err)
	}
}

func TestCheckNodeReadyFalse(t *testing.T) {
	hc := &HealthChecker{
		kubeconfigPath: "/tmp/fake",
		cmdRunner: func(name string, args ...string) ([]byte, error) {
			return []byte("False"), nil
		},
	}

	err := hc.CheckNodeReady()
	if err == nil {
		t.Fatal("expected error for node not ready")
	}
	if !strings.Contains(err.Error(), "not ready") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckNodeReadyCmdFailure(t *testing.T) {
	hc := &HealthChecker{
		kubeconfigPath: "/tmp/fake",
		cmdRunner: func(name string, args ...string) ([]byte, error) {
			return nil, fmt.Errorf("kubectl not found")
		},
	}

	err := hc.CheckNodeReady()
	if err == nil {
		t.Fatal("expected error for command failure")
	}
	if !strings.Contains(err.Error(), "checking node status") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckNodeReadyNoKubectl(t *testing.T) {
	hc := NewHealthChecker("/nonexistent/kubeconfig", "")
	err := hc.CheckNodeReady()
	if err == nil {
		t.Error("expected error when kubectl fails")
	}
}

func TestCheckAllFailsOnFirst(t *testing.T) {
	hc := NewHealthChecker("", "")
	err := hc.CheckAll()
	if err == nil {
		t.Error("expected error from CheckAll when nothing is running")
	}
	if !strings.Contains(err.Error(), "apiserver") {
		t.Errorf("expected apiserver error first, got: %v", err)
	}
}

func TestCheckAllAllPass(t *testing.T) {
	apiSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer apiSrv.Close()

	etcdSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer etcdSrv.Close()

	hc := &HealthChecker{
		kubeconfigPath: "/tmp/fake",
		apiServerURL:   apiSrv.URL,
		etcdURL:        etcdSrv.URL,
		cmdRunner: func(name string, args ...string) ([]byte, error) {
			return []byte("True"), nil
		},
	}

	if err := hc.CheckAll(); err != nil {
		t.Errorf("expected all checks to pass, got error: %v", err)
	}
}

func TestCheckAllFailsOnEtcd(t *testing.T) {
	apiSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer apiSrv.Close()

	etcdSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer etcdSrv.Close()

	hc := &HealthChecker{
		kubeconfigPath: "/tmp/fake",
		apiServerURL:   apiSrv.URL,
		etcdURL:        etcdSrv.URL,
		cmdRunner: func(name string, args ...string) ([]byte, error) {
			return []byte("True"), nil
		},
	}

	err := hc.CheckAll()
	if err == nil {
		t.Fatal("expected error from etcd")
	}
	if !strings.Contains(err.Error(), "etcd") {
		t.Errorf("expected etcd error, got: %v", err)
	}
}

func TestCheckAllFailsOnNode(t *testing.T) {
	apiSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer apiSrv.Close()

	etcdSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer etcdSrv.Close()

	hc := &HealthChecker{
		kubeconfigPath: "/tmp/fake",
		apiServerURL:   apiSrv.URL,
		etcdURL:        etcdSrv.URL,
		cmdRunner: func(name string, args ...string) ([]byte, error) {
			return []byte("False"), nil
		},
	}

	err := hc.CheckAll()
	if err == nil {
		t.Fatal("expected error from node check")
	}
	if !strings.Contains(err.Error(), "node") {
		t.Errorf("expected node error, got: %v", err)
	}
}
