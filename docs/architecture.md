# Architecture

## Design Philosophy

MicroShift 2.0 is a thin orchestration daemon, not a monolithic Kubernetes distribution. Rather than embedding Kubernetes components as Go libraries (as MicroShift 1.x does), it manages standard Fedora-packaged binaries (`kube-apiserver`, `etcd`, `kubelet`, etc.) as separate systemd services.

This design has several advantages:

- **Independent updates:** Components are updated through the system package manager, not by rebuilding MicroShift
- **Failure isolation:** A crash in one component doesn't bring down the rest
- **Systemd integration:** Standard `systemctl`, `journalctl`, and watchdog tooling works out of the box
- **Minimal binary:** The MicroShift binary handles orchestration only; heavy lifting is done by upstream binaries

## Component Overview

```
+---------------------------------------------------+
|                microshift.service                  |
|                                                    |
|  1. Generate TLS certificates                      |
|  2. Write kubeconfig files                         |
|  3. Generate component environment files           |
|  4. Start services via systemd D-Bus               |
|  5. Bootstrap cluster (apply manifests)            |
|  6. Health monitoring loop                         |
+---------------------------------------------------+
        |             |              |
        v             v              v
  +-----------+ +-----------+ +-----------+
  |   etcd    | | apiserver | |  kubelet  |
  +-----------+ +-----------+ +-----------+
                      |
              +-------+-------+
              v               v
        +-----------+   +-----------+
        | controller|   | scheduler |
        |  manager  |   |           |
        +-----------+   +-----------+
```

## Startup Sequence

The daemon follows a strict ordering when starting:

