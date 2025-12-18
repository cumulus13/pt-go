# File Monitor Feature

## Overview
File monitor memungkinkan Anda untuk memantau perubahan file secara real-time berdasarkan pattern tertentu.

## Usage

### Basic Command
```bash
pt -mt <pattern>
pt --monitor <pattern>
```

### Examples

**Monitor Python files in current directory:**
```bash
pt -mt *.py
```

**Monitor Go files in specific directory:**
```bash
pt -mt C:\TEMP\*.go
pt -mt /home/user/src/*.go
```

**Monitor JavaScript files with flexible argument order:**
```bash
pt *.js -mt
pt -mt *.js --debug
```

**Monitor TypeScript files recursively:**
```bash
pt --monitor *.ts
```

## Features

### 1. Recursive Directory Watching
- Automatically watches all subdirectories
- Dynamically adds newly created directories
- Removes deleted directories from watch list

### 2. Smart Directory Filtering
Automatically skips common directories that shouldn't be monitored:
- `.git`
- `node_modules`
- `__pycache__`
- `.vscode`
- `.idea`
- `vendor`
- `dist`
- `build`
- `Diagnostics`

### 3. File Change Detection
Monitors the following events:
- **File Modified** (📝) - When file content changes
- **File Created** (✨) - When new file is created
- **File Deleted** (🗑️) - When file is removed

### 4. Debouncing
Changes are debounced with 300ms delay to avoid multiple triggers for the same modification.

### 5. Notifications
- Desktop notifications via Growl/GNTP
- Displays timestamp and full file path
- Custom icons support

### 6. Auto-Backup (Optional)
When enabled in config, automatically creates backups when files change:
```yaml
auto_backup: true
```

## Output Example

```
🔍 Starting monitor...
📁 Root: /home/user/project
🎯 Pattern: *.py
✅ Monitoring 15 directories
⌨️  Press Ctrl+C to stop

📝 [14:30:45] File modified: /home/user/project/main.py
💾 Auto-backup created: /home/user/project/main.py

✨ [14:31:20] File created: /home/user/project/utils.py
💾 Auto-backup created: /home/user/project/utils.py

📁 New directory: /home/user/project/tests

🗑️ File deleted: /home/user/project/temp.py
```

## Integration with Other Commands

Monitor dapat dikombinasikan dengan command lain:

```bash
# Monitor with debug mode
pt -mt *.go --debug

# Monitor specific pattern
pt --monitor "test_*.py"

# Monitor with custom tool
pt -mt *.js --tool vimdiff
```

## Configuration

Add to your config file (`~/.pt/config.yaml`):

```yaml
# Enable auto-backup on file changes
auto_backup: true

# Custom backup directory
backup_dir_name: ".backups"

# Notification settings
notification:
  enabled: true
  icon: "/path/to/icon.png"
```

## Requirements

### Dependencies
```bash
go get github.com/fsnotify/fsnotify
go get github.com/mattn/go-gntp
```

### Notification Server
For desktop notifications, you need Growl or GNTP-compatible server:
- **Windows**: Growl for Windows
- **macOS**: Growl or Mountain Growl
- **Linux**: gntpd or similar GNTP daemon

## Technical Details

### File Structure
```
project/
├── main.go           # Main entry point with command routing
├── monitor.go        # File monitor implementation
├── config.go         # Configuration management
└── utils.go          # Helper functions
```

### How It Works

1. **Initialization**: Parser identifies `-mt` or `--monitor` command
2. **Pattern Matching**: Extracts file pattern and root directory
3. **Watcher Setup**: Recursively adds all directories to fsnotify watcher
4. **Event Loop**: Continuously monitors file system events
5. **Event Handling**: Filters events by pattern and triggers actions
6. **Debouncing**: Delays action by 300ms to batch rapid changes
7. **Notification**: Sends desktop notification and logs event
8. **Auto-backup**: Optionally creates backup if enabled

### Thread Safety
Uses `sync.Mutex` to safely handle concurrent access to:
- `watchedDirs` map
- `debounceTimers` map

## Tips

1. **Performance**: Avoid monitoring large directories with many subdirectories
2. **Patterns**: Use specific patterns to reduce false positives
3. **Debugging**: Use `--debug` flag to see detailed logs
4. **Icons**: Place `filemonitor.png` or `pt.png` in executable directory for custom icons

## Troubleshooting

### Monitor not detecting changes
- Ensure pattern matches file extension exactly
- Check if directory is in skip list
- Verify file system supports fsnotify events

### Too many notifications
- Pattern may be too broad
- Increase debounce timer in code (default: 300ms)

### Auto-backup not working
- Ensure `auto_backup: true` in config
- Check backup directory permissions
- Verify file content actually changed

## Future Enhancements

- [ ] Configurable debounce timer
- [ ] Multiple pattern support
- [ ] Custom actions/scripts on file change
- [ ] Email notifications
- [ ] Webhook integration
- [ ] Exclude patterns
- [ ] Maximum backup limit per file