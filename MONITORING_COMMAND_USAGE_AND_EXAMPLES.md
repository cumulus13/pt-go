# Monitor Command - Usage Examples & Features

## Default Behavior - Current Directory

### Monitor all files in current directory
```bash
# Just run monitor without arguments
pt -mt
pt --monitor

# Output:
ℹ️  No pattern specified, monitoring all files in current directory
🔍 Starting monitor...
📁 Root: /home/user/project
🎯 Pattern: *
✅ Monitoring 15 directories
⌨️  Press Ctrl+C to stop
```

### Monitor with specific pattern in current directory
```bash
# Only provide pattern (no path)
pt -mt *.py
pt --monitor *.js

# Output:
🔍 Starting monitor...
📁 Root: /home/user/project  (current directory)
🎯 Pattern: *.py
```

### Monitor with full path
```bash
# Provide full path with pattern
pt -mt /home/user/src/*.go
pt --monitor C:\Projects\MyApp\*.cs

# Output:
🔍 Starting monitor...
📁 Root: /home/user/src
🎯 Pattern: *.go
```

## Critical Directory Exclusions

### Automatic Exclusions

The monitor automatically **excludes and ignores** these directories:

**Critical (Hard-coded protection):**
- `.git` - Git repository data
- `.pt` - Application configuration and data

**Common development directories:**
- `node_modules` - Node.js dependencies
- `__pycache__` - Python cache
- `.vscode` - VS Code settings
- `.idea` - IntelliJ IDEA settings
- `vendor` - Go/PHP dependencies
- `dist` - Distribution builds
- `build` - Build outputs
- `.backups` - Backup files
- `target` - Java/Rust builds
- `bin` - Binary files
- `obj` - Object files

### Why .git and .pt are Excluded

**Safety reasons:**
1. **.git directory**: Contains repository metadata, hundreds/thousands of small files
   - Monitoring would trigger constant events
   - Could corrupt git data
   - Performance nightmare

2. **.pt directory**: Contains application data
   - Config files
   - Backup metadata
   - Logs
   - Monitoring would create recursive loops

### Attempting to Monitor Excluded Directories

```bash
# Try to monitor .git directly
pt -mt /home/user/project/.git/*.pack

# Output:
❌ Error: cannot monitor .git directory

# Try to monitor .pt directly  
pt -mt ~/.pt/*

# Output:
❌ Error: cannot monitor .pt directory
```

### Exclusion in Action

```
project/
├── src/
│   ├── main.go          ✅ Monitored
│   └── utils.go         ✅ Monitored
├── .git/                ❌ EXCLUDED (ignored completely)
│   ├── objects/
│   └── refs/
├── .pt/                 ❌ EXCLUDED (ignored completely)
│   ├── config.yaml
│   └── backups/
├── node_modules/        ❌ EXCLUDED
│   └── express/
└── dist/                ❌ EXCLUDED
    └── bundle.js
```

When monitoring `project/`, only `src/` files will trigger events.

## Real-World Examples

### Example 1: Web Development
```bash
cd /home/user/webapp
pt -mt *.js

# Monitors:
# ✅ app.js
# ✅ routes.js
# ✅ src/components/Header.js

# Ignores:
# ❌ .git/*
# ❌ node_modules/*
# ❌ dist/*
```

### Example 2: Python Project
```bash
cd /home/user/python-project
pt --monitor

# Monitors ALL files in current directory
# Pattern: *

# Output when file changes:
📝 [14:30:45] File modified: /home/user/python-project/main.py
💾 Auto-backup created: main.py

✨ [14:31:20] File created: /home/user/python-project/utils.py
💾 Auto-backup created: utils.py

# Automatically ignores:
# ❌ .git/
# ❌ __pycache__/
# ❌ .pt/
```

### Example 3: Multi-Language Project
```bash
# Monitor specific file types
pt -mt *.go

# Or flexible argument order
pt *.rs -mt

# Or with debug mode
pt -mt *.cpp --debug
```

### Example 4: Configuration Files
```bash
# Monitor config changes
pt -mt *.yaml
pt -mt *.json
pt -mt *.toml

# When config changes:
📝 [15:10:30] File modified: /home/user/project/config.yaml
💾 Auto-backup created: config.yaml
```

## Directory Creation/Deletion Events

