# 📋 `pt` — Clipboard to File Tool with Smart Version Management

[![Go Version](https://img.shields.io/badge/Go-1.16+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Version](https://img.shields.io/badge/version-1.0.15-blue.svg)](https://github.com/cumulus13/pt)

> **`pt`** is a powerful CLI tool that writes your clipboard content directly to a file — with automatic timestamped backups, **recursive file search**, **delta diff comparison**, directory tree visualization, and safe file deletion. **It's not just a clipboard manager — it's a complete version control system for your files!**

## ✨ Features

### Core Features
- 📝 **Quick Save** - Write clipboard content to file with one command
- 📦 **Auto Backup** - Automatic timestamped backups stored in `./backup/` directory
- ➕ **Append Mode** - Add content without creating backups
- 🔄 **Restore** - Interactive or quick restore from backups
- 📊 **Beautiful Listings** - Formatted table view of all backups with sizes
- 🔒 **Production Hardened** - Path validation, size limits, error handling
- 🎨 **Colorful Output** - ANSI colors for better readability
- 📈 **Audit Logging** - All operations logged for tracking

### Advanced Features
- 🔍 **Recursive File Search** - Automatically finds files in subdirectories up to 10 levels deep
- 📊 **Delta Diff Integration** - Beautiful side-by-side diff comparison with backups
- 🌳 **Directory Tree View** - Visual file structure with sizes (like `tree` command)
- 📁 **GitIgnore Support** - Respects `.gitignore` patterns in tree view
- 🗑️ **Safe Delete** - Backup before deletion, create empty placeholder
- ⚙️ **Exception Filtering** - Exclude specific files/folders from tree view
- 🎯 **Multi-File Selection** - Interactive prompt when multiple files found
- 🚀 **Smart Path Resolution** - Finds files anywhere in your project

### Version Management Capabilities
**PT acts as a lightweight version control system:**
- 📜 **Complete Version History** - Every file change is preserved
- 📙 **Easy Rollback** - Restore any previous version instantly
- 📊 **Version Comparison** - Diff any two versions visually with delta
- 🎯 **Zero Data Loss** - Never lose work, automatic backup before every write
- 💾 **Space Efficient** - Only changed files are backed up
- 🏷️ **Timestamped Versions** - Microsecond precision timestamps

## 🚀 Installation

### Prerequisites

- Go 1.16 or higher
- Git (for installation)
- **Delta** (optional, for diff functionality) - [Install from here](https://github.com/dandavison/delta)

### Install from Source

```bash
go install github.com/cumulus13/pt/pt@latest

# or Clone the repository
git clone https://github.com/cumulus13/pt.git
cd pt

# Build and install
go build -o pt pt/main.go

# Move to your PATH (Linux/macOS)
sudo mv pt /usr/local/bin/

# Or for Windows, add to your PATH manually
```

### Quick Install (Linux/macOS)

```bash
# One-liner installation
curl -sSL https://raw.githubusercontent.com/cumulus13/pt/main/install.sh | bash
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
# PT version 2.1.0
# Production-hardened clipboard to file tool
# Features: Recursive search, backup management, delta diff, tree view, safe delete
```

## 📖 Usage

### Basic Commands

```bash
# Write clipboard to file (creates backup if exists)
pt myfile.txt

# Append clipboard to file (no backup)
pt + myfile.txt

# List all backups with sizes and timestamps
pt -l myfile.txt

# Restore backup (interactive selection)
pt -r myfile.txt

# Restore last backup directly
pt -r myfile.txt --last

# Show help
pt --help

# Show version
pt --version
```

### Advanced Commands (NEW!)

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
pt --remove script.py       # Alternative syntax
```

## 📚 Examples

### 1. Quick Note Taking

```bash
# Copy some text to clipboard, then:
pt notes.txt
# ✅ Successfully written to: notes.txt
# 📝 Content size: 142 characters
```

### 2. Append to Log Files

```bash
# Copy error message, then:
pt + errors.log
# ✅ Successfully appended to: errors.log
# 📝 Content size: 87 characters
```

### 3. Code Snippet Management with Version Control

```bash
# Copy code from browser
pt snippet.py

# Later, copy another snippet (creates backup automatically)
pt snippet.py
# 📦 Backup created: snippet_py.20251115_151804177132.12345_a1b2c3d4

# List all versions with sizes
pt -l snippet.py
# 📂 Backup files for 'snippet.py'
# Total: 5 backup(s) (stored in ./backup/)
# [Beautiful table showing all versions]

# Compare current with previous version
pt -d snippet.py --last
# [Beautiful colored diff output powered by delta]

# Restore a specific version
pt -r snippet.py
# [Shows table, select version number]
```

### 4. Recursive File Search (NEW!)

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

### 5. Visual Diff Comparison (NEW!)

```bash
# Interactive diff - choose which backup to compare
pt -d main.go
# 📂 Backup files for 'main.go'
# [Shows list of backups]
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

### 6. Directory Tree Visualization

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
# Using .gitignore (12 patterns)
# Exceptions: node_modules, .git, dist

# Tree of specific directory with exceptions
pt -t ~/projects/myapp -e build,vendor,tmp
```

### 7. Safe File Deletion

```bash
# Delete file with automatic backup
pt -rm old_script.py
# 📦 Backup created: old_script_py.20251115_151804...
# 🗑️  File deleted: old_script.py
# 📝 Created empty placeholder: old_script.py
# ℹ️  Original content (1.2 KB) backed up to ./backup/

# Restore if needed
pt -r old_script.py --last
# ✅ Successfully restored: old_script.py
```

### 8. Working with Files in Subdirectories (NEW!)

```bash
# PT automatically searches subdirectories
cd ~/myproject

# These commands work even if files are in subdirectories:
pt app.config           # Finds ./src/config/app.config
pt -l database.sql      # Lists backups for ./db/migrations/database.sql
pt -d styles.css --last # Diffs ./frontend/css/styles.css with backup
pt -r utils.js          # Restores ./lib/helpers/utils.js

# If multiple files with same name exist, PT shows selection prompt
pt config.yaml
# 🔍 Found 2 matching file(s)
# 1. ./config.yaml
# 2. ./docker/config.yaml
# Enter file number to use (1-2) or 0 to cancel:
```

## 🎨 Output Examples

### Backup Listing with Sizes

```
📂 Backup files for 'myfile.txt'
Total: 5 backup(s) (stored in ./backup/)

┌────────────────────────────────────────────────────┬─────────────────────┬─────────────────┐
│ File Name                                          │ Modified            │ Size            │
├────────────────────────────────────────────────────┼─────────────────────┼─────────────────┤
│  1. myfile_txt.20251115_151804177132.12345_a1b2   │ 2025-11-15 15:18:04 │       2.45 KB   │
│  2. myfile_txt.20251115_143022123456.12344_b2c3   │ 2025-11-15 14:30:22 │       2.40 KB   │
│  3. myfile_txt.20251115_120000000000.12343_c3d4   │ 2025-11-15 12:00:00 │       1.98 KB   │
│  4. myfile_txt.20251114_180000000000.12342_d4e5   │ 2025-11-14 18:00:00 │       1.85 KB   │
│  5. myfile_txt.20251114_100000000000.12341_e5f6   │ 2025-11-14 10:00:00 │       1.52 KB   │
└────────────────────────────────────────────────────┴─────────────────────┴─────────────────┘
```

### Recursive File Search Results (NEW!)

```
🔍 Found 3 file(s):

┌──────┬──────────────────────────────────────────────────────────────┬─────────────────────┬──────────────┐
│   #  │ Path                                                         │ Modified            │ Size         │
├──────┼──────────────────────────────────────────────────────────────┼─────────────────────┼──────────────┤
│    1 │ ./config.json                                                │ 2025-11-15 10:30:45 │     1.2 KB   │
│    2 │ ./src/config/config.json                                     │ 2025-11-15 09:15:20 │     856 B    │
│    3 │ ./tests/fixtures/config.json                                 │ 2025-11-14 16:20:10 │     512 B    │
└──────┴──────────────────────────────────────────────────────────────┴─────────────────────┴──────────────┘

Enter file number to use (1-3) or 0 to cancel:
```

### Enhanced Help Message (NEW!)

```bash
pt --help

╔══════════════════════════════════════════════════════════════╗
║  PT - Clipboard to File Tool v2.1.0                          ║
╚══════════════════════════════════════════════════════════════╝

📝 BASIC OPERATIONS:
  pt <filename>               Write clipboard to file
  pt + <filename>             Append clipboard to file

📦 BACKUP OPERATIONS:
  pt -l <filename>            List all backups
  pt -r <filename>            Restore backup (interactive)
  pt -r <filename> --last     Restore most recent backup

📊 DIFF OPERATIONS:
  pt -d <filename>            Compare with backup (interactive)
  pt -d <filename> --last     Compare with most recent backup

ℹ️  INFORMATION:
  pt -h, --help               Show this help message
  pt -v, --version            Show version information

💡 EXAMPLES:
  $ pt notes.txt                # Save clipboard to notes.txt
  $ pt + log.txt                # Append clipboard to log.txt
  $ pt -l notes.txt             # List all backups
  $ pt -r notes.txt             # Interactive restore
  $ pt -d notes.txt --last      # Diff with most recent backup

🔍 RECURSIVE SEARCH:
  • If file not found in current directory, searches recursively
  • Maximum search depth: 10 levels
  • If multiple files found, prompts for selection
  • Skips ./backup/ directories automatically

📂 BACKUP SYSTEM:
  • Location: ./backup/ directory (auto-created)
  • Naming: <filename>_<ext>.<timestamp>.<unique-id>
  • Retention: Keeps most recent 100 backups per file
  • Auto-backup: Creates backup before overwriting existing files
  • Empty files: Not backed up (skipped automatically)

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

📄 LICENSE: MIT | AUTHOR: Hadi Cahyadi <cumulus13@gmail.com>
```

## 🗂️ Project Structure

```
pt/
├── backup/                         # Auto-created backup directory
│   ├── main_go.20251115_163913... # Timestamped backups
│   └── main_go.20251115_151804...
├── go.mod                          # Go module definition
├── go.sum                          # Dependency checksums
├── pt/
│   └── main.go                     # Main application code
├── README.md                       # This file
├── LICENSE                         # MIT License
├── .gitignore                      # Git ignore rules
└── install.sh                      # Installation script (optional)
```

### Backup Directory Structure

All backups are stored in a `./backup/` subdirectory relative to the file location:

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

## 🔧 Configuration

### Environment Variables

Currently, PT uses hardcoded limits for safety:

| Setting | Default | Range | Description |
|---------|---------|-------|-------------|
| **Max Clipboard Size** | 100MB | 1-1GB | Maximum content size |
| **Max Backup Count** | 100 | 1-10000 | Backups kept per file |
| **Max Filename Length** | 200 | 1-1000 | Maximum filename chars |
| **Max Search Depth** | 10 | 1-100 | Recursive search depth |
| **Backup Dir Name** | `backup` | - | Backup directory name |

To customize, edit constants in `main.go`:

```go
const (
    MaxClipboardSize = 100 * 1024 * 1024 // 100MB
    MaxBackupCount   = 100
    MaxFilenameLen   = 200
    BackupDirName    = "backup"
    MaxSearchDepth   = 10                 // NEW!
)
```

### Backup Naming Format

Backups use this format for zero-collision guarantee:
```
originalname_ext.YYYYMMDD_HHMMSS_MICROSECONDS.PID_RANDOMID
```

Example:
```
notes_txt.20251115_151804177132.12345_a1b2c3d4
```

Components:
- `notes_txt` - Original filename without extension
- `20251115_151804177132` - Timestamp with microsecond precision
- `12345` - Process ID
- `a1b2c3d4` - Random 8-character hex ID

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
- ✅ Validates recursive search depth (NEW!)

### Size Limits
- ✅ Maximum 100MB clipboard content
- ✅ Prevents disk exhaustion attacks
- ✅ Validates disk space before writing
- ✅ Checks write permissions

### Input Validation
- ✅ All user inputs sanitized
- ✅ Numeric inputs validated for range
- ✅ Graceful handling of malformed input
- ✅ Protected against command injection
- ✅ Safe file selection in multi-match scenarios (NEW!)

### Safe Operations
- ✅ Atomic-like file operations
- ✅ Verification of write completion
- ✅ Automatic rollback on errors
- ✅ Backup before destructive operations
- ✅ Backup directory exclusion from search (NEW!)

## ⚠️ Limitations

1. **Text Only** - Only supports text content (no binary clipboard data)
2. **Single File** - Operates on one file at a time
3. **Local Only** - No network or cloud storage support
4. **Platform Support** - Requires clipboard access (may need X11 on Linux headless)
5. **Delta Required** - Diff feature requires delta to be installed
6. **Search Depth** - Recursive search limited to 10 levels by default
7. **Backup Exclusion** - `./backup/` directories are automatically excluded from search

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
**Solution**: Content exceeds safety limit. Save directly from source application

### File Not Found (NEW!)
```bash
❌ Error: file not found: config.json
```
**Solutions**:
- Check filename spelling
- File might be deeper than 10 levels (increase MaxSearchDepth)
- Ensure file exists somewhere in the directory tree
- Use absolute path if outside search scope

### Multiple Files Found (NEW!)
```bash
🔍 Found 3 matching file(s)
[Table showing options]
Enter file number to use (1-3) or 0 to cancel:
```
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

### 1. Version Control System
Use PT as a lightweight version control for any text file:
- Track every change automatically
- Compare versions visually with delta (NEW!)
- Restore any previous version instantly
- No git repository needed
- Works with any file, anywhere (recursive search) (NEW!)

### 2. Quick Note Management
Perfect for rapid note-taking:
- Copy from anywhere, paste to file
- All versions preserved
- Easy to find and restore
- Find notes even if you forgot the exact location (NEW!)

### 3. Code Snippet Library
Build your snippet collection:
- Save snippets with one command
- Version history included
- Compare different versions with beautiful diffs (NEW!)
- Organize snippets in subdirectories (NEW!)

### 4. Configuration Management
Track configuration changes:
- Backup before every edit
- Compare with previous configs using delta (NEW!)
- Easy rollback on mistakes
- See what changed and when
- Find configs in complex project structures (NEW!)

### 5. Log File Management
Efficient log handling:
- Append mode for logs
- Tree view of log directories
- Compare log versions
- Safe cleanup with backups
- Track logs across different locations (NEW!)

### 6. Documentation Workflow
Better documentation management:
- Track all documentation changes
- Visual diff of updates with delta (NEW!)
- Restore previous versions
- Organized backup history
- Work with docs in any subdirectory (NEW!)

### 7. Multi-Project Workspace (NEW!)
Perfect for working across multiple projects:
- Quickly save files in any project subdirectory
- Automatic file location discovery
- Compare versions across project restructures
- Never lose track of where files are located

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
git clone https://github.com/cumulus13/pt.git
cd pt

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
- [ ] Config file support (`.ptrc`)
- [ ] Custom backup directory location
- [ ] Backup compression (gzip)
- [ ] Backup to cloud storage (S3, GCS)
- [ ] Web UI for backup management
- [ ] Backup cleanup strategies (by age, size)
- [ ] File watching mode (auto-backup on change)
- [ ] Backup metadata (tags, comments)
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

- 🐛 **Bug Reports**: [GitHub Issues](https://github.com/cumulus13/pt/issues)
- 💡 **Feature Requests**: [GitHub Discussions](https://github.com/cumulus13/pt/discussions)
- 📧 **Email**: cumulus13@gmail.com

**Made with ❤️ by Hadi Cahyadi**

*Your complete file version management system in a single command.* ⚡

If you find PT useful, consider supporting its development and please consider giving it a star on GitHub! ⭐:

[![Buy Me a Coffee](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/cumulus13)

[![Donate via Ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/cumulus13)
 
[Support me on Patreon](https://www.patreon.com/cumulus13)

## � Acknowledgments

- [atotto/clipboard](https://github.com/atotto/clipboard) - Cross-platform clipboard library
- [dandavison/delta](https://github.com/dandavison/delta) - Beautiful diff viewer
- Go community for excellent tooling and documentation
- All contributors and users


> 🌟 **PT: More than a clipboard tool — it's your file version manager!** 
> 
> Save, compare, restore, diff, and manage all your file versions effortlessly. Never lose work again!

## 🎓 Quick Start Tutorial

### 5-Minute Guide to Master PT

```bash
# 1. Install PT
go install github.com/cumulus13/pt/pt@latest

# 2. Save your first file
echo "Hello PT" | pbcopy  # Copy something
pt notes.txt              # Save to file

# 3. Make changes
echo "Hello PT v2" | pbcopy
pt notes.txt              # Creates backup automatically

# 4. See your versions
pt -l notes.txt           # List all versions

# 5. Compare versions
pt -d notes.txt --last    # Visual diff

# 6. Restore if needed
pt -r notes.txt --last    # Restore previous version

# 7. Explore your project
pt -t                     # See file tree

# Congratulations! You're now a PT expert! 🎉
```

---

**🔥 Start managing your file versions like a pro today!**