Examples
========

Here are comprehensive examples showing PT's capabilities.

1. Quick Note Taking
--------------------

.. code-block:: bash

   # Copy meeting notes to clipboard, then:
   pt notes.txt -m "Meeting notes from sprint planning"

Output::

   ✅ Successfully written to: notes.txt
   📄 Content size: 142 characters
   💬 Comment: "Meeting notes from sprint planning"

2. Version Control for Code
---------------------------

.. code-block:: bash

   # Before making risky changes
   pt main.go -m "Working version before refactoring"

   # 📦 Backup created: main_go.20251118_141241...
   # 💬 Comment: "Working version before refactoring"

   # After changes (only saves if different)
   pt main.go -c -m "Refactored authentication module"

   # 🔍 Content differs, proceeding with backup and write
   # ✅ Successfully written to: main.go

   # View version history with comments
   pt -l main.go

3. Configuration Management
---------------------------

.. code-block:: bash

   # Save production config
   pt config.json -m "Production config for v2.1.0 release"

   # Later, update for testing
   pt config.json -m "Testing new cache settings"

   # View all config versions with comments
   pt -l config.json

Output::

   ┌────────────┬─────────────┬──────┬────────────────────────────┐
   │ File Name  │ Modified    │ Size │ Comment                    │
   ├────────────┼─────────────┼──────┼────────────────────────────┤
   │ 1. config..│ 14:12:41    │ 2 KB │ Testing new cache settings │
   │ 2. config..│ 10:30:15    │ 2 KB │ Production config v2.1.0   │
   └────────────┴─────────────┴──────┴────────────────────────────┘

   # Restore production config
   pt -r config.json --last -m "Reverting to production config"

4. Check Mode - Save Disk Space
-------------------------------

.. code-block:: bash

   # Only write if content actually changed
   pt data.json -c

   # ℹ️  Content identical to current file, no changes needed
   # 📄 File: data.json

   # Or with comment if it does change
   pt data.json -c -m "Updated user preferences"

   # 🔍 Content differs, proceeding with backup and write
   # 📦 Backup created with comment

5. Safe Delete with Context
---------------------------

.. code-block:: bash

   # Delete old implementation with explanation
   pt -rm legacy_auth.py -m "Replaced by new OAuth2 implementation"

Output::

   📦 Backup created: legacy_auth_py.20251118_141241...
   💬 Comment: "Replaced by new OAuth2 implementation"
   🗑️  File deleted: legacy_auth.py
   📄 Created empty placeholder: legacy_auth.py

6. Append Mode with Comments
-----------------------------

.. code-block:: bash

   # Append log entries with context
   pt + errors.log -m "Error logs from production incident"

Output::

   ✅ Successfully appended to: errors.log
   📝 Content size: 87 characters
   💬 Comment: "Error logs from production incident"

7. Interactive Restore
----------------------

.. code-block:: bash

   pt -r main.go

Output::

   📂 Backup files for 'main.go'
   Total: 5 backup(s) (stored in .pt/)

   ┌──────────────────────────┬─────────────────────┬──────────┬────────────────────────────────┐
   │ File Name                │ Modified            │     Size │ Comment                        │
   ├──────────────────────────┼─────────────────────┼──────────┼────────────────────────────────┤
   │ 1. main_go.20251118...   │ 2025-11-18 14:12:41 │  50.5 KB │ Add comment system             │
   │ 2. main_go.20251118...   │ 2025-11-18 14:11:24 │  57.0 KB │ Working version before refactor│
   │ 3. main_go.20251118...   │ 2025-11-18 13:43:01 │  52.6 KB │ Fixed authentication bug       │
   │ 4. main_go.20251113...   │ 2025-11-13 11:47:02 │  49.2 KB │ -                              │
   │ 5. main_go.20251113...   │ 2025-11-13 11:39:49 │  49.2 KB │ -                              │
   └──────────────────────────┴─────────────────────┴──────────┴────────────────────────────────┘

   Enter backup number to restore (1-5) or 0 to cancel: 2
   ✅ Successfully restored: main.go
   📦 From backup: main_go.20251118_141124...
   💬 Restore comment: "Restored from backup"

8. Configuration File Setup
---------------------------

.. code-block:: bash

   # Create configuration file
   pt config init

   # Edit pt.yml
   cat > pt.yml << EOF
   # PT Configuration File
   max_clipboard_size: 104857600    # 100MB
   max_backup_count: 100            # Keep 100 backups
   max_filename_length: 200         # Max filename length
   backup_dir_name: .pt             # Git-like directory
   max_search_depth: 10             # Max recursive search depth
   EOF

   # View current config
   pt config show

