#!/bin/bash
# Build MicroShift disk images using bootc-image-builder (osbuild).
#
# Prerequisites: podman, root/sudo
#
# Usage:
#   sudo ./scripts/build-disk-images.sh --format qcow2
#   sudo ./scripts/build-disk-images.sh --format all --ssh-key ~/.ssh/id_ed25519.pub

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
CONFIG_TOML="$PROJECT_DIR/images/diskimage/config.toml"
CONTAINERFILE="$PROJECT_DIR/images/bootc/Containerfile"
OUTPUT_DIR="$PROJECT_DIR/_output/images"
BIB_IMAGE="quay.io/centos-bootc/bootc-image-builder:latest"
BOOTC_IMAGE="localhost/microshift-bootc:latest"
ROOTFS="xfs"

FORMAT="qcow2"
SSH_KEY=""
PASSWORD=""

usage() {
    cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Build MicroShift disk images from the bootc container image.

Options:
  --format FORMAT   Image format: qcow2, raw, iso, or all (default: qcow2)
  --ssh-key PATH    Path to SSH public key to inject into the image
  --password HASH   Hashed password for the microshift user (mkpasswd --method=sha-512)
  --output DIR      Output directory (default: _output/images)
  --rootfs FS       Root filesystem type: xfs or ext4 (default: xfs)
  -h, --help        Show this help

Examples:
  sudo ./scripts/build-disk-images.sh --format qcow2 --ssh-key ~/.ssh/id_ed25519.pub
  sudo ./scripts/build-disk-images.sh --format all
  sudo ./scripts/build-disk-images.sh --format raw --rootfs ext4
EOF
    exit 0
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --format)   FORMAT="$2"; shift 2 ;;
        --ssh-key)  SSH_KEY="$2"; shift 2 ;;
        --password) PASSWORD="$2"; shift 2 ;;
        --output)   OUTPUT_DIR="$2"; shift 2 ;;
        --rootfs)   ROOTFS="$2"; shift 2 ;;
        -h|--help)  usage ;;
        *)          echo "Unknown option: $1" >&2; exit 1 ;;
    esac
done

if [[ $EUID -ne 0 ]]; then
    echo "Error: this script must be run as root (sudo)" >&2
    exit 1
fi

# Build the bootc container image if not already present
if ! podman image exists "$BOOTC_IMAGE" 2>/dev/null; then
    echo "==> Building bootc container image..."
    make -C "$PROJECT_DIR" build
    podman build -t "$BOOTC_IMAGE" -f "$CONTAINERFILE" "$PROJECT_DIR"
else
    echo "==> Using existing bootc image: $BOOTC_IMAGE"
fi

# Prepare config.toml with optional SSH key and password
WORK_DIR="$(mktemp -d)"
trap 'rm -rf "$WORK_DIR"' EXIT
cp "$CONFIG_TOML" "$WORK_DIR/config.toml"

if [[ -n "$SSH_KEY" ]]; then
    if [[ ! -f "$SSH_KEY" ]]; then
        echo "Error: SSH key file not found: $SSH_KEY" >&2
        exit 1
    fi
    KEY_CONTENT="$(cat "$SSH_KEY")"
    # Append key to the microshift user block
    sed -i "/^\[\[customizations\.user\]\]/,/^$/{/^groups/a key = \"$KEY_CONTENT\"
}" "$WORK_DIR/config.toml"
fi

if [[ -n "$PASSWORD" ]]; then
    sed -i "/^\[\[customizations\.user\]\]/,/^$/{/^groups/a password = \"$PASSWORD\"
}" "$WORK_DIR/config.toml"
fi

# Determine --type flags
declare -a TYPE_ARGS=()
case "$FORMAT" in
    qcow2)  TYPE_ARGS=(--type qcow2) ;;
    raw)    TYPE_ARGS=(--type raw) ;;
    iso)    TYPE_ARGS=(--type anaconda-iso) ;;
    all)    TYPE_ARGS=(--type qcow2 --type raw --type anaconda-iso) ;;
    *)      echo "Error: unknown format '$FORMAT'. Use: qcow2, raw, iso, all" >&2; exit 1 ;;
esac

mkdir -p "$OUTPUT_DIR"

echo "==> Building disk image(s): $FORMAT"
echo "    Output: $OUTPUT_DIR"

podman run \
    --rm --privileged \
    --security-opt label=type:unconfined_t \
    -v /var/lib/containers/storage:/var/lib/containers/storage \
    -v "$WORK_DIR/config.toml":/config.toml:ro \
    -v "$OUTPUT_DIR":/output \
    "$BIB_IMAGE" \
    "${TYPE_ARGS[@]}" \
    --rootfs "$ROOTFS" \
    --local \
    "$BOOTC_IMAGE"

echo ""
echo "==> Build complete. Output:"
find "$OUTPUT_DIR" -type f -name "*.qcow2" -o -name "*.raw" -o -name "*.iso" 2>/dev/null | while read -r f; do
    echo "    $(du -h "$f" | cut -f1)  $f"
done
