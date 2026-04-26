// Package services manages systemd services and generates configuration
// files for MicroShift's Kubernetes components.
package services

import (
	"context"
	"fmt"
	"time"

	"github.com/coreos/go-systemd/v22/dbus"
)

// ServiceManager manages systemd services via D-Bus.
type ServiceManager struct {
	conn *dbus.Conn
}

// ServiceStatus represents the current status of a systemd service.
type ServiceStatus struct {
	Name     string
	Active   bool
	SubState string
}

// NewServiceManager connects to the system D-Bus and returns a ServiceManager.
func NewServiceManager() (*ServiceManager, error) {
	ctx := context.Background()
	conn, err := dbus.NewWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("connecting to system D-Bus: %w", err)
	}
	return &ServiceManager{conn: conn}, nil
}

// Close closes the D-Bus connection.
func (m *ServiceManager) Close() {
	if m.conn != nil {
		m.conn.Close()
	}
}

// StartService starts the named systemd unit.
func (m *ServiceManager) StartService(name string) error {
	ctx := context.Background()
	ch := make(chan string, 1)
	_, err := m.conn.StartUnitContext(ctx, name, "replace", ch)
	if err != nil {
		return fmt.Errorf("starting %s: %w", name, err)
	}
	result := <-ch
	if result != "done" {
		return fmt.Errorf("starting %s: job result %s", name, result)
	}
	return nil
}

// StopService stops the named systemd unit.
func (m *ServiceManager) StopService(name string) error {
	ctx := context.Background()
	ch := make(chan string, 1)
	_, err := m.conn.StopUnitContext(ctx, name, "replace", ch)
	if err != nil {
		return fmt.Errorf("stopping %s: %w", name, err)
	}
	result := <-ch
	if result != "done" {
		return fmt.Errorf("stopping %s: job result %s", name, result)
	}
	return nil
}

// RestartService restarts the named systemd unit.
func (m *ServiceManager) RestartService(name string) error {
	ctx := context.Background()
	ch := make(chan string, 1)
	_, err := m.conn.RestartUnitContext(ctx, name, "replace", ch)
	if err != nil {
		return fmt.Errorf("restarting %s: %w", name, err)
	}
	result := <-ch
	if result != "done" {
		return fmt.Errorf("restarting %s: job result %s", name, result)
	}
	return nil
}

// IsActive returns true if the named systemd unit is active (running).
func (m *ServiceManager) IsActive(name string) (bool, error) {
	ctx := context.Background()
	units, err := m.conn.ListUnitsByNamesContext(ctx, []string{name})
	if err != nil {
		return false, fmt.Errorf("checking status of %s: %w", name, err)
	}
	if len(units) == 0 {
		return false, nil
	}
	return units[0].ActiveState == "active", nil
}

// WaitForReady polls IsActive for the named unit until it is active or the
// timeout expires.
func (m *ServiceManager) WaitForReady(name string, timeout time.Duration) error {
	deadline := time.After(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		active, err := m.IsActive(name)
		if err != nil {
			return err
		}
		if active {
			return nil
		}

		select {
		case <-deadline:
			return fmt.Errorf("timed out waiting for %s to become active after %s", name, timeout)
		case <-ticker.C:
		}
	}
}

// DaemonReload reloads systemd unit files (equivalent to systemctl daemon-reload).
func (m *ServiceManager) DaemonReload() error {
	ctx := context.Background()
	if err := m.conn.ReloadContext(ctx); err != nil {
		return fmt.Errorf("daemon-reload: %w", err)
	}
	return nil
}
