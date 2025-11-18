# 📋 `pt` – Clipboard to File Tool with Smart Version Management

[![Go Version](https://img.shields.io/badge/Go-1.16+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Version](https://img.shields.io/badge/version-1.0.21-blue.svg)](https://github.com/cumulus13/pt-go)

> **`pt`** is a powerful CLI tool that writes your clipboard content directly to a file – with automatic timestamped backups, **backup comments**, **recursive file search**, **delta diff comparison**, directory tree visualization, and safe file deletion. **It's not just a clipboard manager – it's a complete version control system for your files!**

## ✨ Features

### Core Features
- 📝 **Quick Save** - Write clipboard content to file with one command
- 📦 **Auto Backup** - Automatic timestamped backups stored in `./backup/` directory
- 💬 **Backup Comments** - Add descriptive comments to track why changes were made ✨ NEW!
- ➕ **Append Mode** - Add content without creating backups
- 🔄 **Restore** - Interactive or quick restore from backups with comments
- 📊 **Beautiful Listings** - Formatted table view of all backups with sizes and comments ✨ NEW!
- 🔒 **Production Hardened** - Path validation, size limits, error handling
- 🎨 **Colorful Output** - ANSI colors for better readability
- 📈 **Audit Logging** - All operations logged for tracking
- ✅ **Check Mode** - Skip writes if content unchanged (saves disk space) ✨ NEW!

### Advanced Features
- 🔍 **Recursive File Search** - Automatically finds files in subdirectories up to 10 levels deep
- 📊 **Delta Diff Integration** - Beautiful side-by-side diff comparison with backups
- 🌳 **Directory Tree View** - Visual file structure with sizes (like `tree` command)
- 📁 **GitIgnore Support** - Respects `.gitignore` patterns in tree view
- 🗑️ **Safe Delete** - Backup before deletion, create empty placeholder
- ⚙️ **Exception Filtering** - Exclude specific files/folders from tree view
- 🎯 **Multi-File Selection** - Interactive prompt when multiple files found
- 🚀 **Smart Path Resolution** - Finds files anywhere in your project
- ⚙️ **Configurable** - Customize behavior via `pt.yml` config file ✨ NEW!

### Version Management Capabilities with Comments ✨ NEW!
**PT acts as a lightweight version control system with descriptive comments:**
- 📜 **Complete Version History** - Every file change is preserved with optional comments
- 💬 **Comment System** - Track why changes were made, not just when
- 📝 **Contextual Notes** - Add meaningful descriptions to each backup
- 🔙 **Easy Rollback** - Restore any previous version instantly, see why it was saved
- 📊 **Version Comparison** - Diff any two versions visually with delta
- 🎯 **Zero Data Loss** - Never lose work, automatic backup before every write
- 💾 **Space Efficient** - Only changed files are backed up
- 🏷️ **Timestamped Versions** - Microsecond precision timestamps + human-readable comments

## 🚀 Installation

### Prerequisites

- Go 1.16 or higher
- Git (for installation)
- **Delta** (optional, for diff functionality) - [Install from here](https://github.com/dandavison/delta)

### Install from Source

```bash
go install github.com/cumulus13/pt-go/pt@latest

# or Clone the repository
git clone https://github.com/cumulus13/pt-go.git
cd pt-go

# Build and install
go build -o pt pt/main.go

# Move to your PATH (Linux/macOS)
sudo mv pt /usr/local/bin/

# Or for Windows, add to your PATH manually
```

### Quick Install (Linux/macOS)

```bash
# One-liner installation
curl -sSL https://raw.githubusercontent.com/cumulus13/pt-go/pt/main/install.sh | bash
```

### Install Delta (for diff feature)

```bash
# macOS
brew install git-delta

# Ubuntu/Debian
sudo apt install git-delta

# Arch Linux
sudo pacman -S git-delta

# Fedora/RHEL
sudo dnf install git-delta

# Windows (with Chocolatey)
choco install delta

# Windows (with Scoop)
scoop install delta

# Or download from: https://github.com/dandavison/delta/releases
```

### Verify Installation

```bash
pt --version
# PT version 1.0.19
# Production-hardened clipboard to file tool
# Features: Recursive search, backup management, delta diff, tree view, safe delete, configurable, comments
```

## 📖 Usage

### Basic Commands

```bash
# Write clipboard to file (creates backup if exists)
pt myfile.txt

# Write with comment ✨ NEW!
pt myfile.txt -m "Fixed bug in authentication logic"

# Write with check mode (skip if unchanged) ✨ NEW!
pt myfile.txt -c

# Combine check mode with comment ✨ NEW!
pt myfile.txt -c -m "Updated configuration"

# Append clipboard to file (no backup)
pt + myfile.txt

# Append with comment ✨ NEW!
pt + myfile.txt -m "Added new log entry"

# List all backups with sizes, timestamps, and comments ✨ NEW!
pt -l myfile.txt

# Restore backup (interactive selection)
pt -r myfile.txt

# Restore with comment ✨ NEW!
pt -r myfile.txt -m "Rolled back to stable version"

# Restore last backup directly
pt -r myfile.txt --last

# Restore last backup with comment ✨ NEW!
pt -r myfile.txt --last -m "Emergency rollback"

# Show help
pt --help

# Show version
pt --version
```

### Configuration Commands

```bash
# Initialize configuration file
pt config init              # Creates pt.yml in current directory
pt config init ~/.pt.yml    # Create in custom location

# Show current configuration
pt config show

# Show config file location
pt config path
```

### Advanced Commands

```bash
# 🔍 RECURSIVE SEARCH - Automatically finds files in subdirectories
pt config.json              # Searches up to 10 levels deep
pt -l utils.go              # List backups (searches recursively)
pt -r main.py               # Restore (searches recursively)

# 📊 DIFF OPERATIONS - Compare with backups using delta
pt -d myfile.txt            # Interactive: choose which backup to compare
pt -d myfile.txt --last     # Quick: compare with most recent backup
pt --diff script.py         # Alternative syntax

# 🌳 DIRECTORY TREE - Visualize file structure
pt -t                       # Show tree of current directory
pt -t /path/to/dir          # Show tree of specific directory
pt -t -e node_modules,.git  # Tree with exceptions (exclude folders)
pt -t /path -e build,dist   # Combine path and exceptions

# 🗑️ SAFE DELETE - Backup before deletion
pt -rm old_file.txt         # Backup, delete, create empty placeholder
pt -rm old_file.txt -m "Deprecated old implementation"  # With comment ✨ NEW!
pt --remove script.py       # Alternative syntax
```

## 📚 Examples

### 1. Quick Note Taking with Comments ✨ NEW!

```bash
# Copy some text to clipboard, then:
pt notes.txt -m "Meeting notes from sprint planning"
# ✅ Successfully written to: notes.txt
# 📄 Content size: 142 characters
# 💬 Comment: "Meeting notes from sprint planning"
```

### 2. Version Control for Code Changes ✨ NEW!

```bash
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
# Shows table with comments for each version
```

### 3. Configuration Management with Context ✨ NEW!

```bash
# Save production config
pt config.json -m "Production config for v2.1.0 release"

# Later, update for testing
pt config.json -m "Testing new cache settings"

# View all config versions with comments
pt -l config.json
# ┌────────────┬─────────────┬──────┬────────────────────────────┐
# │ File Name  │ Modified    │ Size │ Comment                    │
# ├────────────┼─────────────┼──────┼────────────────────────────┤
# │ 1. config..│ 14:12:41    │ 2 KB │ Testing new cache settings │
# │ 2. config..│ 10:30:15    │ 2 KB │ Production config v2.1.0   │
# └────────────┴─────────────┴──────┴────────────────────────────┘

# Restore production config
pt -r config.json -m "Reverting to production config"
```

### 4. Check Mode - Save Disk Space ✨ NEW!

```bash
# Only write if content actually changed
pt data.json -c
# ℹ️  Content identical to current file, no changes needed
# 📄 File: data.json

# Or with comment if it does change
pt data.json -c -m "Updated user preferences"
# 🔍 Content differs, proceeding with backup and write
# 📦 Backup created with comment
```

### 5. Safe Delete with Context ✨ NEW!

```bash
# Delete old implementation with explanation
pt -rm legacy_auth.py -m "Replaced by new OAuth2 implementation"
# 📦 Backup created: legacy_auth_py.20251118_141241...
# 💬 Comment: "Replaced by new OAuth2 implementation"
# 🗑️  File deleted: legacy_auth.py
# 📄 Created empty placeholder: legacy_auth.py
```

### 6. Append Mode with Comments ✨ NEW!

```bash
# Append log entries with context
pt + errors.log -m "Error logs from production incident"
# ✅ Successfully appended to: errors.log
# 📝 Content size: 87 characters
# 💬 Comment: "Error logs from production incident"
```

### 7. Interactive Restore with Comment History ✨ NEW!

```bash
pt -r main.go

# 📂 Backup files for 'main.go'
# Total: 5 backup(s) (stored in ./backup/)
# 
# ┌──────────────────────────┬─────────────────────┬──────────┬────────────────────────────────┐
# │ File Name                │ Modified            │     Size │ Comment                        │
# ├──────────────────────────┼─────────────────────┼──────────┼────────────────────────────────┤
# │ 1. main_go.20251118...   │ 2025-11-18 14:12:41 │  50.5 KB │ Add comment system             │
# │ 2. main_go.20251118...   │ 2025-11-18 14:11:24 │  57.0 KB │ Working version before refactor│
# │ 3. main_go.20251118...   │ 2025-11-18 13:43:01 │  52.6 KB │ Fixed authentication bug       │
# │ 4. main_go.20251113...   │ 2025-11-13 11:47:02 │  49.2 KB │ -                              │
# │ 5. main_go.20251113...   │ 2025-11-13 11:39:49 │  49.2 KB │ -                              │
# └──────────────────────────┴─────────────────────┴──────────┴────────────────────────────────┘
# 
# Enter backup number to restore (1-5) or 0 to cancel: 2
# ✅ Successfully restored: main.go
# 📦 From backup: main_go.20251118_141124...
# 💬 Restore comment: "Restored from backup"
```

### 8. Configuration File ✨ NEW!

```bash
# Create configuration file
pt config init

# Edit pt.yml
cat > pt.yml << EOF
# PT Configuration File
max_clipboard_size: 104857600    # 100MB
max_backup_count: 100            # Keep 100 backups
max_filename_length: 200         # Max filename length
backup_dir_name: backup          # Backup directory name
max_search_depth: 10             # Max recursive search depth
EOF

# View current config
pt config show
# 
# Current PT Configuration:
# 
# Max Clipboard Size: 104857600 bytes (100.0 MB)
# Max Backup Count: 100
# Max Filename Length: 200 characters
# Backup Directory: backup/
# Max Search Depth: 10 levels
# 
# Config loaded from: ./pt.yml
```

### 9. Recursive File Search

```bash
# File not in current directory? PT finds it automatically!
pt config.json
# 🔍 Searching for 'config.json' recursively...
# ✓ Found at: /path/to/project/src/config.json

# Multiple files found? PT shows interactive selection
pt README.md
# 🔍 Found 3 matching file(s)
# 
# ┌──────┬────────────────────────────┬─────────────────────┬──────────────┐
# │   #  │ Path                       │ Modified            │ Size         │
# ├──────┼────────────────────────────┼─────────────────────┼──────────────┤
# │    1 │ ./README.md                │ 2025-11-15 10:30:00 │ 15.2 KB      │
# │    2 │ ./docs/README.md           │ 2025-11-14 15:20:00 │ 8.5 KB       │
# │    3 │ ./examples/README.md       │ 2025-11-13 09:15:00 │ 3.2 KB       │
# └──────┴────────────────────────────┴─────────────────────┴──────────────┘
# 
# Enter file number to use (1-3) or 0 to cancel: 1
# ✓ Using: ./README.md
```

### 10. Visual Diff Comparison

```bash
# Interactive diff - choose which backup to compare
pt -d main.go
# 📂 Backup files for 'main.go'
# [Shows list of backups with comments]
# Enter backup number to compare (1-5) or 0 to cancel: 1
# 📊 Comparing with backup: main_go.20251115_120000...
# [Beautiful side-by-side diff powered by delta]

# Quick diff with last backup
pt -d main.go --last
# 📊 Comparing with last backup: main_go.20251115_151804...
# Current file: /path/to/main.go
# Backup file:  /path/to/backup/main_go.20251115_151804...
# [Beautiful colored diff output]
```

### 11. Directory Tree Visualization

```bash
# Show current directory tree
pt -t
# myproject/
# ├── src/
# │   ├── main.go (15.2 KB)
# │   └── utils.go (3.4 KB)
# ├── backup/
# │   └── main_go.20251115_101530.12345 (8.1 KB)
# ├── README.md (2.1 KB)
# └── go.mod (456 B)
# 
# 2 directories, 5 files, 29.2 KB total

# Exclude specific folders
pt -t -e node_modules,.git,dist
```

## 🎨 Output Examples

### Backup Listing with Comments ✨ NEW!

```
📂 Backup files for 'myfile.txt'
Total: 5 backup(s) (stored in ./backup/)

┌──────────────────────────────────────┬─────────────────────┬──────────────┬──────────────────────────────────────┐
│ File Name                            │ Modified            │         Size │ Comment                              │
├──────────────────────────────────────┼─────────────────────┼──────────────┼──────────────────────────────────────┤
│ 1. myfile_txt.20251118_141241...     │ 2025-11-18 14:12:41 │      2.45 KB │ Add comment system                   │
│ 2. myfile_txt.20251118_140030...     │ 2025-11-18 14:00:30 │      2.40 KB │ Fixed bug in auth logic              │
│ 3. myfile_txt.20251118_120000...     │ 2025-11-18 12:00:00 │      1.98 KB │ Updated configuration                │
│ 4. myfile_txt.20251114_180000...     │ 2025-11-14 18:00:00 │      1.85 KB │ -                                    │
│ 5. myfile_txt.20251114_100000...     │ 2025-11-14 10:00:00 │      1.52 KB │ -                                    │
└──────────────────────────────────────┴─────────────────────┴──────────────┴──────────────────────────────────────┘
```

### Check Mode Output ✨ NEW!

```bash
# When content is identical
pt data.json -c
# ℹ️  Content identical to current file, no changes needed
# 📄 File: data.json

# When content differs
pt data.json -c -m "Updated schema"
# 🔍 Content differs, proceeding with backup and write
# 📦 Backup created: data_json.20251118_141241...
# 💬 Comment: "Updated schema"
# ✅ Successfully written to: data.json
```

## 🗂️ Project Structure

```
pt/
├── backup/                         # Auto-created backup directory
│   ├── main_go.20251115_163913... # Timestamped backups
│   ├── main_go.20251115_163913.....meta.json # Metadata with comments ✨ NEW!
│   └── main_go.20251115_151804...
├── go.mod                          # Go module definition
├── go.sum                          # Dependency checksums
├── pt/
│   └── main.go                     # Main application code
├── pt.yml                          # Configuration file (optional) ✨ NEW!
├── README.md                       # This file
├── LICENSE                         # MIT License
├── VERSION                         # Version file
├── .gitignore                      # Git ignore rules
└── install.sh                      # Installation script (optional)
```

### Backup Metadata Format ✨ NEW!

All backups are stored in a `./backup/` subdirectory relative to the file location. Each backup now has an associated `.meta.json` file:

```
project/
├── myfile.txt                      # Current version
├── backup/                         # Backup directory (auto-created)
│   ├── myfile_txt.20251115_151804... # Backup 1
│   ├── myfile_txt.20251115_143022... # Backup 2
│   └── myfile_txt.20251115_120000... # Backup 3
├── src/
│   ├── app.js                      # Current version
│   └── backup/                     # Separate backup dir per location
│       └── app_js.20251115_120000...
└── other_files.txt
```

```json
{
  "comment": "Fixed authentication bug",
  "timestamp": "2025-11-18T14:12:41.500000Z",
  "size": 51712,
  "original_file": "/path/to/main.go"
}
```

## 🔧 Configuration

### Configuration File (pt.yml) ✨ NEW!

PT now supports configuration via `pt.yml` file. Search locations (in order):

1. `./pt.yml` or `./pt.yaml` (current directory)
2. `~/.config/pt/pt.yml` or `~/.config/pt/pt.yaml`
3. `~/pt.yml` or `~/pt.yaml` (home directory)

#### Create Config File

```bash
# Generate sample config
pt config init

# Or create manually
cat > pt.yml << EOF
# PT Configuration File
max_clipboard_size: 104857600    # 100MB (1-1GB)
max_backup_count: 100            # 100 backups (1-10000)
max_filename_length: 200         # 200 chars (1-1000)
backup_dir_name: backup          # "backup" directory
max_search_depth: 10             # 10 levels (1-100)
EOF
```

#### Configuration Options

| Setting | Default | Range | Description |
|---------|---------|-------|-------------|
| **max_clipboard_size** | 104857600 (100MB) | 1 - 1073741824 (1GB) | Maximum clipboard content size |
| **max_backup_count** | 100 | 1 - 10000 | Maximum backups kept per file |
| **max_filename_length** | 200 | 1 - 1000 | Maximum filename length |
| **backup_dir_name** | backup | - | Backup directory name |
| **max_search_depth** | 10 | 1 - 100 | Recursive search depth |

#### View Configuration

```bash
# Show current config
pt config show

# Show config file location
pt config path
```

### Backup Naming Format

Backups use this format for zero-collision guarantee:
```
originalname_ext.YYYYMMDD_HHMMSS_MICROSECONDS.PID_RANDOMID
```

Example:
```
notes_txt.20251115_151804177132.12345_a1b2c3d4
notes_txt.20251115_151804177132.12345_a1b2c3d4.meta.json  ✨ NEW!
```

Components:
- `notes_txt` - Original filename without extension
- `20251115_151804177132` - Timestamp with microsecond precision
- `12345` - Process ID
- `a1b2c3d4` - Random 8-character hex ID
- `.meta.json` - Metadata file with comment ✨ NEW!

This ensures **zero collision** risk even with:
- Multiple concurrent PT instances
- Same-second operations
- Parallel processing
- Multiple files with same name in different directories

## 🔒 Security Features

### Path Validation
- ✅ Prevents path traversal attacks (`../../../etc/passwd`)
- ✅ Blocks writes to system directories (`/etc`, `/sys`, `C:\Windows`)
- ✅ Validates filename length limits
- ✅ Sanitizes all file paths
- ✅ Validates recursive search depth
- ✅ Validates configuration values ✨ NEW!

### Size Limits
- ✅ Maximum 100MB clipboard content (configurable) ✨ NEW!
- ✅ Prevents disk exhaustion attacks
- ✅ Validates disk space before writing
- ✅ Checks write permissions

### Input Validation
- ✅ All user inputs sanitized
- ✅ Numeric inputs validated for range
- ✅ Graceful handling of malformed input
- ✅ Protected against command injection
- ✅ Safe file selection in multi-match scenarios
- ✅ Configuration file validation ✨ NEW!

### Safe Operations
- ✅ Atomic-like file operations
- ✅ Verification of write completion
- ✅ Automatic rollback on errors
- ✅ Backup before destructive operations
- ✅ Backup directory exclusion from search
- ✅ Metadata integrity checks ✨ NEW!

## ⚠️ Limitations

1. **Text Only** - Only supports text content (no binary clipboard data)
2. **Single File** - Operates on one file at a time
3. **Local Only** - No network or cloud storage support
4. **Platform Support** - Requires clipboard access (may need X11 on Linux headless)
5. **Delta Required** - Diff feature requires delta to be installed
6. **Search Depth** - Recursive search limited to configurable depth (default 10)
7. **Backup Exclusion** - Configured backup directories excluded from search
8. **Comment Length** - No enforced limit on comment length ✨ NEW!

## 🛠 Troubleshooting

### Clipboard Empty Error
```bash
⚠️  Warning: Clipboard is empty
```
**Solution**: Copy some text to clipboard before running PT

### No Write Permission
```bash
❌ Error: no write permission in directory
```
**Solution**: Check directory permissions or use a different location

### File Too Large
```bash
❌ Error: clipboard content too large (max 100MB)
```
**Solution**: Content exceeds safety limit. Increase `max_clipboard_size` in config or save directly

### File Not Found (NEW!)
```bash
❌ Error: file not found: config.json
```
**Solutions**:
- Check filename spelling
- File might be deeper than 10 levels (increase MaxSearchDepth)
- Ensure file exists somewhere in the directory tree
- Use absolute path if outside search scope

### Config File Issues ✨ NEW!
```bash
⚠️  Warning: invalid max_clipboard_size, using default
```
**Solution**: Check config file syntax and value ranges. Use `pt config show` to verify

### Content Unchanged (Check Mode) ✨ NEW!
```bash
ℹ️  Content identical to current file, no changes needed
```
**This is normal**: Check mode (`-c`) prevents unnecessary writes when content hasn't changed
**Solution**: Select the file number you want to work with, or press 0 to cancel

### Linux Clipboard Issues
```bash
❌ Error: failed to read clipboard
```
**Solution**: Install clipboard utilities:
```bash
# Ubuntu/Debian
sudo apt-get install xclip xsel

# Fedora/RHEL
sudo dnf install xclip xsel

# Arch
sudo pacman -S xclip xsel
```

### Delta Not Found
```bash
❌ Error: delta not installed. Install it from: https://github.com/dandavison/delta
```
**Solution**: Install delta for diff functionality:
```bash
# macOS
brew install git-delta

# Ubuntu/Debian
sudo apt install git-delta

# Arch Linux
sudo pacman -S git-delta

# Windows
scoop install delta
```

### Recursive Search Issues (NEW!)
```bash
🔍 Searching for 'file.txt' recursively...
❌ Error: file not found: file.txt
```
**Possible causes**:
- File is deeper than 10 directory levels
- File is in a `backup` subdirectory (automatically excluded)
- Permission issues reading some directories
- Typo in filename

**Solutions**:
- Use absolute path: `pt /full/path/to/file.txt`
- Increase MaxSearchDepth in code
- Check file permissions
- Verify filename spelling

## 🧪 Testing

### Manual Testing

```bash
# Test basic write
echo "Hello World" | pbcopy  # macOS (or xclip -selection clipboard on Linux)
pt test.txt

# Test append
echo "Line 2" | pbcopy
pt + test.txt

# Test backup listing
pt -l test.txt

# Test restore
pt -r test.txt --last

# Test diff
pt -d test.txt --last

# Test recursive search (NEW!)
mkdir -p deep/nested/path
echo "test" | pbcopy
pt deep/nested/path/file.txt
cd ../../..
pt -l file.txt  # Should find it automatically

# Test multiple file selection (NEW!)
echo "test1" | pbcopy
pt test.txt
mkdir subdir
echo "test2" | pbcopy
pt subdir/test.txt
pt test.txt  # Should show selection prompt

# Test tree
pt -t

# Test safe delete
pt -rm test.txt
```

### Run Tests (if implemented)

```bash
go test ./...
```

## 📊 Performance

| Operation | Performance | Notes |
|-----------|-------------|-------|
| **Startup Time** | < 50ms | Very fast startup |
| **Write Speed** | Disk I/O limited | Depends on storage |
| **Memory Usage** | ~5MB + content | Minimal footprint |
| **Backup Creation** | < 100ms | For typical files |
| **Search Speed** | < 200ms | For 1000+ files (NEW!) |
| **Multi-file Search** | < 500ms | Up to 10 levels deep (NEW!) |
| **Tree Generation** | < 500ms | For 5000+ files/folders |
| **Diff Rendering** | Depends on delta | Powered by delta |

## 🎯 Use Cases

### 1. Version Control System with Context ✨ NEW!
Use PT as a lightweight version control with meaningful comments:
- Track every change with why it was made
- Compare versions visually with delta
- Restore any previous version instantly
- **See the context** of each change through comments
- No git repository needed

### 2. Configuration Management with Audit Trail ✨ NEW!
Perfect for tracking config changes:
- Document why each config change was made
- Check mode prevents duplicate backups
- Compare configurations visually
- Easy rollback with context
- Complete audit trail with comments

### 3. Code Snippet Library with Notes ✨ NEW!
Build your snippet collection with context:
- Save snippets with descriptive comments
- Version history with reasons for changes
- Compare different versions
- Organize with meaningful metadata

### 4. Documentation Workflow with Tracking ✨ NEW!
Better documentation management:
- Track all changes with explanations
- Visual diff of updates
- Restore previous versions
- **Know why changes were made**
- Organized backup history with comments

### 5. Emergency Rollback with Context ✨ NEW!
Quick recovery with understanding:
- Instant rollback to any version
- See why each version was saved
- Compare what changed
- Document the rollback reason
- Complete incident tracking

## 🤝 Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Setup

```bash
# Clone your fork
git clone https://github.com/cumulus13/pt-go.git
cd pt-go

# Install dependencies
go mod download

# Run in development
go run pt/main.go --help

# Build
go build -o pt pt/main.go

# Format code
go fmt ./...

# Run linters (optional)
golangci-lint run
```

### Feature Ideas

Want to contribute? Here are some ideas:
- [x] Config file support (`.ptrc`) (✅ DONE in v1.0.19)
- [x] Backup comments/metadata (✅ DONE in v1.0.19)
- [x] Check mode to skip identical content (✅ DONE in v1.0.19)
- [ ] Custom backup directory location (absolute path)
- [ ] Backup compression (gzip)
- [ ] Backup to cloud storage (S3, GCS)
- [ ] Web UI for backup management
- [ ] Backup cleanup strategies (by age, size)
- [ ] File watching mode (auto-backup on change)
- [ ] Backup tags (additional metadata)
- [ ] Multi-file operations
- [ ] Backup encryption
- [x] Recursive file search (✅ DONE in v2.1.0)
- [x] Delta diff integration (✅ DONE in v2.1.0)
- [x] Interactive file selection (✅ DONE in v2.1.0)

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

```
MIT License

Copyright (c) 2025 Hadi Cahyadi

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

## 💻 Author

[**Hadi Cahyadi**](mailto:cumulus13@gmail.com)

- GitHub: [@cumulus13](https://github.com/cumulus13)
- Email: cumulus13@gmail.com

## 💖 Support

- 🐛 **Bug Reports**: [GitHub Issues](https://github.com/cumulus13/pt-go/issues)
- 💡 **Feature Requests**: [GitHub Discussions](https://github.com/cumulus13/pt-go/discussions)
- 📧 **Email**: cumulus13@gmail.com

**Made with ❤️ by Hadi Cahyadi**

*Your complete file version management system with contextual history in a single command.* ⚡

If you find PT useful, consider supporting its development and please consider giving it a star on GitHub! ⭐:

[![Buy Me a Coffee](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/cumulus13)

[![Donate via Ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/cumulus13)
 
[Support me on Patreon](https://www.patreon.com/cumulus13)

## 🙏 Acknowledgments

- [atotto/clipboard](https://github.com/atotto/clipboard) - Cross-platform clipboard library
- [dandavison/delta](https://github.com/dandavison/delta) - Beautiful diff viewer
- [gopkg.in/yaml.v3](https://gopkg.in/yaml.v3) - YAML parser for Go
- Go community for excellent tooling and documentation
- All contributors and users

> 🌟 **PT: More than a clipboard tool – it's your file version manager with context!** 
> 
> Save, compare, restore, diff, and manage all your file versions effortlessly with meaningful comments. Never lose work again, and always know why changes were made!

## 🎓 Quick Start Tutorial

### 5-Minute Guide to Master PT

```bash
# 1. Install PT
go install github.com/cumulus13/pt-go/pt@latest

# 2. Save your first file with a comment ✨ NEW!
echo "Hello PT" | pbcopy
pt notes.txt -m "Initial version"

# 3. Make changes with context ✨ NEW!
echo "Hello PT v2" | pbcopy
pt notes.txt -m "Added version number"

# 4. See your versions with comments ✨ NEW!
pt -l notes.txt
# Shows table with all versions and their comments

# 5. Use check mode to save space ✨ NEW!
echo "Hello PT v2" | pbcopy  # Same content
pt notes.txt -c -m "Attempted update"
# ℹ️  Content identical, no backup created

# 6. Compare versions
pt -d notes.txt --last

# 7. Restore if needed with context ✨ NEW!
pt -r notes.txt --last -m "Rollback for testing"

# 8. Set up your preferences ✨ NEW!
pt config init
pt config show

# Congratulations! You're now a PT expert with version context! 🎉
```

---

**🔥 Start managing your file versions with meaningful context like a pro today!**
