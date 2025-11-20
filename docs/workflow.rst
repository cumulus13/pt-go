Git-Like Workflow
=================

PT introduces a powerful git-like workflow for file version management, making it easy to track and manage changes across your project.

Workflow Overview
-----------------

The workflow mirrors Git's commands:

1. **Check Status** - See what files changed
2. **Commit Changes** - Backup all modified files
3. **View History** - List backups with comments
4. **Compare Changes** - Diff with previous versions
5. **Rollback** - Restore previous versions

Step-by-Step Workflow
---------------------

1. Check File Status
~~~~~~~~~~~~~~~~~~~~

See the status of all files in your project::

   pt check

Output::

   📊 PT Status (like git status)

   myproject/
   ├── src/
   │   ├── main.go (15.2 KB) [modified]
   │   └── utils.go (3.4 KB)
   ├── config.json (2.1 KB) [new]
   └── old_config.json (1.8 KB) [deleted]

   Summary:
     1 modified
     1 new
     1 deleted

   Use 'pt commit -m "message"' to backup all changes

Status colors:

-  **Green**  = Unchanged (matches last backup)
-  **Yellow**  = Modified (content changed)
-  **Cyan**  = New (no backup exists yet)
-  **Red**  = Deleted (backup exists but file gone)

2. Commit All Changes
~~~~~~~~~~~~~~~~~~~~~

Backup all modified and new files with a single command::

   pt commit -m "Fixed authentication bug and updated config"

Output::

   📦 Committing changes...

   Files to backup:
     1. src/main.go [modified]
     2. config.json [new]

   Commit 2 file(s) with message "Fixed authentication bug and updated config"? (y/N): y
   ✓ src/main.go
   ✓ config.json

   📦 Commit Summary:
     ✓ 2 files backed up
     💬 Message: "Fixed authentication bug and updated config"

**Commit Behavior:**

- Only backs up **modified** and **new** files
- Skips **unchanged** files (no backup needed)
- All backups tagged with ``commit: message``
- Confirmation prompt before backing up

3. View Commit History
~~~~~~~~~~~~~~~~~~~~~~

See the history of a specific file::

   pt -l src/main.go

Output::

   📂 Backup files for 'src/main.go'
   Total: 5 backup(s) (stored in .pt/)

   ┌──────────────────────────┬─────────────────────┬──────────┬────────────────────────────────┐
   │ File Name                │ Modified            │     Size │ Comment                        │
   ├──────────────────────────┼─────────────────────┼──────────┼────────────────────────────────┤
   │ 1. main_go.20251118...   │ 2025-11-18 14:12:41 │  50.5 KB │ commit: Fixed authentication   │
   │ 2. main_go.20251118...   │ 2025-11-18 14:11:24 │  57.0 KB │ Working version before refactor│
   │ 3. main_go.20251118...   │ 2025-11-18 13:43:01 │  52.6 KB │ commit: Initial implementation │
   └──────────────────────────┴─────────────────────┴──────────┴────────────────────────────────┘

4. Compare Changes
~~~~~~~~~~~~~~~~~~

See what changed in the latest commit::

   pt -d src/main.go --last

This shows a beautiful side-by-side diff using delta.

5. Rollback if Needed
~~~~~~~~~~~~~~~~~~~~~

Restore the previous version::

   pt -r src/main.go --last -m "Rolling back due to test failure"

Output::

   ✅ Successfully restored: src/main.go
   📦 From backup: main_go.20251118_141124...
   📄 Content size: 57046 characters
   💬 Restore comment: "Rolling back due to test failure"

Workflow Use Cases
------------------

Daily Development
~~~~~~~~~~~~~~~~~

.. code-block:: bash

   # Morning: Check what changed yesterday
   pt check

   # Commit all work with descriptive message
   pt commit -m "Day's work: feature X and bugfixes"

   # Continue working...

   # Before risky changes: backup specific file
   pt main.go -m "Before refactoring auth module"

   # After changes: review diff
   pt -d main.go --last

   # If tests fail: rollback
   pt -r main.go --last -m "Reverting due to test failures"

Configuration Management
~~~~~~~~~~~~~~~~~~~~~~~~

.. code-block:: bash

   # Update config with context
   pt config.json -m "Production DB settings for v2.1"

   # Later update for testing
   pt config.json -m "Testing new cache timeout"

   # View all config versions
   pt -l config.json

   # Compare configs
   pt -d config.json --last

   # Restore production config
   pt -r config.json --last -m "Reverting to production"

Emergency Rollback
~~~~~~~~~~~~~~~~~~

.. code-block:: bash

   # Check what files were recently modified
   pt check

   # View history of problematic file
   pt -l script.sh

   # Quick rollback
   pt -r script.sh --last -m "Emergency rollback"

Workflow Advantages
-------------------

1. **Zero Setup** - No repository initialization needed
2. **File-Level** - Works on individual files, not entire directories
3. **Lightweight** - No .git directory bloat, only .pt with actual backups
4. **Flexible** - Mix file-level backups with project-wide commits
5. **Contextual** - Every change has a comment explaining why
6. **Safe** - Automatic backups before destructive operations

Comparison with Git
-------------------

+---------------------+---------------------+---------------------+
| Feature             | Git                 | PT                  |
+=====================+=====================+=====================+
| Setup               | `git init` required | None needed         |
+---------------------+---------------------+---------------------+
| Scope               | Entire repository   | Per-file + project  |
+---------------------+---------------------+---------------------+
| Storage             | .git directory      | .pt directory       |
+---------------------+---------------------+---------------------+
| Comments            | Commit messages     | Per-backup comments |
+---------------------+---------------------+---------------------+
| Learning Curve      | Steep               | Very simple         |
+---------------------+---------------------+---------------------+
| Best For            | Code projects       | Files, configs,     |
|                     |                     | snippets, notes     |
+---------------------+---------------------+---------------------+

PT complements Git - use Git for code collaboration and PT for quick local version control of files, configurations, and snippets.