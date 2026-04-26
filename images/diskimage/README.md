# MicroShift Disk Images

Build bootable disk images for VMs and physical devices using [bootc-image-builder](https://github.com/osbuild/bootc-image-builder) (osbuild).

## Prerequisites

- `podman` installed
- Root access (sudo)
- Go toolchain (to build the MicroShift binary)

## Quick Start

```bash
# Build a qcow2 VM image
sudo make disk-image-qcow2

# Build all formats (qcow2, raw, ISO)
sudo make disk-image-all
```

## Image Formats

| Format | Output | Use Case |
|--------|--------|----------|
| qcow2 | `_output/images/qcow2/disk.qcow2` | VMs (libvirt, QEMU, virt-manager) |
| raw | `_output/images/raw/disk.raw` | Write to disk/SD card with `dd` |
| iso | `_output/images/bootiso/install.iso` | Bare metal install via USB/PXE |

## Building with Options

```bash
# Inject an SSH public key
sudo ./scripts/build-disk-images.sh --format qcow2 --ssh-key ~/.ssh/id_ed25519.pub

# Set a password (generate hash with: mkpasswd --method=sha-512)
sudo ./scripts/build-disk-images.sh --format qcow2 --password '$6$rounds=4096$salt$hash...'

# Use ext4 instead of xfs
sudo ./scripts/build-disk-images.sh --format qcow2 --rootfs ext4

# Build all formats at once
sudo ./scripts/build-disk-images.sh --format all --ssh-key ~/.ssh/id_ed25519.pub
```

## Using the Images

### qcow2 with libvirt

```bash
sudo virt-install \
    --name microshift \
    --memory 4096 --vcpus 2 \
    --import \
    --disk path=_output/images/qcow2/disk.qcow2,format=qcow2 \
    --os-variant fedora44 \
    --network network=default \
    --noautoconsole
```

Or with QEMU directly:

```bash
qemu-system-x86_64 -m 4096 -smp 2 -enable-kvm \
    -drive file=_output/images/qcow2/disk.qcow2,format=qcow2 \
    -nic user,hostfwd=tcp::6443-:6443,hostfwd=tcp::2222-:22 \
    -nographic
```

### Raw image to disk

```bash
# Write to an SD card or USB drive
sudo dd if=_output/images/raw/disk.raw of=/dev/sdX bs=4M status=progress conv=fsync
```

### ISO bare metal install

Write the ISO to a USB drive and boot from it:

```bash
sudo dd if=_output/images/bootiso/install.iso of=/dev/sdX bs=4M status=progress conv=fsync
```

## Default Configuration

- **Hostname:** `microshift`
- **User:** `microshift` (in `wheel` group)
- **Enabled services:** sshd, crio, microshift
- **Firewall ports:** 6443/tcp (API), 22/tcp (SSH), 10250/tcp (kubelet), 2379/tcp (etcd)
- **Root filesystem:** XFS, 10 GiB minimum
- **Data partition:** `/var/lib/microshift`, XFS, 5 GiB minimum

## Customization

Edit `images/diskimage/config.toml` to change the blueprint. See the [bootc-image-builder blueprint reference](https://osbuild.org/docs/user-guide/blueprint-reference/) for all options.

For ISO installs, the kickstart is in `images/diskimage/kickstart.ks`.

## Accessing the Cluster

After the VM boots:

```bash
# SSH into the node
ssh microshift@<ip-address>

# Copy the kubeconfig
scp microshift@<ip-address>:/var/lib/microshift/kubeconfig/admin.kubeconfig ~/.kube/config

# Verify
kubectl get nodes
```
