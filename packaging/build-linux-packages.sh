#!/usr/bin/env bash
# Build .deb and .apk packages with nfpm.
# Usage: packaging/build-linux-packages.sh <version> <amd64|arm64> <path-to-static-linux-binary> [outdir]
# Run from the repository root. Requires `nfpm` in PATH.
set -euo pipefail

VERSION="${1:?version required, e.g. 1.0.81}"
ARCH="${2:?arch required: amd64|arm64}"
BIN="${3:?binary path required}"
OUT="${4:-dist}"

VERSION="${VERSION#v}"
case "$ARCH" in amd64|arm64) ;; *) echo "unsupported arch: $ARCH" >&2; exit 2 ;; esac
[ -f "$BIN" ] || { echo "binary not found: $BIN" >&2; exit 2; }
[ -f pt.yml.example ] && [ -f README.md ] || { echo "run from the repo root" >&2; exit 2; }

mkdir -p "$OUT"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

# VERSION file shipped next to the binary (pt reads it from its own directory)
printf 'version = "%s"\n' "$VERSION" > "$WORK/VERSION"

export PT_VERSION="$VERSION" PT_ARCH="$ARCH" PT_BIN="$BIN" PT_VERSION_FILE="$WORK/VERSION"

for fmt in deb apk; do
  nfpm package --config packaging/nfpm.yaml --packager "$fmt" --target "$OUT/"
done

ls -l "$OUT"
