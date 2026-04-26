package services

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ausil/microshift-2.0/pkg/config"
)

// GenerateSchedulerConfig writes the scheduler environment file:
//   - scheduler.env — KUBE_SCHEDULER_ARGS with required flags
func GenerateSchedulerConfig(cfg *config.Config) error {
	configDir := cfg.ComponentConfigDir()
	kubeconfigDir := cfg.KubeconfigDir()

	args := fmt.Sprintf(
		"--kubeconfig=%s",
		filepath.Join(kubeconfigDir, "scheduler.kubeconfig"),
	)

	env := fmt.Sprintf("KUBE_SCHEDULER_ARGS=\"%s\"\n", args)

	if err := os.WriteFile(filepath.Join(configDir, "scheduler.env"), []byte(env), 0644); err != nil {
		return fmt.Errorf("writing scheduler.env: %w", err)
	}

	return nil
}
