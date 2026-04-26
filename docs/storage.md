# Storage Guide

MicroShift supports persistent storage for workloads through pluggable storage drivers. Each driver runs as pods inside the cluster, not as host-level services.

## Choosing a Driver

| Driver | Best For | Prerequisites | Performance |
|--------|----------|---------------|-------------|
| `local-path` | Development, single-app workloads | None | Local disk speed |
| `lvms` | Production edge, dedicated disks | `lvm2` package, block device or existing VG | Local disk speed with thin provisioning |
| `nfs` | Shared storage, multi-node access | `nfs-utils` package, NFS server | Network-dependent |
| `none` | Stateless workloads, BYO storage | None | N/A |

Configure the driver in `/etc/microshift/config.yaml` under the `storage` section. The default is `local-path`.

## local-path-provisioner (default)

Creates PersistentVolumes as directories on the host filesystem. Simple, requires no additional packages, and works out of the box.

### How it works

When a PVC is created, the provisioner creates a directory under the configured storage path and binds it to the pod. When the PVC is deleted, the directory is removed.

### Configuration

```yaml
storage:
  driver: local-path
  localPath:
    storagePath: "/var/lib/microshift/storage"
```

The `storagePath` directory is created automatically on startup.

### StorageClass

- **Name:** `local-path`
- **Reclaim policy:** Delete
- **Volume binding:** WaitForFirstConsumer
- **Marked as default:** Yes

### Limitations

- Data is local to the node -- not replicated or backed up
- No thin provisioning or capacity enforcement
- Directory-based, not block-based
- Not suitable for databases that require strict I/O guarantees

## TopoLVM / LVMS

Uses LVM thin provisioning via the TopoLVM CSI driver. Provides block-level volumes with capacity tracking, making it suitable for production workloads on edge devices with dedicated storage.

### Prerequisites

Install the `lvm2` package:

```bash
sudo dnf install lvm2
```

You need either an existing LVM volume group or a block device that MicroShift can use to create one.

### Configuration with existing volume group

If you already have a VG:

```bash
sudo vgcreate microshift-vg /dev/sdb
```

```yaml
storage:
  driver: lvms
  lvms:
    volumeGroup: microshift-vg
```

### Configuration with automatic VG creation

Provide a block device and MicroShift will create the VG on first start:

```yaml
storage:
  driver: lvms
  lvms:
    volumeGroup: microshift-vg
    device: /dev/sdb
```

MicroShift runs `pvcreate` and `vgcreate` automatically if the VG does not exist. The device must be unused and unpartitioned.

### StorageClass

- **Name:** `topolvm-provisioner`
- **Reclaim policy:** Delete
- **Volume binding:** WaitForFirstConsumer
- **Marked as default:** Yes

### How it works

TopoLVM runs two components:

1. **topolvm-controller** (Deployment) -- CSI controller that handles provisioning and resizing
2. **topolvm-node** (DaemonSet) -- CSI node plugin that creates and manages logical volumes on the host

Both run in the `topolvm-system` namespace.

## NFS

Uses the NFS subdir external provisioner to create PersistentVolumes backed by subdirectories on an NFS share. Useful when you have existing NFS infrastructure.

### Prerequisites

Install the `nfs-utils` package:

```bash
sudo dnf install nfs-utils
```

You need an accessible NFS server with an exported path.

### Configuration

```yaml
storage:
  driver: nfs
  nfs:
    server: "192.168.1.10"
    path: "/exports/microshift"
```

Both `server` and `path` are required.

### StorageClass

- **Name:** `nfs-client`
- **Reclaim policy:** Delete
- **Volume binding:** Immediate
- **Marked as default:** Yes

### How it works

The provisioner mounts the NFS share and creates a subdirectory for each PVC. The subdirectory is named `${namespace}-${pvcName}-${pvName}`. When a PVC is deleted, the subdirectory is removed.

### Limitations

- Performance depends on network and NFS server
- NFS server must be reachable from the node
- File locking behavior differs from local storage
- Not suitable for workloads requiring block storage

## Disabling Storage

Set the driver to `none` if you don't need a storage provisioner (for example, if you're providing your own CSI driver or running stateless workloads):

```yaml
storage:
  driver: none
```

## Verifying Storage

After starting MicroShift, verify that the storage class is available:

```bash
kubectl get storageclass
```

You should see a default storage class matching your configured driver.

### Test with a PVC

Create a test PVC to verify provisioning works:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: test-pvc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
EOF
```

For `WaitForFirstConsumer` storage classes, the PV is created only when a pod mounts the PVC. Create a test pod:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: test-storage
spec:
  containers:
    - name: test
      image: busybox
      command: ["sleep", "3600"]
      volumeMounts:
        - mountPath: /data
          name: test-vol
  volumes:
    - name: test-vol
      persistentVolumeClaim:
        claimName: test-pvc
EOF
```

Verify the PVC is bound:

```bash
kubectl get pvc test-pvc
```

Clean up:

```bash
kubectl delete pod test-storage
kubectl delete pvc test-pvc
```
