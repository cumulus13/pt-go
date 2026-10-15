#!/usr/bin/env bash
# Build the NSIS installer for Windows (amd64).
# Usage: packaging/build-windows-installer.sh <version> <path-to-pt.exe> [outdir]
# Run from the repository root. Requires `makensis` (apt-get install nsis).
set -euo pipefail

VERSION="${1:?version required, e.g. 1.0.81}"
EXE="${2:?path to pt.exe required}"
OUT="${3:-dist}"

VERSION="${VERSION#v}"
[ -f "$EXE" ] || { echo "binary not found: $EXE" >&2; exit 2; }
[ -f packaging/windows/pt.nsi ] || { echo "run from the repo root" >&2; exit 2; }
command -v makensis >/dev/null || { echo "makensis not found (apt-get install nsis)" >&2; exit 2; }

mkdir -p "$OUT"
OUT="$(cd "$OUT" && pwd)"   # absolute: makensis resolves relative OutFile against the script dir
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

# VERSION file installed next to pt.exe (pt reads it from its own directory)
printf 'version = "%s"\n' "$VERSION" > "$WORK/VERSION"
cp pt.yml.example README.md pt.ico packaging/windows/path.ps1 "$WORK/"
cp "$EXE" "$WORK/pt-bin.exe"

OUTFILE="$OUT/pt-${VERSION}-windows-amd64-setup.exe"
DEFS=(-DVERSION="$VERSION" -DBIN="$WORK/pt-bin.exe" -DREPO="$WORK" -DOUTFILE="$OUTFILE")

# VIProductVersion needs strictly N.N.N.N
if [[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  DEFS+=(-DVI_VERSION="${VERSION}.0")
fi

makensis -V2 "${DEFS[@]}" packaging/windows/pt.nsi
ls -l "$OUTFILE"
