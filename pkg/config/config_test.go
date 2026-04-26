package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()

	if cfg.ClusterName != "microshift" {
		t.Errorf("expected ClusterName=microshift, got %q", cfg.ClusterName)
	}
	if cfg.BaseDomain != "microshift.local" {
		t.Errorf("expected BaseDomain=microshift.local, got %q", cfg.BaseDomain)
	}
	if cfg.ServiceCIDR != "10.96.0.0/16" {
		t.Errorf("expected ServiceCIDR=10.96.0.0/16, got %q", cfg.ServiceCIDR)
	}
	if cfg.ClusterCIDR != "10.42.0.0/16" {
		t.Errorf("expected ClusterCIDR=10.42.0.0/16, got %q", cfg.ClusterCIDR)
	}
	if cfg.DataDir != "/var/lib/microshift" {
		t.Errorf("expected DataDir=/var/lib/microshift, got %q", cfg.DataDir)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected LogLevel=info, got %q", cfg.LogLevel)
	}
	if cfg.CNI != "kindnet" {
		t.Errorf("expected CNI=kindnet, got %q", cfg.CNI)
	}
	if cfg.EtcdMemoryLimit != 0 {
		t.Errorf("expected EtcdMemoryLimit=0, got %d", cfg.EtcdMemoryLimit)
	}
	if cfg.DNSAddress != "10.96.0.10" {
		t.Errorf("expected DNSAddress=10.96.0.10, got %q", cfg.DNSAddress)
	}
}

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	yamlContent := `
clusterName: "my-cluster"
baseDomain: "example.com"
nodeIP: "192.168.1.100"
serviceCIDR: "10.100.0.0/16"
clusterCIDR: "10.200.0.0/16"
logLevel: "debug"
cni: "ovn-kubernetes"
etcdMemoryLimit: 512
`
	if err := os.WriteFile(cfgPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("writing test config: %v", err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.ClusterName != "my-cluster" {
		t.Errorf("expected ClusterName=my-cluster, got %q", cfg.ClusterName)
	}
	if cfg.BaseDomain != "example.com" {
		t.Errorf("expected BaseDomain=example.com, got %q", cfg.BaseDomain)
	}
	if cfg.NodeIP != "192.168.1.100" {
		t.Errorf("expected NodeIP=192.168.1.100, got %q", cfg.NodeIP)
	}
	if cfg.ServiceCIDR != "10.100.0.0/16" {
		t.Errorf("expected ServiceCIDR=10.100.0.0/16, got %q", cfg.ServiceCIDR)
	}
	if cfg.ClusterCIDR != "10.200.0.0/16" {
		t.Errorf("expected ClusterCIDR=10.200.0.0/16, got %q", cfg.ClusterCIDR)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected LogLevel=debug, got %q", cfg.LogLevel)
	}
	if cfg.CNI != "ovn-kubernetes" {
		t.Errorf("expected CNI=ovn-kubernetes, got %q", cfg.CNI)
	}
	if cfg.EtcdMemoryLimit != 512 {
		t.Errorf("expected EtcdMemoryLimit=512, got %d", cfg.EtcdMemoryLimit)
	}
	// DNS should be recomputed from overridden ServiceCIDR.
	if cfg.DNSAddress != "10.100.0.10" {
		t.Errorf("expected DNSAddress=10.100.0.10, got %q", cfg.DNSAddress)
	}
	// DataDir should keep its default since it wasn't in the YAML.
	if cfg.DataDir != "/var/lib/microshift" {
		t.Errorf("expected DataDir=/var/lib/microshift, got %q", cfg.DataDir)
	}
}

func TestLoadConfigPartialOverride(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	yamlContent := `
clusterName: "edge-cluster"
nodeIP: "10.0.0.5"
`
	if err := os.WriteFile(cfgPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("writing test config: %v", err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.ClusterName != "edge-cluster" {
		t.Errorf("expected ClusterName=edge-cluster, got %q", cfg.ClusterName)
	}
	if cfg.NodeIP != "10.0.0.5" {
		t.Errorf("expected NodeIP=10.0.0.5, got %q", cfg.NodeIP)
	}
	// Defaults should be preserved.
	if cfg.BaseDomain != "microshift.local" {
		t.Errorf("expected default BaseDomain, got %q", cfg.BaseDomain)
	}
	if cfg.CNI != "kindnet" {
		t.Errorf("expected default CNI, got %q", cfg.CNI)
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for missing config file")
	}
}

func TestValidateValid(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.NodeIP = "192.168.1.1"
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected valid config, got error: %v", err)
	}
}

func TestValidateInvalidServiceCIDR(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.ServiceCIDR = "not-a-cidr"
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid ServiceCIDR")
	}
}

func TestValidateInvalidClusterCIDR(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.ClusterCIDR = "also-not-a-cidr"
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid ClusterCIDR")
	}
}

func TestValidateInvalidCNI(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.CNI = "calico"
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid CNI")
	}
}

func TestValidateEmptyClusterName(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.ClusterName = ""
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty ClusterName")
	}
}

func TestDNSServiceIP(t *testing.T) {
	tests := []struct {
		name        string
		serviceCIDR string
		expected    string
	}{
		{"default", "10.96.0.0/16", "10.96.0.10"},
		{"custom", "10.100.0.0/16", "10.100.0.10"},
		{"small subnet", "172.30.0.0/24", "172.30.0.10"},
		{"class A", "10.0.0.0/8", "10.0.0.10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{ServiceCIDR: tt.serviceCIDR}
			got := cfg.DNSServiceIP()
			if got != tt.expected {
				t.Errorf("DNSServiceIP() for %s: got %q, want %q", tt.serviceCIDR, got, tt.expected)
			}
		})
	}
}

func TestDirPaths(t *testing.T) {
	cfg := &Config{DataDir: "/var/lib/microshift"}

	if got := cfg.CertDir(); got != "/var/lib/microshift/certs" {
		t.Errorf("CertDir() = %q", got)
	}
	if got := cfg.KubeconfigDir(); got != "/var/lib/microshift/kubeconfig" {
		t.Errorf("KubeconfigDir() = %q", got)
	}
	if got := cfg.ComponentConfigDir(); got != "/var/lib/microshift/config" {
		t.Errorf("ComponentConfigDir() = %q", got)
	}
	if got := cfg.EtcdDataDir(); got != "/var/lib/microshift/etcd" {
		t.Errorf("EtcdDataDir() = %q", got)
	}
}
