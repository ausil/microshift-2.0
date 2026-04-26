# Configuration Reference

MicroShift reads its configuration from `/etc/microshift/config.yaml`. All fields are optional; if omitted, the default value is used.

## Fields

### clusterName

The name of the Kubernetes cluster.

- **Type:** string
- **Default:** `microshift`

### baseDomain

The base DNS domain for the cluster.

- **Type:** string
- **Default:** `microshift.local`

### nodeIP

The IP address to use for the node. If left empty, MicroShift auto-detects the default route interface IP.

- **Type:** string
- **Default:** `""` (auto-detected)

### serviceCIDR

The CIDR range used for Kubernetes ClusterIP services.

- **Type:** string
- **Default:** `10.96.0.0/16`

### clusterCIDR

The CIDR range used for pod networking.

- **Type:** string
- **Default:** `10.42.0.0/16`

### dataDir

The directory where MicroShift stores its state, including certificates, kubeconfig files, component configuration, and etcd data.

- **Type:** string
- **Default:** `/var/lib/microshift`

### logLevel

The logging verbosity level.

- **Type:** string
- **Allowed values:** `debug`, `info`, `warn`, `error`
- **Default:** `info`

### cni

The CNI plugin to use for pod networking.

- **Type:** string
- **Allowed values:** `kindnet`, `ovn-kubernetes`
- **Default:** `kindnet`

### etcdMemoryLimit

Maximum memory for the etcd process in megabytes. Set to 0 to disable the limit.

- **Type:** integer
- **Default:** `0` (no limit)

### storage

Configures persistent storage for workloads.

#### storage.driver

The storage provisioner to deploy.

- **Type:** string
- **Allowed values:** `local-path`, `lvms`, `nfs`, `none`
- **Default:** `local-path`

#### storage.localPath.storagePath

Host directory where local-path-provisioner creates persistent volumes.

- **Type:** string
- **Default:** `/var/lib/microshift/storage`
- **Used when:** `storage.driver: local-path`

#### storage.lvms.volumeGroup

Name of the LVM volume group for TopoLVM.

- **Type:** string
- **Required when:** `storage.driver: lvms`

#### storage.lvms.device

Block device to auto-create the volume group from. If empty, the VG must already exist.

- **Type:** string
- **Default:** `""` (VG must pre-exist)
- **Used when:** `storage.driver: lvms`

#### storage.nfs.server

NFS server hostname or IP address.

- **Type:** string
- **Required when:** `storage.driver: nfs`

#### storage.nfs.path

NFS export path on the server.

- **Type:** string
- **Required when:** `storage.driver: nfs`

## Examples

### Basic (defaults)

```yaml
clusterName: my-edge-cluster
nodeIP: "192.168.1.100"
```

### With LVMS storage

```yaml
clusterName: production-edge
nodeIP: "10.0.0.5"
storage:
  driver: lvms
  lvms:
    volumeGroup: microshift-vg
    device: /dev/sdb
```

### With NFS storage

```yaml
clusterName: shared-storage-edge
nodeIP: "192.168.1.50"
storage:
  driver: nfs
  nfs:
    server: "192.168.1.10"
    path: "/exports/microshift"
```

### Full configuration

```yaml
clusterName: my-edge-cluster
baseDomain: edge.example.com
nodeIP: "192.168.1.100"
serviceCIDR: "10.96.0.0/16"
clusterCIDR: "10.42.0.0/16"
dataDir: "/var/lib/microshift"
logLevel: "debug"
cni: "kindnet"
etcdMemoryLimit: 512
storage:
  driver: local-path
  localPath:
    storagePath: "/var/lib/microshift/storage"
```
