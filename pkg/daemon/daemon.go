package daemon

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ausil/microshift-2.0/pkg/bootstrap"
	"github.com/ausil/microshift-2.0/pkg/certs"
	"github.com/ausil/microshift-2.0/pkg/config"
	"github.com/ausil/microshift-2.0/pkg/healthcheck"
	"github.com/ausil/microshift-2.0/pkg/kubeconfig"
	"github.com/ausil/microshift-2.0/pkg/services"
)

type Daemon struct {
	cfg     *config.Config
	svcMgr  *services.ServiceManager
	health  *healthcheck.HealthChecker
	stopCh  chan struct{}
}

var componentServices = []string{
	"microshift-etcd.service",
	"microshift-apiserver.service",
	"microshift-controller-manager.service",
	"microshift-scheduler.service",
	"microshift-kubelet.service",
}

func New(cfg *config.Config) *Daemon {
	return &Daemon{
		cfg:    cfg,
		stopCh: make(chan struct{}),
	}
}

func (d *Daemon) Run() error {
	log.Printf("MicroShift starting with data dir %s", d.cfg.DataDir)

	// Create data directories
	for _, dir := range []string{
		d.cfg.CertDir(),
		d.cfg.KubeconfigDir(),
		d.cfg.ComponentConfigDir(),
		d.cfg.EtcdDataDir(),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}

	// Generate certificates
	log.Println("Generating TLS certificates")
	clusterCerts, err := certs.GenerateAllCerts(d.cfg)
	if err != nil {
		return fmt.Errorf("generating certificates: %w", err)
	}
	if err := certs.WriteAllCerts(d.cfg, clusterCerts); err != nil {
		return fmt.Errorf("writing certificates: %w", err)
	}

	// Generate kubeconfig files
	log.Println("Generating kubeconfig files")
	if err := kubeconfig.GenerateAllKubeconfigs(d.cfg, clusterCerts); err != nil {
		return fmt.Errorf("generating kubeconfigs: %w", err)
	}

	// Generate component configurations
	log.Println("Generating component configurations")
	if err := d.generateComponentConfigs(); err != nil {
		return fmt.Errorf("generating component configs: %w", err)
	}

	// Connect to systemd
	svcMgr, err := services.NewServiceManager()
	if err != nil {
		return fmt.Errorf("connecting to systemd: %w", err)
	}
	d.svcMgr = svcMgr
	defer svcMgr.Close()

	// Start component services
	log.Println("Starting component services")
	if err := d.startComponents(); err != nil {
		return fmt.Errorf("starting components: %w", err)
	}

	// Bootstrap cluster
	log.Println("Bootstrapping cluster")
	if err := bootstrap.Bootstrap(d.cfg); err != nil {
		return fmt.Errorf("bootstrapping cluster: %w", err)
	}

	// Notify systemd we're ready
	sdNotify("READY=1")
	log.Println("MicroShift is ready")

	// Set up health checker
	kubeconfigPath := filepath.Join(d.cfg.KubeconfigDir(), "admin.kubeconfig")
	d.health = healthcheck.NewHealthChecker(kubeconfigPath, d.cfg.CertDir())

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	// Health monitoring loop
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case sig := <-sigCh:
			log.Printf("Received signal %s, shutting down", sig)
			d.stopComponents()
			return nil
		case <-ticker.C:
			sdNotify("WATCHDOG=1")
		case <-d.stopCh:
			d.stopComponents()
			return nil
		}
	}
}

func (d *Daemon) generateComponentConfigs() error {
	generators := []func(*config.Config) error{
		services.GenerateEtcdConfig,
		services.GenerateAPIServerConfig,
		services.GenerateControllerManagerConfig,
		services.GenerateSchedulerConfig,
		services.GenerateKubeletConfig,
	}

	for _, gen := range generators {
		if err := gen(d.cfg); err != nil {
			return err
		}
	}
	return nil
}

func (d *Daemon) startComponents() error {
	if err := d.svcMgr.DaemonReload(); err != nil {
		return fmt.Errorf("daemon-reload: %w", err)
	}

	// Start etcd first and wait for it
	log.Println("Starting etcd")
	if err := d.svcMgr.StartService("microshift-etcd.service"); err != nil {
		return fmt.Errorf("starting etcd: %w", err)
	}
	if err := d.svcMgr.WaitForReady("microshift-etcd.service", 60*time.Second); err != nil {
		return fmt.Errorf("waiting for etcd: %w", err)
	}

	// Start API server and wait
	log.Println("Starting kube-apiserver")
	if err := d.svcMgr.StartService("microshift-apiserver.service"); err != nil {
		return fmt.Errorf("starting apiserver: %w", err)
	}
	if err := d.svcMgr.WaitForReady("microshift-apiserver.service", 60*time.Second); err != nil {
		return fmt.Errorf("waiting for apiserver: %w", err)
	}

	// Start remaining services
	remaining := []string{
		"microshift-controller-manager.service",
		"microshift-scheduler.service",
		"microshift-kubelet.service",
	}
	for _, svc := range remaining {
		log.Printf("Starting %s", svc)
		if err := d.svcMgr.StartService(svc); err != nil {
			return fmt.Errorf("starting %s: %w", svc, err)
		}
	}

	return nil
}

func (d *Daemon) stopComponents() {
	if d.svcMgr == nil {
		return
	}

	// Stop in reverse order
	for i := len(componentServices) - 1; i >= 0; i-- {
		svc := componentServices[i]
		log.Printf("Stopping %s", svc)
		if err := d.svcMgr.StopService(svc); err != nil {
			log.Printf("Warning: failed to stop %s: %v", svc, err)
		}
	}
}

func (d *Daemon) Stop() {
	close(d.stopCh)
}

func sdNotify(state string) {
	sockAddr := os.Getenv("NOTIFY_SOCKET")
	if sockAddr == "" {
		return
	}
	sock, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return
	}
	defer syscall.Close(sock)
	addr := &syscall.SockaddrUnix{Name: sockAddr}
	_ = syscall.Sendto(sock, []byte(state), 0, addr)
}
