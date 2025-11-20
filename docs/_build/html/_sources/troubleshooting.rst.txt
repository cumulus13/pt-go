Troubleshooting
===============

This page lists common issues and solutions.

Clipboard Empty Error
---------------------

**Error**::

   ⚠️  Warning: Clipboard is empty

**Solution**: Copy some text to clipboard before running PT.

macOS::

   echo "Hello World" | pbcopy

Linux::

   echo "Hello World" | xclip -selection clipboard
   # or
   echo "Hello World" | xsel --clipboard --input

Windows::

   echo Hello World | clip

No Write Permission
-------------------

**Error**::

   ❌ Error: no write permission in directory

**Solution**: Check directory permissions or use a different location.

Check current directory permissions::

   ls -la  # Linux/macOS
   icacls .  # Windows

Fix permissions::

   chmod +w .  # Linux/macOS

File Too Large
--------------

**Error**::

   ❌ Error: clipboard content too large (max 100MB)

**Solution**: Content exceeds safety limit. Options:

1. Increase ``max_clipboard_size`` in ``pt.yml`` (up to 1GB)
2. Save content directly to file without using clipboard

File Not Found
--------------

**Error**::

   ❌ Error: file not found: config.json

**Causes**:

- File doesn't exist
- File is deeper than 10 directory levels
- File is in a ``.pt`` subdirectory (automatically excluded)
- Typo in filename

**Solutions**:

- Use absolute path::
   
     pt /full/path/to/config.json

- Increase ``max_search_depth`` in ``pt.yml``::

     max_search_depth: 20

- Check filename spelling
- Ensure file is not in ``.pt`` directory

Delta Not Found
---------------

**Error**::

   ❌ Error: delta is not installed. Install it from: https://github.com/dandavison/delta

**Solution**: Install delta for diff functionality.

macOS::

   brew install git-delta

Ubuntu/Debian::

   sudo apt install git-delta

Arch Linux::

   sudo pacman -S git-delta

Fedora/RHEL::

   sudo dnf install git-delta

Windows (Chocolatey)::

   choco install git-delta

Windows (Scoop)::

   scoop install delta

Config File Issues
------------------

**Warning**::

   ⚠️  Warning: invalid max_clipboard_size, using default

**Solution**: Check config file syntax and value ranges::

   pt config show

**Valid ranges**:

- ``max_clipboard_size``: 1 - 1073741824 (1GB)
- ``max_backup_count``: 1 - 10000
- ``max_filename_length``: 1 - 1000
- ``max_search_depth``: 1 - 100

Content Unchanged (Check Mode)
-------------------------------

**Message**::

   ℹ️  Content identical to current file, no changes needed

This is **normal behavior**. Check mode (``-c``) prevents unnecessary writes when content hasn't changed.

To force write anyway, omit the ``-c`` flag.

Recursive Search Issues
-----------------------

**Error**::

   🔍 Searching for 'file.txt' recursively...
   ❌ Error: file not found: file.txt

**Possible causes**:

1. File is deeper than 10 directory levels (default)
2. File is in a ``.pt`` subdirectory (automatically excluded)
3. Permission issues reading some directories
4. Typo in filename

**Solutions**:

- Use absolute path::
   
     pt /full/path/to/file.txt

- Increase search depth in ``pt.yml``::

     max_search_depth: 20

- Check file permissions::

     ls -la /path/to/file.txt

- Verify filename spelling

Multiple File Selection
-----------------------

When multiple files are found, PT shows an interactive prompt::

   🔍 Found 3 matching file(s)

   ┌──────┬────────────────────────────┬─────────────────────┬──────────────┐
   │   #  │ Path                       │ Modified            │ Size         │
   ├──────┼────────────────────────────┼─────────────────────┼──────────────┤
   │    1 │ ./README.md                │ 2025-11-15 10:30:00 │ 15.2 KB      │
   │    2 │ ./docs/README.md           │ 2025-11-14 15:20:00 │ 8.5 KB       │
   └──────┴────────────────────────────┴─────────────────────┴──────────────┘

   Enter file number to use (1-3) or 0 to cancel: 

**Solution**: Select the file number you want to work with, or press 0 to cancel.

Linux Clipboard Issues
----------------------

**Error**::

   ❌ Error: failed to read clipboard

**Solution**: Install clipboard utilities.

Ubuntu/Debian::

   sudo apt-get install xclip xsel

Fedora/RHEL::

   sudo dnf install xclip xsel

Arch::

   sudo pacman -S xclip xsel

After installation, test::

   echo "test" | xclip -selection clipboard
   pt test.txt

.pt Directory Not Found
-----------------------

**Message**::

   📁 Created .pt directory: /path/to/.pt

This is **informational**, not an error.

PT creates a ``.pt`` directory (Git-like) to store backups. It's automatically:

- Created in the file's directory or parent
- Added to ``.gitignore``
- Searched upward like ``.git``

Backup Directory Confusion
--------------------------

**Issue**: Backups not found where expected

**Explanation**: PT uses a Git-like ``.pt`` directory structure:

.. code-block:: text

   project/
   ├── .pt/                          # Git-like backup directory
   │   ├── main_go/
   │   │   └── main_go.20251115_...
   │   └── config_json/
   │       └── config_json.20251115_...
   ├── main.go
   └── config.json

The ``.pt`` directory is **shared across files** like ``.git``.

To find where backups are stored::

   pt -l myfile.txt

The header shows the backup location.

Windows-Specific Issues
-----------------------

**Path separator issues**: PT handles both ``/`` and ``\`` automatically.

**Clipboard access**: May require PowerShell or WSL on some Windows versions.

Getting Help
------------

If you encounter issues not listed here:

1. Check the full documentation: :doc:`usage`, :doc:`configuration`
2. Run with ``--debug`` for detailed logs::
   
     pt --debug myfile.txt

3. File an issue on GitHub: https://github.com/cumulus13/pt-go/issues