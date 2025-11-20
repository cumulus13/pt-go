Features
========

Core Features
-------------

📝 Quick Save
~~~~~~~~~~~~~

Write clipboard content to file with one command::

   pt myfile.txt

Automatic backup is created if file exists.

📦 Auto Backup
~~~~~~~~~~~~~~

Automatic timestamped backups stored in `.pt/` directory (Git-like structure). Each backup includes:

- Microsecond-precision timestamp
- Process ID
- Random ID for zero-collision guarantee
- Optional comment

Backup format::

   filename_ext.YYYYMMDD_HHMMSS_MICROSECONDS.PID_RANDOMID

💬 Backup Comments
~~~~~~~~~~~~~~~~~~

Add descriptive comments to track why changes were made::

   pt main.go -m "Fixed authentication bug"

Comments are stored in `.meta.json` files alongside backups.

➕ Append Mode
~~~~~~~~~~~~~~

Add content without creating backups::

   pt + myfile.txt

Useful for logs or accumulating content.

🔄 Restore
~~~~~~~~~~

Interactive or quick restore from backups::

   pt -r myfile.txt          # Interactive selection
   pt -r myfile.txt --last   # Restore most recent

Restore with comment::

   pt -r myfile.txt --last -m "Emergency rollback"

📊 Beautiful Listings
~~~~~~~~~~~~~~~~~~~~~

Formatted table view of all backups with sizes, timestamps, and comments::

   pt -l myfile.txt

Output::

   ┌──────────────────────────┬─────────────────────┬──────────┬────────────────────────────────┐
   │ File Name                │ Modified            │     Size │ Comment                        │
   ├──────────────────────────┼─────────────────────┼──────────┼────────────────────────────────┤
   │ 1. main_go.20251118...   │ 2025-11-18 14:12:41 │  50.5 KB │ Fixed auth bug                 │
   │ 2. main_go.20251118...   │ 2025-11-18 14:11:24 │  57.0 KB │ Working version before refactor│
   └──────────────────────────┴─────────────────────┴──────────┴────────────────────────────────┘

🔒 Production Hardened
~~~~~~~~~~~~~~~~~~~~~~~

Path validation:

- Prevents path traversal attacks (``../../../etc/passwd``)
- Blocks writes to system directories (``/etc``, ``/sys``, ``C:\Windows``)
- Validates filename length limits
- Sanitizes all file paths

Size limits:

- Maximum 100MB clipboard content (configurable)
- Prevents disk exhaustion attacks
- Validates disk space before writing

Safe operations:

- Atomic-like file operations
- Verification of write completion
- Automatic rollback on errors
- Backup before destructive operations

🎨 Colorful Output
~~~~~~~~~~~~~~~~~~

ANSI colors for better readability:

-  **Green**  : Success, unchanged files
-  **Yellow**  : Warnings, modified files
-  **Cyan**  : Info, new files
-  **Red**  : Errors, deleted files
-  **Blue**  : Search operations

📈 Audit Logging
~~~~~~~~~~~~~~~~

All operations logged to stderr for tracking::

   pt myfile.txt -m "Test"

Log output::

   2025-11-18 14:12:41 Successfully written to: myfile.txt (1234 bytes)

Use ``--debug`` for verbose logging::

   pt --debug myfile.txt

✅ Check Mode
~~~~~~~~~~~~~

Skip writes if content unchanged (saves disk space)::

   pt data.json -c

Output if identical::

   ℹ️  Content identical to current file, no changes needed
   📄 File: data.json

Output if different::

   🔍 Content differs, proceeding with backup and write

Advanced Features
-----------------

🔍 Recursive File Search
~~~~~~~~~~~~~~~~~~~~~~~~~

Automatically finds files in subdirectories up to 10 levels deep::

   pt config.json              # Finds it anywhere in project

If multiple files found, interactive selection::

   🔍 Found 3 matching file(s)

   ┌──────┬────────────────────────────┬─────────────────────┬──────────────┐
   │   #  │ Path                       │ Modified            │ Size         │
   ├──────┼────────────────────────────┼─────────────────────┼──────────────┤
   │    1 │ ./README.md                │ 2025-11-15 10:30:00 │ 15.2 KB      │
   │    2 │ ./docs/README.md           │ 2025-11-14 15:20:00 │ 8.5 KB       │
   └──────┴────────────────────────────┴─────────────────────┴──────────────┘

   Enter file number to use (1-3) or 0 to cancel: 1

Search respects:

