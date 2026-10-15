#!/bin/sh
# POSIX sh only (Alpine uses busybox ash).
set -e
echo "pt installed in /TOOLS/pt (added to PATH via /etc/profile.d/pt.sh)."
echo "Open a new login shell, or run:  . /etc/profile.d/pt.sh"
exit 0
