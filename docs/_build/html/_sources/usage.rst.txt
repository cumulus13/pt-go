Usage Reference
===============

Basic Commands
--------------

Write clipboard to file (creates backup if exists)::

   pt myfile.txt

Write with comment::

   pt myfile.txt -m "Fixed bug in authentication logic"

Write with check mode (skip if unchanged)::

   pt myfile.txt -c

Combine check mode with comment::

   pt myfile.txt -c -m "Updated configuration"

Append clipboard to file (no backup)::

   pt + myfile.txt

Append with comment::

   pt + myfile.txt -m "Added new log entry"

List all backups with sizes, timestamps, and comments::

   pt -l myfile.txt

Restore backup (interactive selection)::

   pt -r myfile.txt

Restore with comment::

   pt -r myfile.txt -m "Rolled back to stable version"

Restore last backup directly::

   pt -r myfile.txt --last

Restore last backup with comment::

   pt -r myfile.txt --last -m "Emergency rollback"

Show help::

   pt --help

Show version::

   pt --version

Git-Like Workflow Commands
--------------------------

Check file status (like ``git status``)::

   pt check

Check single file status::

   pt check myfile.txt

Commit all changes (backup all modified files)::

   pt commit -m "Your commit message"

Advanced Commands
-----------------

Recursive file search::

   pt config.json              # Searches up to 10 levels deep
   pt -l utils.go              # List backups (searches recursively)
   pt -r main.py               # Restore (searches recursively)

Diff operations::

   pt -d myfile.txt            # Interactive: choose which backup to compare
   pt -d myfile.txt --last     # Quick: compare with most recent backup
   pt --diff script.py         # Alternative syntax
   pt -d <file> -z             # Diff clipboard with file (no backup)

Directory tree::

   pt -t                       # Show tree of current directory
   pt -t /path/to/dir          # Show tree of specific directory
   pt -t -e node_modules,.git  # Tree with exceptions (exclude folders)

Safe delete::

   pt -rm old_file.txt         # Backup, delete, create empty placeholder
   pt -rm old_file.txt -m "Deprecated old implementation"

Configuration::

   pt config init              # Creates pt.yml in current directory
   pt config init ~/.pt.yml    # Create in custom location
   pt config show              # Show current configuration
   pt config path              # Show config file location

View clipboard content::

   pt -z                       # Show clipboard in less
   pt -z --lexer python        # Show with syntax highlighting

Command Reference Table
-----------------------

.. list-table:: PT Commands
   :header-rows: 1
   :widths: 25 50 25

   * - Command
     - Description
     - Example
   * - ``pt <file>``
     - Save clipboard to file
     - ``pt notes.txt``
   * - ``pt <file> -m "msg"``
     - Save with comment
     - ``pt code.go -m "Fix bug"``
   * - ``pt <file> -c``
     - Save only if changed
     - ``pt data.json -c``
   * - ``pt + <file>``
     - Append to file
     - ``pt + log.txt``
   * - ``pt -l <file>``
     - List backups
     - ``pt -l script.py``
   * - ``pt -r <file>``
     - Restore backup
     - ``pt -r config.json``
   * - ``pt -d <file>``
     - Diff with backup
     - ``pt -d main.go --last``
   * - ``pt -t``
     - Show directory tree
     - ``pt -t -e node_modules``
   * - ``pt -rm <file>``
     - Safe delete file
     - ``pt -rm old.py``
   * - ``pt check``
     - Show all file statuses
     - ``pt check``
   * - ``pt commit -m "msg"``
     - Backup all changes
     - ``pt commit -m "Updates"``
   * - ``pt config show``
     - Show configuration
     - ``pt config show``
   * - ``pt -z``
     - View clipboard
     - ``pt -z --lexer yaml``