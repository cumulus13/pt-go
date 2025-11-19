.. _usage:

Usage
=====

Basic Operation
---------------

The primary function of ``pt`` is to write clipboard content to a file. The general syntax is:

.. code-block:: bash

   pt [options] <filename>

Examples:

* ``pt notes.txt``: Writes clipboard content to ``notes.txt``.
* ``pt report.md -m "Initial draft"``: Writes content to ``report.md`` with a backup comment.
* ``pt + log.txt``: Appends clipboard content to ``log.txt``.

Git-like Workflow
-----------------

``pt`` also supports a Git-like workflow for managing changes to multiple files:

1. **Check Status**: ``pt check`` - Lists files showing their status (modified, new, unchanged, deleted).
2. **Commit Changes**: ``pt commit -m "message"`` - Backs up all modified and new files found by ``pt check``.
3. **List Backups**: ``pt -l <filename>`` - Shows all available backups for a specific file.
4. **Restore Backup**: ``pt -r <filename>`` - Interactively restores a previous backup.
5. **Diff**: ``pt -d <filename>`` - Compares the current file with a backup using ``delta``.

.. note::
   The ``.pt`` directory stores all backups. It searches parent directories like Git does.

.. _command_reference:

Command Reference
-----------------

The ``pt`` command supports various subcommands and options, as detailed in the help message below:

.. code-block:: text

   ╔══════════════════════════════════════════════════════════╗
   ║          PT - Clipboard to File Tool v1.0.23             ║
   ║                                                          ║
   ║                     by cumulus13                         ║
   ╚══════════════════════════════════════════════════════════╝
   📝 BASIC OPERATIONS:
     pt <filename>               Write clipboard to file
     pt <filename> -c            Write only if content differs
     pt <filename> -m "msg"      Write with comment
     pt + <filename>             Append clipboard to file

   🎯 GIT-LIKE WORKFLOW (NEW!):
     pt check                    Show status of all files (like git status)
     pt check <filename>         Check single file status
     pt commit -m "message"     Backup all changed files (like git commit)

   📦 BACKUP OPERATIONS:
     pt -l <filename>            List all backups (with comments)
     pt -r <filename>            Restore backup (interactive)
     pt -r <filename> --last     Restore most recent backup

   📊 DIFF OPERATIONS:
     pt -d <filename>            Compare with backup (interactive)
     pt -d <filename> --last     Compare with most recent backup

   🌳 TREE & UTILITIES:
     pt -t [path]                Show directory tree
     pt -t [path] -e items,items       Tree with exceptions
     pt -rm <filename>           Safe delete (backup first)

   ⚙️  CONFIGURATION:
     pt config init              Create sample config file
     pt config show              Show current configuration
     pt config path              Show config file location

   ℹ️  INFORMATION:
     pt -h, --help               Show this help message
     pt -v, --version            Show version information

   💡 EXAMPLES:
     $ pt notes.txt                # Save clipboard
     $ pt check                    # Show all file statuses
     $ pt commit -m "fix bugs"    # Backup all changes
     $ pt -l notes.txt             # List backups
     $ pt -d notes.txt --last      # Diff with last backup

   🎯 GIT-LIKE WORKFLOW:
     1. pt check                  - See what files changed (like git status)
     2. pt commit -m "msg"        - Backup all changes (like git commit)
     3. pt -l <file>              - View commit history
     4. pt -d <file> --last       - See what changed
     5. pt -r <file> --last       - Rollback if needed

   📊 CHECK/STATUS OUTPUT:
     • Green   = Unchanged (matches last backup)
     • Yellow  = Modified (content changed)
     • Cyan    = New (no backup exists yet)
     • Red     = Deleted (backup exists but file gone)

   📦 COMMIT BEHAVIOR:
     • Only backs up modified and new files
     • Skips unchanged files (no backup needed)
     • All backups tagged with "commit: message"
     • Confirmation prompt before backing up

   🔍 RECURSIVE SEARCH:
     • If file not in current directory, searches recursively
     • Maximum search depth: 10 levels
     • If multiple files found, prompts for selection
     • Respects .ptignore and .gitignore patterns

   📂 .pt DIRECTORY (Git-like structure):
     • Location: .pt/ directory (like .git)
     • Searches parent directories for existing .pt/
     • If found in parent, uses that (like git)
     • If not found, creates .pt/ in current directory
     • Automatically added to .gitignore
     • Backups organized by file path inside .pt/

   📄 IGNORE FILES:
     • .ptignore: PT-specific ignore patterns (higher priority)
     • .gitignore: Also respected for recursive search
     • Format: One pattern per line, # for comments
     • .pt/ directory always excluded from search

   ⚙️  SYSTEM LIMITS:
     • Max file size: 100MB
     • Max filename: 200 characters
     • Max backups: 100 per file
     • Search depth: 10 levels

   🔧 REQUIREMENTS:
     • delta: Required for diff operations
       Install: https://github.com/dandavison/delta
       - macOS:     brew install git-delta
       - Linux:     cargo install git-delta
       - Windows:   scoop install delta

   🛡️  SECURITY FEATURES:
     • Path traversal protection (blocks '..' in paths)
     • System directory protection (blocks /etc, /sys, etc.)
     • Write permission validation
     • File size validation
     • Atomic-like backup operations

   📋 NOTES:
     • All operations are logged to stderr for audit trail
     • Backup timestamps use microsecond precision
     • Files are synced to disk after writing
     • Supports cross-platform operation (Linux, macOS, Windows)
     • .pt/ directory works like .git/ - searches upward

   📄 LICENSE: MIT | AUTHOR: Hadi Cahyadi <cumulus13@gmail.com>

   🔍 RECURSIVE SEARCH:
     • If file not in current directory, searches recursively
     • Maximum search depth: 10 levels
     • If multiple files found, prompts for selection
     • Respects .ptignore and .gitignore patterns

   📂 .pt DIRECTORY (Git-like structure):
     • Location: .pt/ directory (like .git)
     • Searches parent directories for existing .pt/
     • If found in parent, uses that (like git)
     • If not found, creates .pt/ in current directory
     • Automatically added to .gitignore
     • Backups organized by file path inside .pt/

   📄 IGNORE FILES:
     • .ptignore: PT-specific ignore patterns (higher priority)
     • .gitignore: Also respected for recursive search
     • Format: One pattern per line, # for comments
     • .pt/ directory always excluded from search

   ⚙️  SYSTEM LIMITS:
     • Max file size: 100MB
     • Max filename: 200 characters
     • Max backups: 100 per file
     • Search depth: 10 levels

   🔧 REQUIREMENTS:
     • delta: Required for diff operations
       Install: https://github.com/dandavison/delta
       - macOS:     brew install git-delta
       - Linux:     cargo install git-delta
       - Windows:   scoop install delta

   🛡️  SECURITY FEATURES:
     • Path traversal protection (blocks '..' in paths)
     • System directory protection (blocks /etc, /sys, etc.)
     • Write permission validation
     • File size validation
     • Atomic-like backup operations

   📋 NOTES:
     • All operations are logged to stderr for audit trail
     • Backup timestamps use microsecond precision
     • Files are synced to disk after writing
     • Supports cross-platform operation (Linux, macOS, Windows)
     • .pt/ directory works like .git/ - searches upward

   📄 LICENSE: MIT | AUTHOR: Hadi Cahyadi <cumulus13@gmail.com>