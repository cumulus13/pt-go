Quick Start Guide
=================

In this guide, you'll learn the basics of PT in just 5 minutes.

Installation
------------

Install PT using Go::

   go install github.com/cumulus13/pt-go/pt@latest

Or use the quick install script (Linux/macOS)::

   curl -sSL https://raw.githubusercontent.com/cumulus13/pt-go/pt/main/install.sh | bash

Verify installation::

   pt --version

Basic Usage
-----------

1. **Save Clipboard to File**

   Copy some text, then save it to a file::

      pt myfile.txt -m "My first note"

2. **View Backups**

   See all versions of your file::

      pt -l myfile.txt

3. **Compare Changes**

   See what changed between versions::

      pt -d myfile.txt --last

4. **Restore a Version**

   Restore the previous version::

      pt -r myfile.txt --last -m "Rolling back"

5. **Git-Like Workflow**

   Check status of all files::

      pt check

   Commit all changes::

      pt commit -m "Save all my work"

Configuration
-------------

Initialize a config file::

   pt config init

Edit ``pt.yml`` to customize settings::

   # PT Configuration File
   max_clipboard_size: 104857600    # 100MB
   max_backup_count: 100            # Keep 100 backups
   max_filename_length: 200         # Max filename length
   backup_dir_name: .pt             # .pt directory (Git-like)
   max_search_depth: 10             # Max recursive search depth

View current configuration::

   pt config show

Next Steps
----------

- Learn about :doc:`installation` options
- Explore detailed :doc:`usage` examples
- Understand the :doc:`workflow` for version control
- Read about :doc:`configuration` options