- ``.gitignore`` patterns
- ``.ptignore`` patterns (PT-specific, higher priority)
- Excludes ``.pt/`` directory automatically

📊 Delta Diff Integration
~~~~~~~~~~~~~~~~~~~~~~~~~~

Beautiful side-by-side diff comparison with backups::

   pt -d myfile.txt            # Interactive: choose backup
   pt -d myfile.txt --last     # Quick: compare with last

Requires `Delta <https://github.com/dandavison/delta>`_ to be installed.

Diff clipboard with file (no backup)::

   pt -d myfile.txt -z

🌳 Directory Tree View
~~~~~~~~~~~~~~~~~~~~~~

Visual file structure with sizes (like ``tree`` command)::

   pt -t                       # Current directory
   pt -t /path/to/dir          # Specific directory

Output::

   myproject/
   ├── src/
   │   ├── main.go (15.2 KB)
   │   └── utils.go (3.4 KB)
   ├── .pt/
   │   └── ... (backup files)
   ├── README.md (2.1 KB)
   └── go.mod (456 B)

   2 directories, 5 files, 29.2 KB total

Tree with exceptions::

   pt -t -e node_modules,.git,dist

📁 GitIgnore Support
~~~~~~~~~~~~~~~~~~~~

Respects ``.gitignore`` patterns in tree view and recursive search.

🗑️ Safe Delete
~~~~~~~~~~~~~~~

Backup before deletion, create empty placeholder::

   pt -rm old_file.txt

Output::

   📦 Backup created: old_file_txt.20251118_141241...
   🗑️  File deleted: old_file.txt
   📄 Created empty placeholder: old_file.txt

Use with comment::

   pt -rm legacy_auth.py -m "Replaced by new OAuth2 implementation"

⚙️ Exception Filtering
~~~~~~~~~~~~~~~~~~~~~~

Exclude specific files/folders from tree view::

   pt -t /path -e build,dist,node_modules

🎯 Multi-File Selection
~~~~~~~~~~~~~~~~~~~~~~~

Interactive prompt when multiple files found during recursive search.

🚀 Smart Path Resolution
~~~~~~~~~~~~~~~~~~~~~~~~

Finds files anywhere in your project, automatically searching upward for ``.pt`` directory.

⚙️ Configurable
~~~~~~~~~~~~~~~~

Customize behavior via ``pt.yml`` config file::

   max_clipboard_size: 104857600
   max_backup_count: 100
   max_filename_length: 200
   backup_dir_name: .pt
   max_search_depth: 10

Version Management Capabilities
-------------------------------

PT acts as a lightweight version control system with descriptive comments:

📜 Complete Version History
~~~~~~~~~~~~~~~~~~~~~~~~~~~

Every file change is preserved with optional comments.

💬 Comment System
~~~~~~~~~~~~~~~~~

Track why changes were made, not just when::

   pt config.json -m "Production config for v2.1.0 release"

📝 Contextual Notes
~~~~~~~~~~~~~~~~~~~

Add meaningful descriptions to each backup::

   pt main.go -m "Fixed race condition in goroutine"

🔙 Easy Rollback
~~~~~~~~~~~~~~~~

Restore any previous version instantly, see why it was saved::

   pt -r main.go --last

📊 Version Comparison
~~~~~~~~~~~~~~~~~~~~~

Diff any two versions visually with delta::

   pt -d main.go --last

💾 Space Efficient
~~~~~~~~~~~~~~~~~~

Only changed files are backed up (check mode).

🏷️ Timestamped Versions
~~~~~~~~~~~~~~~~~~~~~~~

Microsecond precision timestamps + human-readable comments.

Zero Data Loss
--------------

Never lose work:

- Automatic backup before every write
- Backup before deletion
- Atomic-like operations
- Verification of write completion

Performance
-----------

.. list-table:: Performance Metrics
   :header-rows: 1
   :widths: 30 20 50

   * - Operation
     - Performance
     - Notes
   * - **Startup Time**
     - < 50ms
     - Very fast startup
   * - **Write Speed**
     - Disk I/O limited
     - Depends on storage
   * - **Memory Usage**
     - ~5MB + content
     - Minimal footprint
   * - **Backup Creation**
     - < 100ms
     - For typical files
   * - **Search Speed**
     - < 200ms
     - For 1000+ files
   * - **Multi-file Search**
     - < 500ms
     - Up to 10 levels deep
   * - **Tree Generation**
     - < 500ms
     - 5000+ files/folders
   * - **Diff Rendering**
     - Depends on delta
     - Powered by delta