1. **Load configuration** -- Read `/etc/microshift/config.yaml`, apply defaults, validate
2. **Create data directories** -- Ensure `/var/lib/microshift/{certs,kubeconfig,config,etcd}` exist
3. **Generate TLS certificates** -- Create root CA, front-proxy CA, API server cert, kubelet client cert, controller-manager cert, scheduler cert, admin cert, and service account key pair
4. **Write certificates to disk** -- All certs written to `/var/lib/microshift/certs/`
5. **Generate kubeconfig files** -- Create admin, controller-manager, scheduler, and kubelet kubeconfigs in `/var/lib/microshift/kubeconfig/`
6. **Generate component configs** -- Write environment files and config files for each component to `/var/lib/microshift/config/`
7. **Connect to systemd D-Bus** -- Establish a connection to the system bus for service management
8. **Start etcd** -- Start `microshift-etcd.service` and wait for it to become active
9. **Start API server** -- Start `microshift-apiserver.service` and wait for it to become active
10. **Start remaining components** -- Start controller-manager, scheduler, and kubelet (no wait required, they'll retry connections)
11. **Bootstrap cluster** -- Wait for API server health (`/healthz`), then apply embedded manifests: RBAC, Kindnet, CoreDNS, storage provisioner
12. **Signal ready** -- Send `READY=1` via sd_notify to systemd
13. **Enter monitoring loop** -- Periodically send `WATCHDOG=1` and monitor component health

## Systemd Dependency Chain

```
cri-o.service
      |
      v
microshift.service  (Type=notify, WatchdogSec=300)
      |
      v
microshift-etcd.service  (After=microshift.service)
      |
      v
microshift-apiserver.service  (After=microshift-etcd.service)
      |
      +---> microshift-controller-manager.service
      |
      +---> microshift-scheduler.service
      |
      +---> microshift-kubelet.service  (After=cri-o.service)
```

All component services have `PartOf=microshift.service`. Stopping or restarting `microshift.service` automatically stops all components. The daemon stops components in reverse order (kubelet, scheduler, controller-manager, API server, etcd).

## Certificate Architecture

MicroShift generates all TLS certificates on first startup using Go's `crypto/x509` and `crypto/rsa` standard library packages.

### Certificate hierarchy

```
Root CA (ca.crt / ca.key)
  |
  +-- API server cert (apiserver.crt)
  |     SANs: kubernetes, kubernetes.default, kubernetes.default.svc,
  |           localhost, 127.0.0.1, <nodeIP>, <serviceCIDR first IP>
  |
  +-- API server kubelet client cert (apiserver-kubelet-client.crt)
  |     CN: kube-apiserver-kubelet-client, O: system:masters
  |
  +-- Controller manager client cert (controller-manager.crt)
  |     CN: system:kube-controller-manager
  |
  +-- Scheduler client cert (scheduler.crt)
  |     CN: system:kube-scheduler
  |
  +-- Admin client cert (admin.crt)
        CN: kubernetes-admin, O: system:masters

Front Proxy CA (front-proxy-ca.crt / front-proxy-ca.key)
  |
  +-- Front proxy client cert (front-proxy-client.crt)
        CN: front-proxy-client

Service Account Key Pair (sa.pub / sa.key)
  Used for signing and verifying ServiceAccount tokens
```

### Certificate files

All certificates are stored in `/var/lib/microshift/certs/`:

| File | Purpose |
|------|---------|
| `ca.crt` / `ca.key` | Root CA for cluster communication |
| `apiserver.crt` / `apiserver.key` | API server TLS and etcd client/server/peer cert |
| `apiserver-kubelet-client.crt` / `.key` | API server to kubelet authentication |
| `controller-manager.crt` / `.key` | Controller manager client auth |
| `scheduler.crt` / `.key` | Scheduler client auth |
| `admin.crt` / `.key` | Admin/kubectl client auth |
| `front-proxy-ca.crt` / `.key` | CA for API aggregation layer |
| `front-proxy-client.crt` / `.key` | Front proxy client cert |
| `sa.pub` / `sa.key` | ServiceAccount token signing key pair |

Key files are written with `0600` permissions. Certificate files are `0644`.

## Bootstrap Process

After the control plane is running, MicroShift bootstraps the cluster by applying embedded manifests via `kubectl apply`:

1. **Wait for API server** -- Poll `https://127.0.0.1:6443/healthz` until it returns 200 (up to 120 seconds)
2. **LVM setup** (if storage driver is `lvms`) -- Create volume group from block device if needed
3. **Local-path setup** (if storage driver is `local-path`) -- Create the host storage directory
4. **Apply manifests** in order:
   - `assets/rbac/` -- Default RBAC policies
   - `assets/kindnet/` -- Kindnet CNI DaemonSet
   - `assets/coredns/` -- CoreDNS Deployment and Service
   - `assets/storage/<driver>/` -- Storage provisioner (based on configured driver)

Manifests are Go templates. Variables like `{{.ClusterCIDR}}`, `{{.ServiceCIDR}}`, `{{.DNSAddress}}`, and `{{.NodeIP}}` are expanded from the configuration before applying.

## Health Monitoring

After bootstrap, the daemon enters a monitoring loop:

- **Watchdog:** Every 30 seconds, sends `WATCHDOG=1` to systemd. If the daemon hangs, systemd's `WatchdogSec=300` triggers a restart
- **API server health:** Checks `https://127.0.0.1:6443/healthz`
- **etcd health:** Checks `https://127.0.0.1:2379/health`
- **Node readiness:** Uses kubectl to verify the node reports `Ready`

## Data Directory Layout

```
/var/lib/microshift/
  certs/                 TLS certificates and keys
    ca.crt, ca.key
    apiserver.crt, apiserver.key
    front-proxy-ca.crt, front-proxy-ca.key
    sa.pub, sa.key
    ...
  kubeconfig/            Kubeconfig files for each component
    admin.kubeconfig
    controller-manager.kubeconfig
    scheduler.kubeconfig
    kubelet.kubeconfig
  config/                Component configuration and environment files
    etcd.yaml, etcd.env
    apiserver.env
    controller-manager.env
    scheduler.env
    kubelet-config.yaml, kubelet.env
  etcd/                  etcd data directory
    member/
  storage/               Local-path provisioner PV directory (if using local-path)
```

## Go Package Structure

| Package | Responsibility |
|---------|---------------|
| `pkg/daemon` | Main orchestration loop, signal handling, sd_notify |
| `pkg/config` | Configuration loading, validation, defaults |
| `pkg/certs` | TLS certificate and key generation |
| `pkg/kubeconfig` | Kubeconfig file generation |
| `pkg/services` | Component config generation, systemd D-Bus service management |
| `pkg/bootstrap` | Manifest templating and application via kubectl |
| `pkg/healthcheck` | HTTP and kubectl-based health checks |
| `pkg/version` | Build version info (injected via ldflags) |
