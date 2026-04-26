# Troubleshooting

## Checking Service Status

MicroShift runs as a set of systemd services. Check the status of any component:

```bash
sudo systemctl status microshift.service
sudo systemctl status microshift-etcd.service
sudo systemctl status microshift-apiserver.service
sudo systemctl status microshift-controller-manager.service
sudo systemctl status microshift-scheduler.service
sudo systemctl status microshift-kubelet.service
```

List all MicroShift services at once:

```bash
sudo systemctl list-units 'microshift*'
```

## Viewing Logs

All MicroShift components log through journald:

```bash
# Main daemon logs
sudo journalctl -u microshift.service

# Specific component logs
sudo journalctl -u microshift-etcd.service
sudo journalctl -u microshift-apiserver.service
sudo journalctl -u microshift-kubelet.service

# Follow logs in real time
sudo journalctl -u microshift.service -f

# Logs since last boot
sudo journalctl -u microshift.service -b

# All MicroShift-related logs
sudo journalctl -u 'microshift*' --no-pager
```

## Common Issues

### Services won't start

**Symptom:** `systemctl start microshift` fails or services remain inactive.

**Check dependencies:**

```bash
rpm -q kubernetes1.35 etcd cri-o1.35 containernetworking-plugins
```

If any are missing, install them:

```bash
sudo dnf install kubernetes1.35 etcd cri-o1.35 containernetworking-plugins
```

**Check that CRI-O is running:**

```bash
sudo systemctl status crio
sudo systemctl enable --now crio
```

**Check for port conflicts:**

```bash
sudo ss -tlnp | grep -E '6443|2379|2380|10250'
```

If another process is using these ports, stop it or reconfigure MicroShift.

### API server unreachable

**Symptom:** `kubectl` commands fail with connection refused or TLS errors.

**Check API server status:**

```bash
sudo systemctl status microshift-apiserver.service
sudo journalctl -u microshift-apiserver.service --no-pager -n 50
```

**Check the API server is listening:**

```bash
curl -k https://127.0.0.1:6443/healthz
```

Expected output: `ok`

**Check certificate files exist:**

```bash
ls -la /var/lib/microshift/certs/
```

You should see `ca.crt`, `ca.key`, `apiserver.crt`, `apiserver.key`, and other cert files. If they are missing, the main MicroShift daemon may have failed during startup.

**Verify kubeconfig points to the right server:**

```bash
cat /var/lib/microshift/kubeconfig/admin.kubeconfig | grep server
```

### etcd failures

**Symptom:** etcd won't start or crashes repeatedly.

**Check disk space:**

```bash
df -h /var/lib/microshift/etcd
```

etcd requires adequate free disk space. If the filesystem is full, clean up or expand it.

**Check data directory permissions:**

```bash
ls -la /var/lib/microshift/etcd/
```

The etcd data directory must be writable by the etcd user.

**Check etcd logs:**

```bash
sudo journalctl -u microshift-etcd.service --no-pager -n 100
```

### Pods stuck in Pending

**Symptom:** Pods remain in `Pending` state and never start.

**Check node readiness:**

```bash
kubectl get nodes
```

If the node shows `NotReady`, check kubelet logs:

```bash
sudo journalctl -u microshift-kubelet.service --no-pager -n 50
```

**Check CNI plugin:**

```bash
kubectl get pods -n kube-system -l app=kindnet
```

If kindnet pods are not running, the CNI is not ready and pods cannot get network configuration.

**Check storage provisioning (if PVCs are involved):**

```bash
kubectl get pvc
kubectl describe pvc <name>
```

A PVC stuck in `Pending` means the storage provisioner cannot create a volume. See "Storage provisioning failures" below.

### Certificate errors

**Symptom:** TLS handshake failures, "certificate has expired", or "certificate signed by unknown authority" errors.

**Check certificate expiry:**

```bash
openssl x509 -in /var/lib/microshift/certs/apiserver.crt -noout -dates
```

**Check system clock:**

```bash
timedatectl
```

Certificate validation requires an accurate system clock. If the clock is significantly wrong, TLS will fail. Sync the clock:

```bash
sudo systemctl enable --now chronyd
```

**Regenerate certificates:**

Stop MicroShift, remove the cert directory, and restart:

```bash
sudo systemctl stop microshift
sudo rm -rf /var/lib/microshift/certs
sudo systemctl start microshift
```

MicroShift regenerates all certificates on startup if they don't exist.

### Storage provisioning failures

**LVMS -- volume group not found:**

```bash
sudo vgs
```

If the expected VG is missing and `storage.lvms.device` is configured, check the device exists:

```bash
lsblk
```

Check MicroShift logs for VG creation errors:

```bash
sudo journalctl -u microshift.service | grep -i vg
```

**NFS -- mount failures:**

Verify the NFS server is reachable:

```bash
showmount -e <nfs-server-ip>
```

Check that `nfs-utils` is installed:

```bash
rpm -q nfs-utils
```

Test a manual mount:

```bash
sudo mount -t nfs <server>:<path> /mnt
```

### CRI-O container runtime issues

**Symptom:** Pods fail to start with image pull or container creation errors.

**Check CRI-O status:**

```bash
sudo systemctl status crio
sudo journalctl -u crio --no-pager -n 50
```

**List containers:**

```bash
sudo crictl ps -a
sudo crictl pods
```

**Check images:**

```bash
sudo crictl images
```

## Resetting the Cluster

To completely reset MicroShift and start fresh, stop all services and remove the data directory:

```bash
sudo systemctl stop microshift
sudo rm -rf /var/lib/microshift
sudo systemctl start microshift
```

This removes all certificates, kubeconfig files, etcd data, and component configuration. MicroShift regenerates everything on the next start.

**Warning:** This deletes all cluster state, including deployed workloads, persistent volumes, and secrets. This cannot be undone.

## Debug Commands Reference

| Command | Purpose |
|---------|---------|
| `systemctl status microshift*` | Check all service states |
| `journalctl -u microshift.service -f` | Follow main daemon logs |
| `kubectl get nodes` | Check node readiness |
| `kubectl get pods -A` | List all pods across namespaces |
| `kubectl describe pod <name>` | Detailed pod status and events |
| `kubectl logs <pod>` | View pod container logs |
| `sudo crictl ps -a` | List CRI-O containers |
| `sudo crictl logs <container-id>` | View container logs via CRI |
| `curl -k https://127.0.0.1:6443/healthz` | Test API server health |
| `curl -k https://127.0.0.1:2379/health` | Test etcd health |
| `openssl x509 -in <cert> -noout -text` | Inspect a certificate |
