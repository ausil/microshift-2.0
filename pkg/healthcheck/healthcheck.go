package healthcheck

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

type HealthChecker struct {
	kubeconfigPath string
	certDir        string
	apiServerURL   string
	etcdURL        string
	cmdRunner      func(name string, args ...string) ([]byte, error)
}

func NewHealthChecker(kubeconfigPath, certDir string) *HealthChecker {
	return &HealthChecker{
		kubeconfigPath: kubeconfigPath,
		certDir:        certDir,
		apiServerURL:   "https://127.0.0.1:6443",
		etcdURL:        "https://127.0.0.1:2379",
		cmdRunner:      defaultCmdRunner,
	}
}

func defaultCmdRunner(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

func (h *HealthChecker) CheckAPIServer() error {
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := client.Get(h.apiServerURL + "/healthz")
	if err != nil {
		return fmt.Errorf("API server health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API server unhealthy: status %d", resp.StatusCode)
	}
	return nil
}

func (h *HealthChecker) CheckEtcd() error {
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := client.Get(h.etcdURL + "/health")
	if err != nil {
		return fmt.Errorf("etcd health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("etcd unhealthy: status %d", resp.StatusCode)
	}
	return nil
}

func (h *HealthChecker) CheckNodeReady() error {
	output, err := h.cmdRunner("kubectl",
		"--kubeconfig", h.kubeconfigPath,
		"get", "nodes",
		"-o", `jsonpath={.items[0].status.conditions[?(@.type=="Ready")].status}`,
	)
	if err != nil {
		return fmt.Errorf("checking node status: %w", err)
	}

	if strings.TrimSpace(string(output)) != "True" {
		return fmt.Errorf("node not ready: status=%s", string(output))
	}
	return nil
}

func (h *HealthChecker) CheckAll() error {
	if err := h.CheckAPIServer(); err != nil {
		return fmt.Errorf("apiserver: %w", err)
	}
	if err := h.CheckEtcd(); err != nil {
		return fmt.Errorf("etcd: %w", err)
	}
	if err := h.CheckNodeReady(); err != nil {
		return fmt.Errorf("node: %w", err)
	}
	return nil
}
