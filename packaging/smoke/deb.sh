#!/bin/sh
# Smoke test, run INSIDE a debian container: install the .deb, verify PATH, remove.
# Expects: /dist/pt-go_*_amd64.deb and env EXPECT_VERSION.
set -eu

DEB="$(ls /dist/pt-go_*_amd64.deb)"
apt-get update -qq
apt-get install -y -qq "$DEB"

test -x /TOOLS/pt/pt
test -f /TOOLS/pt/VERSION
test -f /etc/profile.d/pt.sh

# A fresh LOGIN shell must see /TOOLS/pt on PATH
[ "$(bash -lc 'command -v pt')" = "/TOOLS/pt/pt" ]
[ "$(sh -lc 'command -v pt')" = "/TOOLS/pt/pt" ]
bash -lc 'echo "$PATH"' | tr ':' '\n' | grep -qx '/TOOLS/pt'
# no duplicate entries
[ "$(bash -lc 'echo "$PATH"' | tr ':' '\n' | grep -cx '/TOOLS/pt')" = "1" ]

OUT="$(bash -lc 'cd / && pt --version' | head -1)"
echo "$OUT"
[ "$OUT" = "PT version ${EXPECT_VERSION}" ]

apt-get remove -y -qq pt-go
test ! -e /TOOLS/pt
test ! -e /etc/profile.d/pt.sh
echo "DEB SMOKE TEST OK"
