# Networking Guide

MicroShift provides pod networking via CNI plugins and cluster DNS via CoreDNS. Both run as pods inside the cluster, deployed automatically during bootstrap.

## CNI Plugins

The CNI plugin handles pod-to-pod networking. MicroShift supports two options, configured via the `cni` field in `/etc/microshift/config.yaml`.

### Kindnet (default)

Kindnet is a simple, lightweight CNI plugin that provides pod networking using VXLAN overlay tunnels. It is the default and requires no configuration.

```yaml
cni: kindnet
```

Kindnet runs as a DaemonSet in the `kube-system` namespace. It:

- Assigns pod IPs from the `clusterCIDR` range (default: `10.42.0.0/16`)
- Creates VXLAN tunnels between nodes (relevant for future multi-node support)
- Configures IP masquerading for outbound traffic
- Requires the `containernetworking-plugins` package for bridge and host-local CNI binaries

Kindnet is the right choice for most edge deployments. It has minimal resource overhead and requires no external dependencies beyond the base CNI plugins.

### OVN-Kubernetes (optional)

OVN-Kubernetes provides advanced networking features using Open Virtual Network (OVN). It is intended for deployments that need network policies, egress control, or more sophisticated traffic management.

```yaml
cni: ovn-kubernetes
```

OVN-Kubernetes requires additional packages and configuration. Support for OVN-Kubernetes is planned for a future release.

## CoreDNS

CoreDNS provides cluster-internal DNS, enabling service discovery via DNS names. It runs as a Deployment in the `kube-system` namespace and is deployed automatically during bootstrap.

### How DNS works

Every pod in the cluster is configured to use the CoreDNS service IP for DNS resolution. CoreDNS resolves:

- **Service names:** `my-service.my-namespace.svc.cluster.local` resolves to the ClusterIP
- **Pod names:** `pod-ip.my-namespace.pod.cluster.local`
- **Headless services:** Returns individual pod IPs instead of a ClusterIP
- **External names:** Forwards to upstream DNS servers for non-cluster domains

The cluster DNS domain is `cluster.local`.

### DNS Service IP

The DNS service IP is computed as the 10th IP in the `serviceCIDR` range. With the default `serviceCIDR` of `10.96.0.0/16`, the DNS IP is `10.96.0.10`.

| serviceCIDR | DNS IP |
|-------------|--------|
| `10.96.0.0/16` | `10.96.0.10` |
| `10.100.0.0/16` | `10.100.0.10` |
| `172.30.0.0/24` | `172.30.0.10` |

## Network CIDRs

MicroShift uses two CIDR ranges for networking:

### serviceCIDR

The IP range for Kubernetes ClusterIP services. Default: `10.96.0.0/16`.

Services created with `type: ClusterIP` receive an IP from this range. The Kubernetes API server itself listens at the first IP (`10.96.0.1`), and the DNS service uses the 10th IP (`10.96.0.10`).

```yaml
serviceCIDR: "10.96.0.0/16"
```

### clusterCIDR

The IP range for pod networking. Default: `10.42.0.0/16`.

Every pod gets an IP from this range. The CNI plugin allocates and manages these addresses.

```yaml
clusterCIDR: "10.42.0.0/16"
```

### Guidelines for changing CIDRs

- Choose ranges that do not overlap with your host network
- Choose ranges that do not overlap with each other
- Ensure `serviceCIDR` has enough addresses for your services (a `/16` provides ~65,000)
- Ensure `clusterCIDR` has enough addresses for your pods
- CIDRs can only be set before first start -- changing them on a running cluster requires a full reset

## Firewall Configuration

If your system runs a firewall, the following ports must be open for MicroShift to function:

| Port | Protocol | Component | Purpose |
|------|----------|-----------|---------|
| 6443 | TCP | kube-apiserver | Kubernetes API |
| 2379-2380 | TCP | etcd | Client and peer communication |
| 10250 | TCP | kubelet | Kubelet API |
| 10257 | TCP | kube-controller-manager | Health endpoint |
| 10259 | TCP | kube-scheduler | Health endpoint |

On Fedora with `firewalld`:

```bash
sudo firewall-cmd --permanent --add-port=6443/tcp
sudo firewall-cmd --permanent --add-port=2379-2380/tcp
sudo firewall-cmd --permanent --add-port=10250/tcp
sudo firewall-cmd --permanent --add-port=10257/tcp
sudo firewall-cmd --permanent --add-port=10259/tcp
sudo firewall-cmd --reload
```

Since MicroShift is single-node, all these ports only need to be accessible locally. If you are accessing the API server remotely (e.g., from a workstation), port 6443 must be open to that host.

### VXLAN port

If Kindnet is the CNI (default), VXLAN traffic uses UDP port 4789. This only matters in multi-node scenarios.

## Verifying Networking

Check that the CNI plugin pods are running:

```bash
kubectl get pods -n kube-system -l app=kindnet
```

Check that CoreDNS is running:

```bash
kubectl get pods -n kube-system -l k8s-app=kube-dns
```

Test DNS resolution from inside the cluster:

```bash
kubectl run dns-test --image=busybox --rm -it --restart=Never -- nslookup kubernetes.default
```

This should resolve to the first IP in your `serviceCIDR` (default: `10.96.0.1`).
