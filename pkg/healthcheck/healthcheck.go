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
}

func NewHealthChecker(kubeconfigPath, certDir string) *HealthChecker {
	return &HealthChecker{
		kubeconfigPath: kubeconfigPath,
		certDir:        certDir,
	}
}

func (h *HealthChecker) CheckAPIServer() error {
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := client.Get("https://127.0.0.1:6443/healthz")
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

	resp, err := client.Get("https://127.0.0.1:2379/health")
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
	cmd := exec.Command("kubectl",
		"--kubeconfig", h.kubeconfigPath,
		"get", "nodes",
		"-o", `jsonpath={.items[0].status.conditions[?(@.type=="Ready")].status}`,
	)
	output, err := cmd.Output()
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
