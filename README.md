# 📋 `pt` – Clipboard to File Tool, Paste to File with Smart Backups

[![Go Version](https://img.shields.io/badge/Go-1.16+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Version](https://img.shields.io/badge/version-1.0.5-blue.svg)](https://github.com/cumulus13/pt)

> **`pt`**, A tiny but powerful CLI tool that writes your clipboard content directly to a file — with automatic timestamped backups, append mode, restore functionality, and beautiful backup listings. Perfect for quick notes, code snippets, logs, or any text you want to save safely without overwriting.

## ✨ Features

- 📝 **Quick Save** - Write clipboard content to file with one command
- 📦 **Auto Backup** - Automatic timestamped backups before overwriting
- ➕ **Append Mode** - Add content without creating backups
- 🔄 **Restore** - Interactive or quick restore from backups
- 📊 **Beautiful Listings** - Formatted table view of all backups
- 🔒 **Production Hardened** - Path validation, size limits, error handling
- 🎨 **Colorful Output** - ANSI colors for better readability
- 📈 **Audit Logging** - All operations logged for tracking

## 🚀 Installation

### Prerequisites

- Go 1.16 or higher
- Git (for installation)

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

### Verify Installation

```bash
pt --version
```

## 📖 Usage

### Basic Commands

```bash
# Write clipboard to file (creates backup if exists)
pt myfile.txt

# Append clipboard to file (no backup)
pt + myfile.txt

# List all backups
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

### 3. Code Snippet Management

```bash
# Copy code from browser
pt snippet.py

# Later, copy another snippet
pt snippet.py
# 📦 Backup created: snippet_py.20251102_151804177132.12345_a1b2c3d4

# List all versions
pt -l snippet.py
```

### 4. Restore Previous Version

```bash
# Interactive restore
pt -r notes.txt
# Shows table with all backups
# Enter backup number to restore (1-10) or 0 to cancel: 3
# ✅ Successfully restored: notes.txt

# Quick restore last backup
pt -r notes.txt --last
# ✅ Successfully restored: notes.txt
```

## 🎨 Output Examples

### Backup Listing

```
📂 Backup files for 'myfile.txt'
Total: 5 backup(s)

┌────────────────────────────────────────────────────┬─────────────────────┬─────────────────┐
│ File Name                                          │ Modified            │ Size            │
├────────────────────────────────────────────────────┼─────────────────────┼─────────────────┤
│  1. myfile_txt.20251102_151804177132.12345_a1b2   │ 2025-11-02 15:18:04 │       2.45 KB   │
│  2. myfile_txt.20251102_143022123456.12344_b2c3   │ 2025-11-02 14:30:22 │       2.40 KB   │
│  3. myfile_txt.20251102_120000000000.12343_c3d4   │ 2025-11-02 12:00:00 │       1.98 KB   │
│  4. myfile_txt.20251101_180000000000.12342_d4e5   │ 2025-11-01 18:00:00 │       1.85 KB   │
│  5. myfile_txt.20251101_100000000000.12341_e5f6   │ 2025-11-01 10:00:00 │       1.52 KB   │
└────────────────────────────────────────────────────┴─────────────────────┴─────────────────┘
```

## 🏗️ Project Structure

```
pt/
├── go.mod                          # Go module definition
├── go.sum                          # Dependency checksums
├── pt/
│   ├── main.go                     # Main application code
│   ├── main_go.20251030_163913... # Auto-created backups (examples)
│   └── main_go.20251102_151804...
├── README.md                       # This file
├── LICENSE                         # MIT License
├── .gitignore                      # Git ignore rules
└── install.sh                      # Installation script (optional)
```

## 🔧 Configuration

### Environment Variables

Currently, PT uses hardcoded limits for safety:

- **Max Clipboard Size**: 100MB
- **Max Backup Count**: 100 backups
- **Max Filename Length**: 200 characters

To customize, edit the constants in `pt/main.go`:

```go
const (
    MaxClipboardSize = 100 * 1024 * 1024 // 100MB max
    MaxBackupCount   = 100                // Keep max 100 backups
    MaxFilenameLen   = 200                // Max filename length
)
```

### Backup Naming Format

Backups use this format:
```
originalname_ext.YYYYMMDD_HHMMSS_MICROSECONDS.PID_RANDOMID
```

Example:
```
notes_txt.20251102_151804177132.12345_a1b2c3d4
```

Where:
- `notes_txt` - Original filename without extension
- `20251102_151804177132` - Timestamp (microsecond precision)
- `12345` - Process ID
- `a1b2c3d4` - Random 8-char hex ID

This ensures **zero collision** risk even with concurrent operations.

## 🔒 Security Features

### Path Validation
- Prevents path traversal attacks (`../../../etc/passwd`)
- Blocks writes to system directories (`/etc`, `/sys`, `C:\Windows`)
- Validates filename length limits

### Size Limits
- Maximum 100MB clipboard content
- Prevents disk exhaustion attacks
- Validates disk space before writing

### Input Validation
- All user inputs sanitized
- Numeric inputs validated for range
- Graceful handling of malformed input

### Safe Operations
- Atomic-like file operations
- Verification of write completion
- Automatic rollback on errors

## ⚠️ Limitations

1. **Text Only** - Only supports text content (no binary files from clipboard)
2. **Single File** - Operates on one file at a time
3. **Local Only** - No network or cloud storage support
4. **Platform Support** - Requires clipboard access (may need X11 on Linux headless)

## 🐛 Troubleshooting

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

## 🧪 Testing

### Manual Testing

```bash
# Test basic write
echo "Hello World" | pbcopy  # macOS
pt test.txt

# Test append
echo "Line 2" | pbcopy
pt + test.txt

# Test backup listing
pt -l test.txt

# Test restore
pt -r test.txt --last
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

## 👨‍💻 Author

[**Hadi Cahyadi**](mailto:cumulus13@gmail.com)

- GitHub: [@cumulus13](https://github.com/cumulus13)
- Email: cumulus13@gmail.com

[![Buy Me a Coffee](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/cumulus13)

[![Donate via Ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/cumulus13)
 
[Support me on Patreon](https://www.patreon.com/cumulus13)


## 🙏 Acknowledgments

- [atotto/clipboard](https://github.com/atotto/clipboard) - Cross-platform clipboard library
- Go community for excellent tooling and documentation
- All contributors and users

## 📞 Support

- 🐛 **Bug Reports**: [GitHub Issues](https://github.com/cumulus13/pt/issues)
- 💡 **Feature Requests**: [GitHub Discussions](https://github.com/cumulus13/pt/discussions)
- 📧 **Email**: cumulus13@gmail.com

## ⭐ Star History

If you find PT useful, please consider giving it a star on GitHub! ⭐

---

**Made with ❤️ by Hadi Cahyadi**

*Save your clipboard, save your time.* ⚡
---

> 🌟 **Enjoy!** Save your clipboard safely, every time.