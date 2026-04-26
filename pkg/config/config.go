package config

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds the MicroShift configuration.
type Config struct {
	ClusterName     string        `yaml:"clusterName"`
	BaseDomain      string        `yaml:"baseDomain"`
	NodeIP          string        `yaml:"nodeIP"`
	ServiceCIDR     string        `yaml:"serviceCIDR"`
	ClusterCIDR     string        `yaml:"clusterCIDR"`
	DNSAddress      string        `yaml:"dnsAddress"`
	DataDir         string        `yaml:"dataDir"`
	LogLevel        string        `yaml:"logLevel"`
	CNI             string        `yaml:"cni"`
	EtcdMemoryLimit int           `yaml:"etcdMemoryLimit"`
	Storage         StorageConfig `yaml:"storage"`
}

// StorageConfig configures the persistent storage driver.
type StorageConfig struct {
	Driver    string          `yaml:"driver"`
	LocalPath LocalPathConfig `yaml:"localPath"`
	LVMS      LVMSConfig      `yaml:"lvms"`
	NFS       NFSConfig       `yaml:"nfs"`
}

// LocalPathConfig configures the local-path-provisioner.
type LocalPathConfig struct {
	StoragePath string `yaml:"storagePath"`
}

// LVMSConfig configures TopoLVM/LVMS storage.
type LVMSConfig struct {
	VolumeGroup string `yaml:"volumeGroup"`
	Device      string `yaml:"device"`
}

// NFSConfig configures the NFS subdir provisioner.
type NFSConfig struct {
	Server string `yaml:"server"`
	Path   string `yaml:"path"`
}

// NewDefaultConfig returns a Config with all default values populated.
func NewDefaultConfig() *Config {
	c := &Config{
		ClusterName:     "microshift",
		BaseDomain:      "microshift.local",
		ServiceCIDR:     "10.96.0.0/16",
		ClusterCIDR:     "10.42.0.0/16",
		DataDir:         "/var/lib/microshift",
		LogLevel:        "info",
		CNI:             "kindnet",
		EtcdMemoryLimit: 0,
		Storage: StorageConfig{
			Driver: "local-path",
			LocalPath: LocalPathConfig{
				StoragePath: "/var/lib/microshift/storage",
			},
		},
	}
	c.DNSAddress = c.DNSServiceIP()
	return c
}

// LoadConfig loads configuration from a YAML file and fills in defaults
// for any fields that are not set.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	cfg := NewDefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Auto-detect NodeIP if not specified.
	if cfg.NodeIP == "" {
		ip, err := detectNodeIP()
		if err != nil {
			return nil, fmt.Errorf("auto-detecting node IP: %w", err)
		}
		cfg.NodeIP = ip
	}

	// Recompute derived fields.
	cfg.DNSAddress = cfg.DNSServiceIP()

	return cfg, nil
}

// Validate checks that the configuration is valid.
func (c *Config) Validate() error {
	if c.ClusterName == "" {
		return fmt.Errorf("clusterName must not be empty")
	}
	if c.BaseDomain == "" {
		return fmt.Errorf("baseDomain must not be empty")
	}
	if c.DataDir == "" {
		return fmt.Errorf("dataDir must not be empty")
	}

	if _, _, err := net.ParseCIDR(c.ServiceCIDR); err != nil {
		return fmt.Errorf("invalid serviceCIDR %q: %w", c.ServiceCIDR, err)
	}
	if _, _, err := net.ParseCIDR(c.ClusterCIDR); err != nil {
		return fmt.Errorf("invalid clusterCIDR %q: %w", c.ClusterCIDR, err)
	}

	switch c.CNI {
	case "kindnet", "ovn-kubernetes":
		// valid
	default:
		return fmt.Errorf("invalid CNI %q: must be \"kindnet\" or \"ovn-kubernetes\"", c.CNI)
	}

	if c.EtcdMemoryLimit < 0 {
		return fmt.Errorf("etcdMemoryLimit must be >= 0")
	}

	if err := c.Storage.Validate(); err != nil {
		return fmt.Errorf("storage: %w", err)
	}

	return nil
}

// Validate checks the storage configuration.
func (s *StorageConfig) Validate() error {
	switch s.Driver {
	case "local-path":
		if s.LocalPath.StoragePath == "" {
			return fmt.Errorf("localPath.storagePath must not be empty")
		}
	case "lvms":
		if s.LVMS.VolumeGroup == "" {
			return fmt.Errorf("lvms.volumeGroup is required when driver is lvms")
		}
	case "nfs":
		if s.NFS.Server == "" {
			return fmt.Errorf("nfs.server is required when driver is nfs")
		}
		if s.NFS.Path == "" {
			return fmt.Errorf("nfs.path is required when driver is nfs")
		}
	case "none":
		// no storage provisioner
	default:
		return fmt.Errorf("invalid storage driver %q: must be local-path, lvms, nfs, or none", s.Driver)
	}
	return nil
}

// DNSServiceIP computes the DNS service IP from ServiceCIDR.
// It returns the 10th IP in the service CIDR (base + 10 offset, 0-indexed).
// For example, 10.96.0.0/16 yields 10.96.0.10.
func (c *Config) DNSServiceIP() string {
	_, svcNet, err := net.ParseCIDR(c.ServiceCIDR)
	if err != nil {
		return ""
	}

	ip := svcNet.IP.To4()
	if ip == nil {
		return ""
	}

	ipInt := binary.BigEndian.Uint32(ip)
	ipInt += 10 // 10th IP (0-indexed offset of 10)
	result := make(net.IP, 4)
	binary.BigEndian.PutUint32(result, ipInt)

	return result.String()
}

// CertDir returns the directory for TLS certificates.
func (c *Config) CertDir() string {
	return filepath.Join(c.DataDir, "certs")
}

// KubeconfigDir returns the directory for kubeconfig files.
func (c *Config) KubeconfigDir() string {
	return filepath.Join(c.DataDir, "kubeconfig")
}

// ComponentConfigDir returns the directory for component configuration files.
func (c *Config) ComponentConfigDir() string {
	return filepath.Join(c.DataDir, "config")
}

// EtcdDataDir returns the directory for etcd data.
func (c *Config) EtcdDataDir() string {
	return filepath.Join(c.DataDir, "etcd")
}

// detectNodeIP finds the first non-loopback IPv4 address on the system.
func detectNodeIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", fmt.Errorf("listing interface addresses: %w", err)
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		ip := ipNet.IP
		if ip.IsLoopback() {
			continue
		}
		if ip.To4() == nil {
			continue
		}
		return ip.String(), nil
	}

	return "", fmt.Errorf("no non-loopback IPv4 address found")
}