Output::

   Current PT Configuration:

   Max Clipboard Size: 104857600 bytes (100.0 MB)
   Max Backup Count: 100
   Max Filename Length: 200 characters
   Backup Directory: .pt/ (Git-like structure)
   Max Search Depth: 10 levels

   Config loaded from: ./pt.yml

9. Recursive File Search
------------------------

.. code-block:: bash

   # File not in current directory? PT finds it automatically!
   pt config.json

Output::

   🔍 Searching for 'config.json' in subdirectories...
   ✓ Found at: /path/to/project/src/config.json

Multiple files found::

   🔍 Found 3 matching file(s)

   ┌──────┬────────────────────────────┬─────────────────────┬──────────────┐
   │   #  │ Path                       │ Modified            │ Size         │
   ├──────┼────────────────────────────┼─────────────────────┼──────────────┤
   │    1 │ ./config.json              │ 2025-11-15 10:30:00 │ 2.1 KB      │
   │    2 │ ./src/config.json          │ 2025-11-14 15:20:00 │ 1.8 KB       │
   │    3 │ ./examples/config.json     │ 2025-11-13 09:15:00 │ 1.2 KB       │
   └──────┴────────────────────────────┴─────────────────────┴──────────────┘

   Enter file number to use (1-3) or 0 to cancel: 1
   ✓ Using: ./config.json

10. Visual Diff Comparison
---------------------------

.. code-block:: bash

   # Interactive diff - choose which backup to compare
   pt -d main.go

   # Quick diff with last backup
   pt -d main.go --last

Output::

   📊 Comparing with last backup: main_go.20251115_151804...
   Current file: /path/to/main.go
   Backup file:  /path/to/.pt/main_go/main_go.20251115_151804...

   [Beautiful colored diff output from delta]

11. Move recursive/tree
--------------------------

.. code-block:: bash

   C:\TEMP\test-pt>tree2
   📂 C:\TEMP\test-pt/
   ├── 📄 .gitignore (27.00 B)
   ├── 📁 dest/
   ├── 📁 dest_new/
   ├── 📄 test1.go (130.30 KB)
   ├── 📄 test2.go (130.30 KB)
   └── 📄 test3.go (130.30 KB)

   C:\TEMP\test-pt>pt move *.go dest
   🎯 Matched 3 file(s) from patterns

   🚚 Moving 3 file(s) with backup adjustment...
     Destination: C:\TEMP\test-pt\dest
     Type: Directory

   [1/3] Processing: test1.go
     📦 Found 2 backup(s)
   📁 Using existing .pt from: .pt/  ✅ Moved backups (2 metadata updated)
     ✅ Moved to: C:\TEMP\test-pt\dest\test1.go
   [2/3] Processing: test2.go
     📦 Found 1 backup(s)
   📁 Using existing .pt from: .pt/  ✅ Moved backups (1 metadata updated)
     ✅ Moved to: C:\TEMP\test-pt\dest\test2.go
   [3/3] Processing: test3.go
     📦 Found 1 backup(s)
   📁 Using existing .pt from: .pt/  ✅ Moved backups (1 metadata updated)
     ✅ Moved to: C:\TEMP\test-pt\dest\test3.go

   📊 Move Summary:
     ✅ 3 file(s) moved successfully
     📦 4 backup(s) adjusted

   C:\TEMP\test-pt>ls
   .gitignore  .pt\  dest\  dest_new\

   C:\TEMP\test-pt>pt -l test1.go
   🔍 Searching for 'test1.go' in subdirectories...
   ✓ Found: C:\TEMP\test-pt\dest\test1.go

   📂 Backup files for 'C:\TEMP\test-pt\dest\test1.go'
   Total: 2 backup(s) (stored in .pt/)

   ┌──────────────────────────────────────────┬─────────────────────┬──────────────┬────────────────────────────────┐
   │ File Name                                │ Modified            │         Size │ Comment                        │
   ├──────────────────────────────────────────┼─────────────────────┼──────────────┼────────────────────────────────┤
   │   1. test1_go.20251202_150950668211.3... │ 2025-12-02 15:09:50 │     130.3 KB │ Restored from last backup      │
   │   2. test1_go.20251202_150836676759.3... │ 2025-12-02 15:08:36 │     130.3 KB │ -                              │
   └──────────────────────────────────────────┴─────────────────────┴──────────────┴────────────────────────────────┘


   C:\TEMP\test-pt>move dest dest_new
           1 dir(s) moved.

   C:\TEMP\test-pt>pt -l test1.go
   🔍 Searching for 'test1.go' in subdirectories...
   ✓ Found: C:\TEMP\test-pt\dest_new\dest\test1.go
   ℹ️  No backups found for: C:\TEMP\test-pt\dest_new\dest\test1.go (check .pt/ directory)

