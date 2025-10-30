# 📋 `pt` – Paste to File with Smart Backups

> **`pt`** is a tiny but powerful CLI tool that writes your clipboard content directly to a file — with automatic timestamped backups, append mode, and beautiful backup listings. Perfect for quick notes, code snippets, logs, or any text you want to save safely without overwriting.

---

## ✨ Features

- **Write clipboard to file**: `pt notes.txt`
- **Append clipboard to file**: `pt + notes.txt`
- **Auto-backup**: If the file exists, it’s renamed with a precise timestamp before overwriting.
- **List backups**: `pt -l notes.txt` shows all backups in a clean, colorized table.
- **Human-readable sizes**: KB, MB, or bytes.
- **Cross-platform**: Works on Windows, macOS, and Linux.
- **Zero dependencies** (except clipboard access).

---

## 🚀 Installation

### Prerequisites
- [Go](https://golang.org/dl/) 1.16 or higher (to build from source)
- A working clipboard (standard on all modern OSes)

### From Source (Recommended)

```bash
go install github.com/cumulus13/pt-go/pt@latest
```

This installs the `pt` binary to your `$GOPATH/bin` (make sure it’s in your `PATH`).

### Build Manually

```bash
git clone https://github.com/cumulus13/pt-go.git
cd pt-go
go build -o pt .
sudo mv pt /usr/local/bin/  # or add to your PATH
```

---

## 📚 Usage

### Write clipboard to a file (overwrite)
```bash
pt myfile.txt
```
- If `myfile.txt` exists → backed up as `myfile_txt.YYYYMMDD_HHMMSSffffff`
- Clipboard content written to `myfile.txt`

### Append clipboard to a file
```bash
pt + myfile.txt
```
- Content is **appended** (no backup created)
- File created if it doesn’t exist

### List all backups for a file
```bash
pt -l myfile.txt
# or
pt --list myfile.txt
```
Shows a table like:

```
📂 Backup files for 'myfile.txt'

┌────────────────────────────────────────────────────┬─────────────────────┬──────────────┐
│ File Name                                          │ Modified            │ Size         │
├────────────────────────────────────────────────────┼─────────────────────┼──────────────┤
│ myfile_txt.20251030_164050455738                   │ 2025-10-30 16:40:50 │ 599 B        │
└────────────────────────────────────────────────────┴─────────────────────┴──────────────┘
```

---

## 🔒 Backup Naming Convention

When `example.txt` is overwritten, it becomes:

```
example_txt.20251030_164050455738
```

- `example` → base name  
- `txt` → extension (without dot)  
- `20251030_164050455738` → timestamp: `YYYYMMDD_HHMMSSffffff` (microsecond precision)

> This ensures **no collisions** and **chronological sorting**.

---

## 🛠️ Requirements

- Go 1.16+
- `github.com/atotto/clipboard` (for cross-platform clipboard access)

> All dependencies are automatically fetched during `go install`.

---

## 📦 License

MIT License – see [LICENSE](LICENSE) for details.

> Free to use, modify, and distribute — even commercially.

---

## 💡 Tips & Tricks

- Use with shell aliases:
  ```bash
  alias ptn='pt notes.md'        # quick note
  alias pt+='pt +'               # append shorthand
  ```
- Combine with `fzf` or `bat` for advanced workflows.
- Great for saving terminal output: copy → `pt debug.log`

---

## 🐞 Known Issues & Limitations

- Does **not** handle binary clipboard content (text only).
- Backup listing only works in the **same directory** as the original file.
- On some Linux systems, you may need `xclip` or `xsel` installed for clipboard support.

---

## 🙌 Contributing

PRs welcome! Suggestions, bug reports, and feature ideas are appreciated.

1. Fork the repo
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

---

## 📧 Author

[**Hadi Cahyadi**](mailto:cumulus13@gmail.com)
    

[![Buy Me a Coffee](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/cumulus13)

[![Donate via Ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/cumulus13)
 
[Support me on Patreon](https://www.patreon.com/cumulus13)

---

> 🌟 **Enjoy!** Save your clipboard safely, every time.