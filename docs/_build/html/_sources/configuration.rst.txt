Configuration
=============

PT supports configuration via a ``pt.yml`` file with sensible defaults.

Configuration File Locations
----------------------------

PT searches for configuration in this order:

1. ``./pt.yml`` or ``./pt.yaml`` (current directory)
2. ``~/.config/pt/pt.yml`` or ``~/.config/pt/pt.yaml``
3. ``~/pt.yml`` or ``~/pt.yaml`` (home directory)

Creating Configuration File
---------------------------

Initialize a sample config::

   pt config init

This creates ``pt.yml`` in the current directory with default values.

Or create manually::

   cat > pt.yml << EOF
   # PT Configuration File
   max_clipboard_size: 104857600    # 100MB
   max_backup_count: 100            # Keep 100 backups
   max_filename_length: 200         # Max filename length
   backup_dir_name: .pt             # Git-like hidden directory
   max_search_depth: 10             # Max recursive search depth
   EOF

Configuration Options
---------------------

.. list-table:: Configuration Options
   :header-rows: 1
   :widths: 25 15 10 50

   * - Setting
     - Default
     - Range
     - Description
   * - ``max_clipboard_size``
     - 104857600 (100MB)
     - 1 - 1073741824 (1GB)
     - Maximum clipboard content size in bytes
   * - ``max_backup_count``
     - 100
     - 1 - 10000
     - Maximum number of backups kept per file
   * - ``max_filename_length``
     - 200
     - 1 - 1000
     - Maximum filename length in characters
   * - ``backup_dir_name``
     - .pt
     - -
     - Name of the backup directory (Git-like structure)
   * - ``max_search_depth``
     - 10
     - 1 - 100
     - Maximum recursive search depth in levels

Viewing Configuration
---------------------

Show current configuration::

   pt config show

Output::

   Current PT Configuration:

   Max Clipboard Size: 104857600 bytes (100.0 MB)
   Max Backup Count: 100
   Max Filename Length: 200 characters
   Backup Directory: .pt/ (Git-like structure)
   Max Search Depth: 10 levels

   Config loaded from: ./pt.yml

Show config file location::

   pt config path

Configuration Validation
------------------------

PT validates configuration values on startup. Invalid values trigger warnings and fall back to defaults::

   Warning: invalid max_clipboard_size, using default

Backup Directory Structure (.pt/)
-----------------------------------

The ``.pt`` directory works like ``.git``:

.. code-block:: text

   project/
   ├── .pt/                          # Git-like backup directory
   │   ├── main_go/                  # Backups for main.go
   │   │   ├── main_go.20251115_151804.12345_a1b2c3d4
   │   │   └── main_go.20251115_151804.12345_a1b2c3d4.meta.json
   │   └── src_lib_util_go/          # Backups for src/lib/util.go
   │       └── util_go.20251115_120000.67890_abc12345
   ├── main.go                       # Current version
   └── src/lib/util.go               # Current version

Backup Naming Format
--------------------

Backups use this zero-collision format::

   filename_ext.YYYYMMDD_HHMMSS_MICROSECONDS.PID_RANDOMID

Example::

   main_go.20251115_151804177132.12345_a1b2c3d4
   main_go.20251115_151804177132.12345_a1b2c3d4.meta.json

Components:

- ``main_go`` - Original filename without extension
- ``20251115_151804177132`` - Timestamp with microsecond precision
- ``12345`` - Process ID
- ``a1b2c3d4`` - Random 8-character hex ID
- ``.meta.json`` - Metadata file with comment