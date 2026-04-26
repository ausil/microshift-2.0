# Development Guide

## Cloning the repository

```bash
git clone https://github.com/ausil/microshift-2.0.git
cd microshift-2.0
```

## Building

Build the `microshift` binary:

```bash
make build
```

The binary is written to `bin/microshift`. Version and git commit are embedded via ldflags.

## Running tests

```bash
make test     # unit tests
make vet      # go vet
make lint     # golangci-lint (requires golangci-lint installed)
make e2e      # end-to-end smoke test (requires running system)
```

## Code formatting

```bash
make fmt
```

## Project structure

```
cmd/microshift/          CLI entry point (cobra commands)
pkg/config/              Configuration loading, defaults, and validation
pkg/certs/               TLS certificate generation and rotation
pkg/kubeconfig/          Kubeconfig file generation
pkg/services/            Systemd service management and component config generation
pkg/bootstrap/           Cluster bootstrap (applying kindnet, CoreDNS, RBAC manifests)
pkg/healthcheck/         Component health monitoring
pkg/daemon/              Main daemon orchestration loop
pkg/version/             Version information (set via ldflags)
assets/                  Embedded Kubernetes manifests (kindnet, CoreDNS, RBAC)
packaging/systemd/       Systemd unit files for all components
packaging/config/        Default configuration file
packaging/rpm/           RPM spec file for Fedora packaging
images/                  Bootc Containerfile, Fedora spin kickstart
test/                    Test files including e2e smoke tests
docs/                    Project documentation
```

## How it works

The `microshift run` command starts the main daemon loop, which:

1. Loads and validates configuration from `/etc/microshift/config.yaml`
2. Generates TLS certificates and kubeconfig files under `/var/lib/microshift/`
3. Writes environment files for each Kubernetes component (etcd, apiserver, etc.)
4. Starts component systemd services in dependency order via the D-Bus API
5. Bootstraps the cluster by applying embedded manifests (kindnet, CoreDNS)
6. Enters a health monitoring loop, watching component health and handling cert rotation

## Installing locally

To install the binary, systemd units, config, and assets to the system:

```bash
sudo make install
```

To remove:

```bash
sudo make uninstall
```