12. Fix structure/tree 
------------------------

.. code-block:: bash

   C:\TEMP\test-pt>pt fix

   🔍 Scanning for orphaned backups...

   📂 Using .pt directory: C:\TEMP\test-pt\.pt

   ⚠️  Found 4 orphaned backup(s):

   [1] Orphaned backup: dest_main.go
       Expected: C:\TEMP\test-pt\dest\main.go (NOT FOUND)
       No matches found (file may be deleted)

   [2] Orphaned backup: dest_test1.go
       Expected: C:\TEMP\test-pt\dest\test1.go (NOT FOUND)
       Possible matches found:
         1) dest_new\dest\test1.go

   [3] Orphaned backup: dest_test2.go
       Expected: C:\TEMP\test-pt\dest\test2.go (NOT FOUND)
       Possible matches found:
         1) dest_new\dest\test2.go

   [4] Orphaned backup: dest_test3.go
       Expected: C:\TEMP\test-pt\dest\test3.go (NOT FOUND)
       Possible matches found:
         1) dest_new\dest\test3.go

   Options:
     1. Auto-fix: Update backup references for files with single match
     2. Manual: Select correct file for each orphaned backup
     3. Clean: Remove orphaned backups (files deleted)
     0. Cancel

   Choice: 1
   ✅ Fixed: test1.go -> test1.go
   ✅ Fixed: test2.go -> test2.go
   ✅ Fixed: test3.go -> test3.go

   📊 Result: 3 fixed, 1 skipped

   C:\TEMP\test-pt>pt -l test1.go
   🔍 Searching for 'test1.go' in subdirectories...
   ✓ Found: C:\TEMP\test-pt\dest_new\dest\test1.go

   📂 Backup files for 'C:\TEMP\test-pt\dest_new\dest\test1.go'
   Total: 2 backup(s) (stored in .pt/)

   ┌──────────────────────────────────────────┬─────────────────────┬──────────────┬────────────────────────────────┐
   │ File Name                                │ Modified            │         Size │ Comment                        │
   ├──────────────────────────────────────────┼─────────────────────┼──────────────┼────────────────────────────────┤
   │   1. test1_go.20251202_150950668211.3... │ 2025-12-02 15:09:50 │     130.3 KB │ Restored from last backup      │
   │   2. test1_go.20251202_150836676759.3... │ 2025-12-02 15:08:36 │     130.3 KB │ -                              │
   └──────────────────────────────────────────┴─────────────────────┴──────────────┴────────────────────────────────┘


   C:\TEMP\test-pt>tree2
   📂 C:\TEMP\test-pt/
   ├── 📄 .gitignore (27.00 B)
   └── 📁 dest_new/
       └── 📁 dest/
           ├── 📄 test1.go (130.30 KB)
           ├── 📄 test2.go (130.30 KB)
           └── 📄 test3.go (130.30 KB)

13. Directory Tree Visualization
--------------------------------

.. code-block:: bash

   # Show current directory tree
   pt -t

Output::

   myproject/
   ├── src/
   │   ├── main.go (15.2 KB) [modified]
   │   └── utils.go (3.4 KB)
   ├── .pt/
   │   ├── main_go/
   │   │   └── main_go.20251115_151804...
   │   └── utils_go/
   │       └── utils_go.20251115_143022...
   ├── README.md (2.1 KB)
   └── go.mod (456 B)

   2 directories, 5 files, 29.2 KB total

Exclude specific folders::

   pt -t -e node_modules,.git,dist,build

14. Complete Workflow Example
-----------------------------

Daily development session:

.. code-block:: bash

   # Start work, check status
   pt check

   # Make changes to files
   # ... edit main.go ...
   
   # Backup specific file with comment
   pt main.go -m "Added user authentication"

   # Commit all changes at end of day
   pt commit -m "Implemented auth module and updated tests"

   # Review what changed
   pt -l main.go
   pt -d main.go --last

   # Next day: continue working...