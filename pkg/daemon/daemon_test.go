package daemon

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ausil/microshift-2.0/internal/testutil"
	"github.com/ausil/microshift-2.0/pkg/config"
)

type fakeSystemdManager struct {
	started     []string
	stopped     []string
	waited      []string
	reloaded    bool
	startErr    map[string]error
	stopErr     map[string]error
	waitErr     map[string]error
	reloadErr   error
}

func newFakeManager() *fakeSystemdManager {
	return &fakeSystemdManager{
		startErr: make(map[string]error),
		stopErr:  make(map[string]error),
		waitErr:  make(map[string]error),
	}
}

func (f *fakeSystemdManager) StartService(name string) error {
	if err, ok := f.startErr[name]; ok {
		return err
	}
	f.started = append(f.started, name)
	return nil
}

func (f *fakeSystemdManager) StopService(name string) error {
	if err, ok := f.stopErr[name]; ok {
		f.stopped = append(f.stopped, name)
		return err
	}
	f.stopped = append(f.stopped, name)
	return nil
}

func (f *fakeSystemdManager) RestartService(name string) error {
	return nil
}

func (f *fakeSystemdManager) IsActive(name string) (bool, error) {
	return true, nil
}

func (f *fakeSystemdManager) WaitForReady(name string, timeout time.Duration) error {
	if err, ok := f.waitErr[name]; ok {
		return err
	}
	f.waited = append(f.waited, name)
	return nil
}

func (f *fakeSystemdManager) DaemonReload() error {
	f.reloaded = true
	return f.reloadErr
}

func (f *fakeSystemdManager) Close() {}

func TestNew(t *testing.T) {
	cfg := config.NewDefaultConfig()
	d := New(cfg)
	if d == nil {
		t.Fatal("New returned nil")
	}
	if d.cfg != cfg {
		t.Error("daemon config doesn't match")
	}
	if d.stopCh == nil {
		t.Error("stopCh is nil")
	}
}

func TestNewWithManager(t *testing.T) {
	cfg := config.NewDefaultConfig()
	mgr := newFakeManager()
	d := NewWithManager(cfg, mgr)
	if d.svcMgr != mgr {
		t.Error("svcMgr doesn't match injected manager")
	}
}

func TestGenerateComponentConfigs(t *testing.T) {
	cfg := testutil.NewTestConfig(t)

	d := New(cfg)
	if err := d.generateComponentConfigs(); err != nil {
		t.Fatalf("generateComponentConfigs: %v", err)
	}

	expectedFiles := []string{
		"etcd.yaml", "etcd.env",
		"apiserver.env",
		"controller-manager.env",
		"scheduler.env",
		"kubelet-config.yaml", "kubelet.env",
	}

	for _, f := range expectedFiles {
		path := cfg.ComponentConfigDir() + "/" + f
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected file %s: %v", f, err)
		}
	}
}

func TestComponentServices(t *testing.T) {
	expected := []string{
		"microshift-etcd.service",
		"microshift-apiserver.service",
		"microshift-controller-manager.service",
		"microshift-scheduler.service",
		"microshift-kubelet.service",
	}
	if len(componentServices) != len(expected) {
		t.Fatalf("expected %d services, got %d", len(expected), len(componentServices))
	}
	for i, svc := range expected {
		if componentServices[i] != svc {
			t.Errorf("service[%d]: expected %s, got %s", i, svc, componentServices[i])
		}
	}
}

func TestStop(t *testing.T) {
	cfg := config.NewDefaultConfig()
	d := New(cfg)

	d.Stop()

	select {
	case _, ok := <-d.stopCh:
		if ok {
			t.Error("stopCh should be closed")
		}
	default:
		t.Error("stopCh should be readable after Stop")
	}
}

func TestStopComponentsNilManager(t *testing.T) {
	cfg := config.NewDefaultConfig()
	d := New(cfg)
	d.stopComponents()
}

func TestSdNotifyNoSocket(t *testing.T) {
	t.Setenv("NOTIFY_SOCKET", "")
	sdNotify("READY=1")
}

func TestSdNotifyInvalidSocket(t *testing.T) {
	t.Setenv("NOTIFY_SOCKET", "/tmp/nonexistent-microshift-test.sock")
	sdNotify("READY=1")
}

func TestNewInitializesStopCh(t *testing.T) {
	cfg := config.NewDefaultConfig()
	d := New(cfg)

	select {
	case <-d.stopCh:
		t.Error("stopCh should not be closed on a new daemon")
	default:
	}
}

func TestGenerateComponentConfigsCreatesAllFiles(t *testing.T) {
	cfg := testutil.NewTestConfig(t)
	d := New(cfg)

	if err := d.generateComponentConfigs(); err != nil {
		t.Fatalf("generateComponentConfigs: %v", err)
	}

	files := map[string]string{
		"etcd.yaml":              "name:",
		"etcd.env":               "ETCD_",
		"apiserver.env":          "KUBE_APISERVER_ARGS",
		"controller-manager.env": "KUBE_CONTROLLER_MANAGER_ARGS",
		"scheduler.env":          "KUBE_SCHEDULER_ARGS",
		"kubelet-config.yaml":    "KubeletConfiguration",
		"kubelet.env":            "KUBELET_ARGS",
	}

	for f, expected := range files {
		path := cfg.ComponentConfigDir() + "/" + f
		testutil.FileContains(t, path, expected)
	}
}

func TestComponentServicesOrder(t *testing.T) {
	if componentServices[0] != "microshift-etcd.service" {
		t.Errorf("expected etcd first, got %s", componentServices[0])
	}
	if componentServices[1] != "microshift-apiserver.service" {
		t.Errorf("expected apiserver second, got %s", componentServices[1])
	}
	last := componentServices[len(componentServices)-1]
	if last != "microshift-kubelet.service" {
		t.Errorf("expected kubelet last, got %s", last)
	}
}

