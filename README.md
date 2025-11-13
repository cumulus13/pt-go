# 📋 `pt` — Clipboard to File Tool with Smart Version Management

[![Go Version](https://img.shields.io/badge/Go-1.16+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Version](https://img.shields.io/badge/version-1.0.11-blue.svg)](https://github.com/cumulus13/pt)

> **`pt`** is a powerful CLI tool that writes your clipboard content directly to a file — with automatic timestamped backups, recursive file search, delta diff comparison, directory tree visualization, and safe file deletion. **It's not just a clipboard manager — it's a complete version control system for your files!**

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
- 🔍 **Recursive File Search** - Automatically finds files in subdirectories
- 📊 **Delta Diff Integration** - Beautiful side-by-side diff comparison with backups
- 🌳 **Directory Tree View** - Visual file structure with sizes (like `tree` command)
- 📝 **GitIgnore Support** - Respects `.gitignore` patterns in tree view
- 🗑️ **Safe Delete** - Backup before deletion, create empty placeholder
- ⚙️ **Exception Filtering** - Exclude specific files/folders from tree view

### Version Management Capabilities
**PT acts as a lightweight version control system:**
- 📜 **Complete Version History** - Every file change is preserved
- 🔙 **Easy Rollback** - Restore any previous version instantly
- 📊 **Version Comparison** - Diff any two versions visually
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

# Windows (with Chocolatey)
choco install delta

# Or download from: https://github.com/dandavison/delta/releases
```

### Verify Installation

```bash
pt --version
# PT version 2.2.0
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

### Advanced Commands

```bash
# Compare file with backup (interactive)
pt -d myfile.txt

# Compare with last backup directly
pt -d myfile.txt --last

# Delete file safely (backup first, create empty placeholder)
pt -rm myfile.txt

# Show directory tree with file sizes
pt -t

# Show tree of specific directory
pt -t /path/to/dir

# Tree with exceptions (exclude files/folders)
pt -t -e node_modules,.git,dist

# Tree with path and multiple exceptions
pt -t /path/to/project -e .env,build,__pycache__
```

## 📚 Examples

### 1. Quick Note Taking

```bash
# Copy some text to clipboard, then:
pt notes.txt
# ✅ Successfully written to: notes.txt
# 📄 Content size: 142 characters
```

### 2. Append to Log Files

```bash
# Copy error message, then:
pt + errors.log
# ✅ Successfully appended to: errors.log
# 📄 Content size: 87 characters
```

### 3. Code Snippet Management with Version Control

```bash
# Copy code from browser
pt snippet.py

# Later, copy another snippet
pt snippet.py
# 📦 Backup created: snippet_py.20251113_151804177132.12345_a1b2c3d4

# List all versions
pt -l snippet.py

# Compare current with previous version
pt -d snippet.py --last

# Restore a specific version
pt -r snippet.py
# Shows table, select version number
```

### 4. Recursive File Search

```bash
# File not in current directory? PT finds it automatically!
pt config.json
# 🔍 Searching for 'config.json' in subdirectories...
# ✓ Found: /path/to/project/src/config.json

# Multiple files found? PT shows options
pt README.md
# 🔍 Found 3 file(s):
# 1. ./README.md
# 2. ./docs/README.md
# 3. ./examples/README.md
# Enter file number to use (1-3) or 0 to cancel:
```

### 5. Visual Diff Comparison

```bash
# Interactive diff - choose which backup to compare
pt -d main.go
# Shows list of backups, select one
# Beautiful side-by-side diff powered by delta

# Quick diff with last backup
pt -d main.go --last
# 📊 Comparing with last backup: main_go.20251113_120000...
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
# │   └── main_go.20251113_101530.12345 (8.1 KB)
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
# 📦 Backup created: old_script_py.20251113_151804...
# 🗑️  File deleted: old_script.py
# 📄 Created empty placeholder: old_script.py
# ℹ️  Original content (1.2 KB) backed up to ./backup/

# Restore if needed
pt -r old_script.py --last
```

## 🎨 Output Examples

### Backup Listing with Sizes

```
📂 Backup files for 'myfile.txt'
Total: 5 backup(s) (stored in ./backup/)

┌────────────────────────────────────────────────────┬─────────────────────┬─────────────────┐
│ File Name                                          │ Modified            │ Size            │
├────────────────────────────────────────────────────┼─────────────────────┼─────────────────┤
│  1. myfile_txt.20251113_151804177132.12345_a1b2   │ 2025-11-13 15:18:04 │       2.45 KB   │
│  2. myfile_txt.20251113_143022123456.12344_b2c3   │ 2025-11-13 14:30:22 │       2.40 KB   │
│  3. myfile_txt.20251113_120000000000.12343_c3d4   │ 2025-11-13 12:00:00 │       1.98 KB   │
│  4. myfile_txt.20251112_180000000000.12342_d4e5   │ 2025-11-12 18:00:00 │       1.85 KB   │
│  5. myfile_txt.20251112_100000000000.12341_e5f6   │ 2025-11-12 10:00:00 │       1.52 KB   │
└────────────────────────────────────────────────────┴─────────────────────┴─────────────────┘
```

### Directory Tree with Sizes

```
myproject/
├── backup/
│   ├── main_go.20251113_101530177132.12345_a1b2 (14.8 KB)
│   └── utils_go.20251113_093000123456.12344_b2c3 (3.2 KB)
├── cmd/
│   └── server/
│       └── main.go (5.6 KB)
├── internal/
│   ├── config/
│   │   ├── config.go (2.3 KB)
│   │   └── parser.go (1.8 KB)
│   └── handlers/
│       └── http.go (4.2 KB)
├── .gitignore (234 B)
├── go.mod (456 B)
├── go.sum (2.1 KB)
└── README.md (8.5 KB)

6 directories, 11 files, 43.2 KB total
Using .gitignore (8 patterns)
```

### Recursive File Search Results

```
🔍 Found 3 file(s):

┌──────────────────────────────────────────────────────────────┬─────────────────────┬──────────────┐
│ Path                                                         │ Modified            │ Size         │
├──────────────────────────────────────────────────────────────┼─────────────────────┼──────────────┤
│  1. ./config.json                                            │ 2025-11-13 10:30:45 │     1.2 KB   │
│  2. ./src/config.json                                        │ 2025-11-13 09:15:20 │     856 B    │
│  3. ./tests/fixtures/config.json                             │ 2025-11-12 16:20:10 │     512 B    │
└──────────────────────────────────────────────────────────────┴─────────────────────┴──────────────┘

Enter file number to use (1-3) or 0 to cancel:
```

## 🗂️ Project Structure

```
pt/
├── backup/                         # Auto-created backup directory
│   ├── main_go.20251113_163913... # Timestamped backups
│   └── main_go.20251113_151804...
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
│   ├── myfile_txt.20251113_151804... # Backup 1
│   ├── myfile_txt.20251113_143022... # Backup 2
│   └── myfile_txt.20251113_120000... # Backup 3
└── other_files.txt
```

## 🔧 Configuration

### Environment Variables

Currently, PT uses hardcoded limits for safety:

- **Max Clipboard Size**: 100MB
- **Max Backup Count**: 100 backups per file
- **Max Filename Length**: 200 characters
- **Max Search Depth**: 10 directory levels

To customize, edit the constants in `pt.yml`:

```yaml
# PT Configuration File
# This file configures the behavior of the PT tool
# All values are optional - if not specified, defaults will be used

# Maximum clipboard content size in bytes (default: 104857600 = 100MB)
# Range: 1 - 1073741824 (1GB)
max_clipboard_size: 104857600

# Maximum number of backup files to keep per file (default: 100)
# Range: 1 - 10000
# Older backups are automatically removed when limit is reached
max_backup_count: 100

# Maximum filename length in characters (default: 200)
# Range: 1 - 1000
max_filename_length: 200

# Name of the backup directory (default: "backup")
# Backups will be stored in ./{backup_dir_name}/ relative to the file location
backup_dir_name: backup

# Maximum directory depth for recursive file search (default: 10)
# Range: 1 - 100
# When a file is not found in current directory, PT searches subdirectories up to this depth
max_search_depth: 10
```

### Backup Naming Format

Backups use this format for zero-collision guarantee:
```
originalname_ext.YYYYMMDD_HHMMSS_MICROSECONDS.PID_RANDOMID
```

Example:
```
notes_txt.20251113_151804177132.12345_a1b2c3d4
```

Components:
- `notes_txt` - Original filename without extension
- `20251113_151804177132` - Timestamp with microsecond precision
- `12345` - Process ID
- `a1b2c3d4` - Random 8-character hex ID

This ensures **zero collision** risk even with:
- Multiple concurrent PT instances
- Same-second operations
- Parallel processing

## 🔒 Security Features

### Path Validation
- ✅ Prevents path traversal attacks (`../../../etc/passwd`)
- ✅ Blocks writes to system directories (`/etc`, `/sys`, `C:\Windows`)
- ✅ Validates filename length limits
- ✅ Sanitizes all file paths

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

### Safe Operations
- ✅ Atomic-like file operations
- ✅ Verification of write completion
- ✅ Automatic rollback on errors
- ✅ Backup before destructive operations

## ⚠️ Limitations

1. **Text Only** - Only supports text content (no binary clipboard data)
2. **Single File** - Operates on one file at a time
3. **Local Only** - No network or cloud storage support
4. **Platform Support** - Requires clipboard access (may need X11 on Linux headless)
5. **Delta Required** - Diff feature requires delta to be installed

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
❌ Error: delta is not installed
```
**Solution**: Install delta for diff functionality:
```bash
# See installation instructions in Prerequisites section
brew install git-delta  # macOS
sudo apt install git-delta  # Ubuntu/Debian
```

### Recursive Search Not Finding File
```bash
❌ Error: file 'config.json' not found in current directory or subdirectories
```
**Solution**: 
- Check filename spelling
- File might be deeper than 10 levels (default max depth)
- File might be in a `.gitignore`d directory (for tree command)

## 🧪 Testing

### Manual Testing

```bash
# Test basic write
echo "Hello World" | pbcopy  # macOS (or xclip on Linux)
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

- **Startup Time**: < 50ms
- **Write Speed**: Limited by clipboard and disk I/O
- **Memory Usage**: ~5MB base + content size
- **Backup Creation**: < 100ms for typical files
- **Search Speed**: < 200ms for 1000+ files
- **Tree Generation**: < 500ms for 5000+ files/folders

## 🎯 Use Cases

### 1. Version Control System
Use PT as a lightweight version control for any text file:
- Track every change automatically
- Compare versions visually
- Restore any previous version
- No git repository needed

### 2. Quick Note Management
Perfect for rapid note-taking:
- Copy from anywhere, paste to file
- All versions preserved
- Easy to find and restore

### 3. Code Snippet Library
Build your snippet collection:
- Save snippets with one command
- Version history included
- Compare different versions
- Organize with tree view

### 4. Configuration Management
Track configuration changes:
- Backup before every edit
- Compare with previous configs
- Easy rollback on mistakes
- See what changed and when

### 5. Log File Management
Efficient log handling:
- Append mode for logs
- Tree view of log directories
- Compare log versions
- Safe cleanup with backups

### 6. Documentation Workflow
Better documentation management:
- Track all documentation changes
- Visual diff of updates
- Restore previous versions
- Organized backup history

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

If you find PT useful, consider supporting its development:

[![Buy Me a Coffee](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/cumulus13)

[![Donate via Ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/cumulus13)
 
[Support me on Patreon](https://www.patreon.com/cumulus13)

## 🙏 Acknowledgments

- [atotto/clipboard](https://github.com/atotto/clipboard) - Cross-platform clipboard library
- [dandavison/delta](https://github.com/dandavison/delta) - Beautiful diff viewer
- Go community for excellent tooling and documentation
- All contributors and users

## 📞 Support

- 🐛 **Bug Reports**: [GitHub Issues](https://github.com/cumulus13/pt/issues)
- 💡 **Feature Requests**: [GitHub Discussions](https://github.com/cumulus13/pt/discussions)
- 📧 **Email**: cumulus13@gmail.com

## ⭐ Star History

If you find PT useful, please consider giving it a star on GitHub! ⭐

## 🚀 Roadmap

### Version 1.0.10 (Planned)
- [ ] Backup compression
- [ ] Config file support
- [ ] Backup statistics and analytics
- [ ] Smart backup cleanup

### Version 1.0.11 (Planned)
- [ ] Watch mode (auto-backup on file change)
- [ ] Cloud backup integration
- [ ] Web UI dashboard
- [ ] Backup search functionality

### Version 1.0.12 (Future)
- [ ] Multi-file operations
- [ ] Backup encryption
- [ ] Remote sync support
- [ ] Team collaboration features

## 📈 Changelog

### Version 1.0.11 (Current)
- ✅ Recursive file search
- ✅ Delta diff integration
- ✅ Directory tree visualization
- ✅ GitIgnore support
- ✅ Safe file deletion
- ✅ Exception filtering for tree view
- ✅ Automatic backup directory (`./backup/`)

### Version 1.0.10
- ✅ Interactive diff comparison
- ✅ Last backup quick access
- ✅ Improved error handling

### Version 1.0.9
- ✅ Production hardening
- ✅ Path validation
- ✅ Size limits
- ✅ Audit logging

### Version 1.0.8
- ✅ Basic clipboard to file
- ✅ Append mode
- ✅ Backup creation
- ✅ Restore functionality

---

**Made with ❤️ by Hadi Cahyadi**

*Your complete file version management system in a single command.* ⚡

---

> 🌟 **PT: More than a clipboard tool — it's your file version manager!** 
> 
> Save, compare, restore, and manage all your file versions effortlessly. Never lose work again!

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