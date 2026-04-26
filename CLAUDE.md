# MicroShift 2.0

Lightweight single-node Kubernetes for edge, built on Fedora-packaged components.

## What is this?

A from-scratch reimplementation of MicroShift. Instead of embedding K8s components as Go libraries in a monolithic binary, this project is a thin Go daemon that orchestrates separate systemd services using binaries from Fedora's `kubernetes1.35` packages.

## Architecture

MicroShift is a daemon (`microshift.service`) that:
1. Generates TLS certificates and kubeconfig files
2. Writes configuration for each K8s component
3. Manages component lifecycle via systemd D-Bus
4. Bootstraps the cluster (applies kindnet, CoreDNS manifests)
5. Monitors health and handles cert rotation

Components run as separate systemd services:
- `microshift-etcd.service`
- `microshift-apiserver.service`
- `microshift-controller-manager.service`
- `microshift-scheduler.service`
- `microshift-kubelet.service`

All have `PartOf=microshift.service` — stopping MicroShift stops everything.

## Build

```bash
make build    # builds bin/microshift
make test     # runs unit tests
make lint     # runs golangci-lint
make install  # installs binary, systemd units, assets
```

## Project layout

- `cmd/microshift/` — CLI entry point
- `pkg/config/` — configuration loading and defaults
- `pkg/certs/` — TLS certificate generation
- `pkg/kubeconfig/` — kubeconfig file generation
- `pkg/services/` — systemd service management and component config generation
- `pkg/bootstrap/` — cluster bootstrap (manifest application)
- `pkg/healthcheck/` — component health monitoring
- `pkg/daemon/` — main daemon orchestration loop
- `assets/` — embedded K8s manifests (kindnet, coredns, rbac)
- `packaging/` — RPM spec, systemd units, default config
- `images/` — bootc Containerfile, Fedora spin kickstart

## Key conventions

- Go module: `github.com/ausil/microshift-2.0`
- Config file: `/etc/microshift/config.yaml`
- Data directory: `/var/lib/microshift/` (certs, kubeconfigs, component configs, etcd data)
- Targets Fedora IoT with kubernetes1.35, etcd, cri-o1.35, containernetworking-plugins
- Kindnet and CoreDNS run as pods (not host packages)
- Apache-2.0 license

## Dependencies

Host packages (from Fedora repos):
- `kubernetes1.35` — kube-apiserver, kube-controller-manager, kube-scheduler, kubelet
- `etcd` — cluster state store
- `cri-o1.35` — container runtime
- `containernetworking-plugins` — CNI plugins (bridge, host-local)

Go dependencies:
- `k8s.io/client-go` — K8s API client for bootstrap
- `github.com/coreos/go-systemd/v22` — systemd D-Bus integration
- `gopkg.in/yaml.v3` — config file parsing
- `k8s.io/kubelet` — KubeletConfiguration types