// --- New tests using fakeSystemdManager ---

func TestStartComponentsOrder(t *testing.T) {
	cfg := config.NewDefaultConfig()
	mgr := newFakeManager()
	d := NewWithManager(cfg, mgr)

	if err := d.startComponents(); err != nil {
		t.Fatalf("startComponents: %v", err)
	}

	if !mgr.reloaded {
		t.Error("DaemonReload was not called")
	}

	expectedStarts := []string{
		"microshift-etcd.service",
		"microshift-apiserver.service",
		"microshift-controller-manager.service",
		"microshift-scheduler.service",
		"microshift-kubelet.service",
	}
	if len(mgr.started) != len(expectedStarts) {
		t.Fatalf("expected %d starts, got %d: %v", len(expectedStarts), len(mgr.started), mgr.started)
	}
	for i, svc := range expectedStarts {
		if mgr.started[i] != svc {
			t.Errorf("start[%d]: expected %s, got %s", i, svc, mgr.started[i])
		}
	}

	expectedWaits := []string{
		"microshift-etcd.service",
		"microshift-apiserver.service",
	}
	if len(mgr.waited) != len(expectedWaits) {
		t.Fatalf("expected %d waits, got %d: %v", len(expectedWaits), len(mgr.waited), mgr.waited)
	}
	for i, svc := range expectedWaits {
		if mgr.waited[i] != svc {
			t.Errorf("wait[%d]: expected %s, got %s", i, svc, mgr.waited[i])
		}
	}
}

func TestStartComponentsEtcdStartFailure(t *testing.T) {
	cfg := config.NewDefaultConfig()
	mgr := newFakeManager()
	mgr.startErr["microshift-etcd.service"] = fmt.Errorf("etcd failed")
	d := NewWithManager(cfg, mgr)

	err := d.startComponents()
	if err == nil {
		t.Fatal("expected error from etcd start failure")
	}
	if !strings.Contains(err.Error(), "etcd") {
		t.Errorf("error should mention etcd: %v", err)
	}
	if len(mgr.started) != 0 {
		t.Errorf("no services should have been recorded as started, got %v", mgr.started)
	}
}

func TestStartComponentsAPIServerWaitTimeout(t *testing.T) {
	cfg := config.NewDefaultConfig()
	mgr := newFakeManager()
	mgr.waitErr["microshift-apiserver.service"] = fmt.Errorf("timed out")
	d := NewWithManager(cfg, mgr)

	err := d.startComponents()
	if err == nil {
		t.Fatal("expected error from apiserver wait timeout")
	}
	if !strings.Contains(err.Error(), "apiserver") {
		t.Errorf("error should mention apiserver: %v", err)
	}
	// etcd and apiserver should have started, but remaining should not
	if len(mgr.started) != 2 {
		t.Errorf("expected 2 services started (etcd + apiserver), got %d: %v", len(mgr.started), mgr.started)
	}
}

func TestStopComponentsReverseOrder(t *testing.T) {
	cfg := config.NewDefaultConfig()
	mgr := newFakeManager()
	d := NewWithManager(cfg, mgr)

	d.stopComponents()

	expectedStops := []string{
		"microshift-kubelet.service",
		"microshift-scheduler.service",
		"microshift-controller-manager.service",
		"microshift-apiserver.service",
		"microshift-etcd.service",
	}
	if len(mgr.stopped) != len(expectedStops) {
		t.Fatalf("expected %d stops, got %d: %v", len(expectedStops), len(mgr.stopped), mgr.stopped)
	}
	for i, svc := range expectedStops {
		if mgr.stopped[i] != svc {
			t.Errorf("stop[%d]: expected %s, got %s", i, svc, mgr.stopped[i])
		}
	}
}

func TestStopComponentsPartialFailure(t *testing.T) {
	cfg := config.NewDefaultConfig()
	mgr := newFakeManager()
	mgr.stopErr["microshift-scheduler.service"] = fmt.Errorf("stop failed")
	d := NewWithManager(cfg, mgr)

	d.stopComponents()

	if len(mgr.stopped) != 5 {
		t.Errorf("all 5 services should be attempted, got %d: %v", len(mgr.stopped), mgr.stopped)
	}
}

func TestStopDoubleStop(t *testing.T) {
	cfg := config.NewDefaultConfig()
	d := New(cfg)

	d.Stop()
	d.Stop() // should not panic thanks to sync.Once
}

func TestGenerateComponentConfigsIdempotent(t *testing.T) {
	cfg := testutil.NewTestConfig(t)
	d := New(cfg)

	if err := d.generateComponentConfigs(); err != nil {
		t.Fatalf("first generateComponentConfigs: %v", err)
	}
	if err := d.generateComponentConfigs(); err != nil {
		t.Fatalf("second generateComponentConfigs: %v", err)
	}

	testutil.FileContains(t, cfg.ComponentConfigDir()+"/etcd.yaml", "name:")
	testutil.FileContains(t, cfg.ComponentConfigDir()+"/apiserver.env", "KUBE_APISERVER_ARGS")
}

func TestStartComponentsRemainingServiceFailure(t *testing.T) {
	cfg := config.NewDefaultConfig()
	mgr := newFakeManager()
	mgr.startErr["microshift-scheduler.service"] = fmt.Errorf("scheduler broken")
	d := NewWithManager(cfg, mgr)

	err := d.startComponents()
	if err == nil {
		t.Fatal("expected error from scheduler start failure")
	}
	if !strings.Contains(err.Error(), "scheduler") {
		t.Errorf("error should mention scheduler: %v", err)
	}
}
