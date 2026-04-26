# Getting Started

## Prerequisites

MicroShift 2.0 requires a Fedora system (Fedora IoT recommended) with the following packages available from the Fedora repositories:

- `kubernetes1.32` -- provides kube-apiserver, kube-controller-manager, kube-scheduler, and kubelet
- `etcd` -- cluster state store
- `cri-o` -- container runtime
- `containernetworking-plugins` -- CNI plugins (bridge, host-local)

Install the dependencies:

```bash
sudo dnf install kubernetes1.32 etcd cri-o containernetworking-plugins
```

## Installation

### From RPM (recommended)

```bash
sudo dnf install microshift
```

### From source

```bash
git clone https://github.com/ausil/microshift-2.0.git
cd microshift-2.0
make build
sudo make install
```

## Starting the cluster

Enable and start MicroShift. This will automatically start all dependent services (etcd, API server, controller manager, scheduler, kubelet, kube-proxy) via the systemd dependency chain:

```bash
sudo systemctl enable --now microshift.service
```

You can check the status of all MicroShift services:

```bash
sudo systemctl status microshift.service
sudo systemctl status microshift-etcd.service
sudo systemctl status microshift-apiserver.service
```

## Verifying with kubectl

MicroShift writes a kubeconfig file that you can use with kubectl:

```bash
export KUBECONFIG=/var/lib/microshift/resources/kubeadmin/kubeconfig
kubectl get nodes
kubectl get pods -A
```

You should see your node in `Ready` state and system pods (kindnet, CoreDNS) running in the `kube-system` namespace.

## Configuration

The default configuration file is at `/etc/microshift/config.yaml`. All fields are optional; MicroShift uses sensible defaults. See the [Configuration Reference](configuration.md) for details on each field.

## Stopping the cluster

Stopping MicroShift stops all component services:

```bash
sudo systemctl stop microshift.service
```
