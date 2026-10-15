#!/bin/sh
# Smoke test, run INSIDE an alpine container: install the .apk, verify PATH, remove.
# Expects: /dist/pt-go_*_x86_64.apk and env EXPECT_VERSION.
set -eu

APK="$(ls /dist/pt-go_*_x86_64.apk)"
# unsigned package -> --allow-untrusted
apk add --no-cache --allow-untrusted "$APK"

test -x /TOOLS/pt/pt
test -f /TOOLS/pt/VERSION
test -f /etc/profile.d/pt.sh

[ "$(sh -lc 'command -v pt')" = "/TOOLS/pt/pt" ]
sh -lc 'echo "$PATH"' | tr ':' '\n' | grep -qx '/TOOLS/pt'
[ "$(sh -lc 'echo "$PATH"' | tr ':' '\n' | grep -cx '/TOOLS/pt')" = "1" ]

OUT="$(sh -lc 'cd / && pt --version' | head -1)"
echo "$OUT"
[ "$OUT" = "PT version ${EXPECT_VERSION}" ]

apk del pt-go
test ! -e /TOOLS/pt
test ! -e /etc/profile.d/pt.sh
echo "APK SMOKE TEST OK"
