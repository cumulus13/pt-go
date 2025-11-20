Installation
============

Prerequisites
-------------

- Go 1.16 or higher
- Git (for installation)
- **Delta** (optional, for diff functionality) - `Install from here <https://github.com/dandavison/delta>`_

Install Delta
-------------

macOS::

   brew install git-delta

Ubuntu/Debian::

   sudo apt install git-delta

Arch Linux::

   sudo pacman -S git-delta

Fedora/RHEL::

   sudo dnf install git-delta

Windows (with Chocolatey)::

   choco install delta

Windows (with Scoop)::

   scoop install delta

Install PT from Source
----------------------

Using ``go install``::

   go install github.com/cumulus13/pt-go/pt@latest

Clone and build manually::

   git clone https://github.com/cumulus13/pt-go.git
   cd pt-go
   go build -o pt pt/main.go
   sudo mv pt /usr/local/bin/  # Linux/macOS

Quick Install (Linux/macOS)
---------------------------

One-liner installation::

   curl -sSL https://raw.githubusercontent.com/cumulus13/pt-go/pt/main/install.sh | bash

Verify Installation
-------------------

Check version::

   pt --version

Output::

   PT version 1.0.32
   Production-hardened clipboard to file tool
   Features: Git-like .pt structure, recursive search, backup management, delta diff

Troubleshooting
---------------

**Clipboard Empty Error**

Solution: Copy some text to clipboard before running PT::

   echo "Hello World" | pbcopy  # macOS
   echo "Hello World" | xclip -selection clipboard  # Linux

**Delta Not Found**

Error::

   ❌ Error: delta is not installed. Install it from: https://github.com/dandavison/delta

Solution: Install delta using one of the commands above.

**No Write Permission**

Error::

   ❌ Error: no write permission in directory

Solution: Check directory permissions or use a different location.