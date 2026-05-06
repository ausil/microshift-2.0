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

func TestDefaultStorageConfig(t *testing.T) {
	cfg := NewDefaultConfig()
	if cfg.Storage.Driver != "local-path" {
		t.Errorf("expected default storage driver local-path, got %q", cfg.Storage.Driver)
	}
	if cfg.Storage.LocalPath.StoragePath != "/var/lib/microshift/storage" {
		t.Errorf("expected default storage path, got %q", cfg.Storage.LocalPath.StoragePath)
	}
}

func TestValidateStorageLocalPath(t *testing.T) {
	cfg := NewDefaultConfig()
	if err := cfg.Validate(); err != nil {
		t.Errorf("default config should be valid: %v", err)
	}
}

func TestValidateStorageLVMS(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Storage.Driver = "lvms"
	cfg.Storage.LVMS.VolumeGroup = "microshift-vg"
	if err := cfg.Validate(); err != nil {
		t.Errorf("lvms config with VG should be valid: %v", err)
	}
}

func TestValidateStorageLVMSMissingVG(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Storage.Driver = "lvms"
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for lvms without volumeGroup")
	}
}

func TestValidateStorageNFS(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Storage.Driver = "nfs"
	cfg.Storage.NFS.Server = "192.168.1.10"
	cfg.Storage.NFS.Path = "/exports/microshift"
	if err := cfg.Validate(); err != nil {
		t.Errorf("nfs config with server+path should be valid: %v", err)
	}
}

func TestValidateStorageNFSMissing(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Storage.Driver = "nfs"
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for nfs without server")
	}
}

func TestValidateStorageNone(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Storage.Driver = "none"
	if err := cfg.Validate(); err != nil {
		t.Errorf("none driver should be valid: %v", err)
	}
}

func TestValidateStorageInvalidDriver(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Storage.Driver = "ceph"
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid storage driver")
	}
}

func TestValidateEmptyBaseDomain(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.BaseDomain = ""
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty BaseDomain")
	}
}

func TestValidateEmptyDataDir(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.DataDir = ""
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty DataDir")
	}
}

func TestValidateNegativeEtcdMemoryLimit(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.EtcdMemoryLimit = -1
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for negative EtcdMemoryLimit")
	}
}

func TestValidateValidLogLevels(t *testing.T) {
	cfg := NewDefaultConfig()
	// Default config should pass validation with default log level.
	if err := cfg.Validate(); err != nil {
		t.Errorf("default config should be valid: %v", err)
	}
}

