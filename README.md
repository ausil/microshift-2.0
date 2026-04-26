# MicroShift 2.0

[![Build and Test](https://github.com/ausil/microshift-2.0/actions/workflows/build.yml/badge.svg)](https://github.com/ausil/microshift-2.0/actions/workflows/build.yml)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

Lightweight single-node Kubernetes for edge computing, built on Fedora-packaged components.

## What is MicroShift 2.0?

MicroShift 2.0 is a from-scratch reimplementation of MicroShift that takes a fundamentally different approach to running Kubernetes on edge devices. Instead of embedding Kubernetes components as Go libraries in a monolithic binary, MicroShift 2.0 is a thin orchestration daemon that manages separate systemd services using binaries from Fedora's `kubernetes1.35` packages.

The `microshift` daemon handles certificate generation, kubeconfig creation, and component configuration, then delegates actual Kubernetes workloads to standard Fedora-packaged binaries (`kube-apiserver`, `etcd`, `kubelet`, etc.) running as independent systemd services. This means components can be updated independently through the system package manager, failures are isolated per-service, and the entire stack integrates naturally with systemd tooling.

MicroShift 2.0 targets Fedora IoT and CentOS Stream 10, with delivery via both bootc container images and traditional RPM packages.

## Features

- **Single-node Kubernetes** -- Full K8s control plane and worker on one machine, optimized for edge
- **Systemd-native** -- Each component runs as its own systemd service with proper dependency ordering
- **Fedora-packaged dependencies** -- Uses `kubernetes1.35`, `etcd`, `cri-o1.35` directly from Fedora repos
- **Automatic TLS** -- Generates all certificates and kubeconfig files on first start
- **Configurable storage** -- Local-path (default), TopoLVM/LVMS, or NFS provisioners
- **CNI options** -- Kindnet (default, zero-config) or OVN-Kubernetes
- **Minimal footprint** -- The MicroShift binary itself is small; heavy lifting is done by system packages
- **Simple operations** -- `systemctl start microshift` brings up the entire cluster

## Architecture

```
                    microshift.service
                          |
            generates certs, kubeconfigs,
            component configs, then starts:
                          |
          +---------------+---------------+
          |                               |
  microshift-etcd.service                 |
          |                               |
  microshift-apiserver.service            |
          |                               |
    +-----+-----+-----+                  |
    |           |     |                   |
  controller  scheduler  kubelet         kube-proxy
  -manager                               
```

All component services have `PartOf=microshift.service` -- stopping MicroShift cleanly tears down the entire stack.

## Quick Start

### Install dependencies

```bash
sudo dnf install kubernetes1.35 etcd cri-o1.35 containernetworking-plugins
```

### Install MicroShift

From RPM (when available):

```bash
sudo dnf install microshift
```

From source:

```bash
git clone https://github.com/ausil/microshift-2.0.git
cd microshift-2.0
make build
sudo make install
```

### Start the cluster

```bash
sudo systemctl enable --now crio
sudo systemctl enable --now microshift
```

### Verify

```bash
export KUBECONFIG=/var/lib/microshift/kubeconfig/admin.kubeconfig
kubectl get nodes
kubectl get pods -A
```

You should see your node in `Ready` state and system pods (kindnet, CoreDNS) running.

## Configuration

MicroShift reads its configuration from `/etc/microshift/config.yaml`. All fields are optional with sensible defaults. A minimal configuration only needs a node IP:

```yaml
clusterName: my-edge-cluster
nodeIP: "192.168.1.100"
```

See the [Configuration Reference](docs/configuration.md) for all available fields.

## Storage

MicroShift supports three storage drivers, configured via `storage.driver` in `config.yaml`:

| Driver | Use Case | Default |
|--------|----------|---------|
| `local-path` | Development, simple workloads | Yes |
| `lvms` | Production edge with dedicated disks | No |
| `nfs` | Shared storage environments | No |
| `none` | No persistent storage needed | No |

See the [Storage Guide](docs/storage.md) for setup details.

## Documentation

- [Getting Started](docs/getting-started.md) -- Installation and first steps
- [Configuration Reference](docs/configuration.md) -- All config fields and examples
- [Architecture](docs/architecture.md) -- Design, startup sequence, cert model
- [Storage Guide](docs/storage.md) -- Storage driver setup and comparison
- [Networking Guide](docs/networking.md) -- CNI, DNS, and firewall configuration
- [Troubleshooting](docs/troubleshooting.md) -- Common issues and debugging
- [Development Guide](docs/development.md) -- Building, testing, project structure
- [Contributing](CONTRIBUTING.md) -- How to contribute

## Building

```bash
make build      # builds bin/microshift
make test       # runs unit tests
make lint       # runs golangci-lint
make install    # installs binary, systemd units, assets
```

## License

Apache License 2.0. See [LICENSE](LICENSE) for details.
