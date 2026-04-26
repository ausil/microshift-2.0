package kubeconfig

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ausil/microshift-2.0/pkg/certs"
	"github.com/ausil/microshift-2.0/pkg/config"
	"gopkg.in/yaml.v3"
)

// KubeconfigData represents the standard kubeconfig YAML structure.
type KubeconfigData struct {
	APIVersion     string           `yaml:"apiVersion"`
	Kind           string           `yaml:"kind"`
	Clusters       []ClusterEntry   `yaml:"clusters"`
	Users          []UserEntry      `yaml:"users"`
	Contexts       []ContextEntry   `yaml:"contexts"`
	CurrentContext string           `yaml:"current-context"`
}

// ClusterEntry is one entry in the clusters list.
type ClusterEntry struct {
	Name    string  `yaml:"name"`
	Cluster Cluster `yaml:"cluster"`
}

// Cluster holds cluster connection information.
type Cluster struct {
	Server                   string `yaml:"server"`
	CertificateAuthorityData string `yaml:"certificate-authority-data"`
}

// UserEntry is one entry in the users list.
type UserEntry struct {
	Name string `yaml:"name"`
	User User   `yaml:"user"`
}

// User holds user authentication credentials.
type User struct {
	ClientCertificateData string `yaml:"client-certificate-data"`
	ClientKeyData         string `yaml:"client-key-data"`
}

// ContextEntry is one entry in the contexts list.
type ContextEntry struct {
	Name    string  `yaml:"name"`
	Context Context `yaml:"context"`
}

// Context binds a cluster to a user.
type Context struct {
	Cluster string `yaml:"cluster"`
	User    string `yaml:"user"`
}

// GenerateKubeconfig produces a kubeconfig YAML document as bytes.
func GenerateKubeconfig(clusterName, apiServerURL string, caCertPEM, clientCertPEM, clientKeyPEM []byte) ([]byte, error) {
	kc := KubeconfigData{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: []ClusterEntry{
			{
				Name: clusterName,
				Cluster: Cluster{
					Server:                   apiServerURL,
					CertificateAuthorityData: base64.StdEncoding.EncodeToString(caCertPEM),
				},
			},
		},
		Users: []UserEntry{
			{
				Name: clusterName + "-user",
				User: User{
					ClientCertificateData: base64.StdEncoding.EncodeToString(clientCertPEM),
					ClientKeyData:         base64.StdEncoding.EncodeToString(clientKeyPEM),
				},
			},
		},
		Contexts: []ContextEntry{
			{
				Name: clusterName,
				Context: Context{
					Cluster: clusterName,
					User:    clusterName + "-user",
				},
			},
		},
		CurrentContext: clusterName,
	}

	data, err := yaml.Marshal(kc)
	if err != nil {
		return nil, fmt.Errorf("marshaling kubeconfig: %w", err)
	}

	return data, nil
}

// GenerateAllKubeconfigs generates and writes all kubeconfig files needed for
// the cluster to cfg.KubeconfigDir().
func GenerateAllKubeconfigs(cfg *config.Config, clusterCerts *certs.ClusterCerts) error {
	dir := cfg.KubeconfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating kubeconfig dir: %w", err)
	}

	apiServerURL := "https://127.0.0.1:6443"
	caCertPEM := clusterCerts.RootCA.CertPEM

	configs := []struct {
		name    string
		cert    *certs.Certificate
	}{
		{"admin", clusterCerts.AdminCert},
		{"controller-manager", clusterCerts.ControllerManagerCert},
		{"scheduler", clusterCerts.SchedulerCert},
		{"kubelet", clusterCerts.AdminCert},
	}

	for _, kc := range configs {
		data, err := GenerateKubeconfig(cfg.ClusterName, apiServerURL, caCertPEM, kc.cert.CertPEM, kc.cert.KeyPEM)
		if err != nil {
			return fmt.Errorf("generating %s kubeconfig: %w", kc.name, err)
		}

		path := filepath.Join(dir, kc.name+".kubeconfig")
		if err := os.WriteFile(path, data, 0600); err != nil {
			return fmt.Errorf("writing %s kubeconfig: %w", kc.name, err)
		}
	}

	return nil
}