### New Directory Created
```bash
# Monitor running: pt -mt *.py

# User creates new directory:
mkdir new_module

# Output:
📁 New directory: /home/user/project/new_module

# Now monitoring:
# ✅ /home/user/project/new_module/*.py
```

### Directory Deleted
```bash
# User deletes directory:
rm -rf old_module

# Output:
📁 Directory removed: /home/user/project/old_module

# No longer monitoring that path
```

### .git Directory Created (Ignored)
```bash
# User initializes git:
git init

# Output: (nothing - completely ignored)

# Log shows:
# Ignoring excluded directory creation: /home/user/project/.git
```

## Performance Considerations

### Good Practices
```bash
# ✅ Monitor specific patterns
pt -mt *.py
pt -mt *.go

# ✅ Monitor specific subdirectory
pt -mt src/*.js

# ✅ Use from project root with pattern
cd /home/user/project
pt -mt *.cpp
```

### Avoid
```bash
# ❌ Don't monitor entire system
pt -mt /home/*/*.py  # Too broad!

# ❌ Don't monitor root
pt -mt /*  # Disaster!

# ❌ Don't use overly generic patterns on large dirs
pt -mt *  # In directory with 10,000+ files
```

## Debugging

### Enable Debug Mode
```bash
pt -mt *.go --debug

# Shows detailed logs:
# Watching: /home/user/project/src
# Watching: /home/user/project/pkg
# Skipping critical directory: /home/user/project/.git
# Skipping directory: /home/user/project/node_modules
# File modified: main.go
```

### Check What's Being Monitored
The monitor shows count on startup:
```
✅ Monitoring 15 directories
```

If this number seems wrong:
1. Check if directories are being excluded
2. Enable `--debug` to see skip messages
3. Verify pattern is correct

## Integration with Auto-Backup

### Enable Auto-Backup
```yaml
# ~/.pt/config.yaml
auto_backup: true
backup_dir_name: ".backups"
```

### Behavior with Monitor
```bash
pt -mt *.py

# When file changes:
📝 [14:30:45] File modified: main.py
💾 Auto-backup created: main.py

# Backup saved to:
# .backups/main.py_20231218_143045.bak
```

### Smart Backup
- Only creates backup if content actually changed
- Debounced (waits 300ms after last change)
- Won't backup files in excluded directories
- Won't trigger on `.bak` files themselves

## Stopping the Monitor

```bash
# Press Ctrl+C to stop

# Output:
^C
🛑 Monitor stopped
```

## Tips & Tricks

### 1. Quick Start in Any Directory
```bash
cd /path/to/project
pt -mt
# Instantly monitoring all files!
```

### 2. Pattern Combinations
```bash
# Monitor multiple patterns (run multiple instances)
pt -mt *.go &
pt -mt *.py &
pt -mt *.js &
```

### 3. Temporary Monitoring
```bash
# Monitor for quick test
pt -mt *.conf
# Make changes, verify notifications
# Ctrl+C to stop
```

### 4. Project-Specific Monitoring
```bash
# In project root
pt -mt src/*.cpp    # Only C++ in src/
pt -mt tests/*.py   # Only Python tests
```

### 5. Notification Verification
```bash
# Test if notifications work
pt -mt test.txt
echo "change" >> test.txt
# Should see notification!
```

## Troubleshooting

### "cannot monitor .git directory"
**Cause**: Trying to monitor .git directly
**Solution**: Monitor parent directory instead:
```bash
# Don't do this:
pt -mt .git/*

# Do this:
cd ..
pt -mt *.go
```

### No Events Firing
1. Check pattern matches: `*.py` not `*.PY`
2. Verify file is not in excluded directory
3. Enable `--debug` to see what's happening
4. Check file is actually changing (not just accessed)

### Too Many Events
1. Pattern too broad - use specific extension
2. Large directory - monitor subdirectory instead
3. Check debounce is working (300ms delay)

### Performance Issues
1. Too many directories being watched
2. Exclude more directories in code
3. Use more specific pattern
4. Monitor smaller subdirectory

## Summary

✅ **Default**: Current directory, all files
✅ **Safe**: Automatically excludes .git and .pt
✅ **Smart**: Skips common development directories
✅ **Flexible**: Pattern or no pattern
✅ **Integrated**: Auto-backup support
✅ **Fast**: Debounced events
✅ **Reliable**: Thread-safe operations