func TestLoadConfigWithStorage(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	yamlContent := `
clusterName: "storage-test"
nodeIP: "10.0.0.1"
storage:
  driver: "lvms"
  lvms:
    volumeGroup: "test-vg"
    device: "/dev/sdb"
`
	if err := os.WriteFile(cfgPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("writing test config: %v", err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Storage.Driver != "lvms" {
		t.Errorf("expected driver=lvms, got %q", cfg.Storage.Driver)
	}
	if cfg.Storage.LVMS.VolumeGroup != "test-vg" {
		t.Errorf("expected volumeGroup=test-vg, got %q", cfg.Storage.LVMS.VolumeGroup)
	}
	if cfg.Storage.LVMS.Device != "/dev/sdb" {
		t.Errorf("expected device=/dev/sdb, got %q", cfg.Storage.LVMS.Device)
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(cfgPath, []byte("{{invalid yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig(cfgPath)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadConfigEmptyFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(cfgPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig with empty file should use defaults: %v", err)
	}

	// All defaults should be populated.
	if cfg.ClusterName != "microshift" {
		t.Errorf("expected default ClusterName, got %q", cfg.ClusterName)
	}
	if cfg.Storage.Driver != "local-path" {
		t.Errorf("expected default storage driver, got %q", cfg.Storage.Driver)
	}
}

func TestValidateStorageLocalPathEmptyPath(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Storage.Driver = "local-path"
	cfg.Storage.LocalPath.StoragePath = ""
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty local-path storagePath")
	}
}

func TestValidateStorageNFSMissingPath(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Storage.Driver = "nfs"
	cfg.Storage.NFS.Server = "192.168.1.10"
	cfg.Storage.NFS.Path = ""
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for nfs with server but no path")
	}
}

func TestValidateTableDriven(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
	}{
		{"valid default", func(c *Config) {}, false},
		{"empty clusterName", func(c *Config) { c.ClusterName = "" }, true},
		{"empty baseDomain", func(c *Config) { c.BaseDomain = "" }, true},
		{"empty dataDir", func(c *Config) { c.DataDir = "" }, true},
		{"invalid serviceCIDR", func(c *Config) { c.ServiceCIDR = "bad" }, true},
		{"invalid clusterCIDR", func(c *Config) { c.ClusterCIDR = "bad" }, true},
		{"invalid CNI", func(c *Config) { c.CNI = "calico" }, true},
		{"valid ovn-kubernetes", func(c *Config) { c.CNI = "ovn-kubernetes" }, false},
		{"negative etcdMemoryLimit", func(c *Config) { c.EtcdMemoryLimit = -1 }, true},
		{"zero etcdMemoryLimit", func(c *Config) { c.EtcdMemoryLimit = 0 }, false},
		{"storage driver none", func(c *Config) { c.Storage.Driver = "none" }, false},
		{"invalid storage driver", func(c *Config) { c.Storage.Driver = "ceph" }, true},
		{"lvms without VG", func(c *Config) { c.Storage.Driver = "lvms" }, true},
		{"lvms with VG", func(c *Config) {
			c.Storage.Driver = "lvms"
			c.Storage.LVMS.VolumeGroup = "vg0"
		}, false},
		{"nfs without server", func(c *Config) { c.Storage.Driver = "nfs" }, true},
		{"nfs with server+path", func(c *Config) {
			c.Storage.Driver = "nfs"
			c.Storage.NFS.Server = "10.0.0.1"
			c.Storage.NFS.Path = "/data"
		}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewDefaultConfig()
			tt.modify(cfg)
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestDNSServiceIPEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		serviceCIDR string
		expected    string
	}{
		{"empty string", "", ""},
		{"/32 CIDR", "10.96.0.0/32", "10.96.0.10"},
		{"192.168 range", "192.168.0.0/24", "192.168.0.10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{ServiceCIDR: tt.serviceCIDR}
			got := cfg.DNSServiceIP()
			if got != tt.expected {
				t.Errorf("DNSServiceIP() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestLoadConfigPreservesDefaults(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(cfgPath, []byte("clusterName: test\nnodeIP: \"10.0.0.1\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	defaults := NewDefaultConfig()
	if cfg.BaseDomain != defaults.BaseDomain {
		t.Errorf("BaseDomain: expected %q, got %q", defaults.BaseDomain, cfg.BaseDomain)
	}
	if cfg.ServiceCIDR != defaults.ServiceCIDR {
		t.Errorf("ServiceCIDR: expected %q, got %q", defaults.ServiceCIDR, cfg.ServiceCIDR)
	}
	if cfg.ClusterCIDR != defaults.ClusterCIDR {
		t.Errorf("ClusterCIDR: expected %q, got %q", defaults.ClusterCIDR, cfg.ClusterCIDR)
	}
	if cfg.DataDir != defaults.DataDir {
		t.Errorf("DataDir: expected %q, got %q", defaults.DataDir, cfg.DataDir)
	}
	if cfg.LogLevel != defaults.LogLevel {
		t.Errorf("LogLevel: expected %q, got %q", defaults.LogLevel, cfg.LogLevel)
	}
	if cfg.CNI != defaults.CNI {
		t.Errorf("CNI: expected %q, got %q", defaults.CNI, cfg.CNI)
	}
	if cfg.Storage.Driver != defaults.Storage.Driver {
		t.Errorf("Storage.Driver: expected %q, got %q", defaults.Storage.Driver, cfg.Storage.Driver)
	}
}

func TestDNSServiceIPInvalidCIDR(t *testing.T) {
	cfg := &Config{ServiceCIDR: "not-a-cidr"}
	got := cfg.DNSServiceIP()
	if got != "" {
		t.Errorf("expected empty string for invalid CIDR, got %q", got)
	}
}

func TestDirPathsCustomDataDir(t *testing.T) {
	cfg := &Config{DataDir: "/opt/microshift"}

	if got := cfg.CertDir(); got != "/opt/microshift/certs" {
		t.Errorf("CertDir() = %q", got)
	}
	if got := cfg.KubeconfigDir(); got != "/opt/microshift/kubeconfig" {
		t.Errorf("KubeconfigDir() = %q", got)
	}
	if got := cfg.ComponentConfigDir(); got != "/opt/microshift/config" {
		t.Errorf("ComponentConfigDir() = %q", got)
	}
	if got := cfg.EtcdDataDir(); got != "/opt/microshift/etcd" {
		t.Errorf("EtcdDataDir() = %q", got)
	}
}
