package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/mattn/go-gntp"
)

var (
	debounceTimers = make(map[string]*time.Timer)
	watchedDirs    = make(map[string]bool)
	monitorMu      sync.Mutex
)

// handleMonitorCommand handles file monitoring with pattern matching
func handleMonitorCommand(args []string) error {
	var pattern string
	var root string

	// Default to current directory with all files if no args provided
	if len(args) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		root = cwd
		pattern = "*" // Monitor all files

		fmt.Printf("%sℹ️  No pattern specified, monitoring all files in current directory%s\n", ColorYellow, ColorReset)
	} else {
		argPath := filepath.Clean(args[0])
		pattern = filepath.Base(argPath)
		root = filepath.Dir(argPath)

		// If only filename pattern is provided, use current directory
		if root == "." {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}
			root = cwd
		}
	}

	// Convert to absolute path
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}
	root = absRoot

	// Check if root is .git or .pt directory
	baseName := filepath.Base(root)
	if baseName == ".git" || baseName == ".pt" {
		return fmt.Errorf("cannot monitor %s directory", baseName)
	}

	fmt.Printf("🔍 Starting monitor...\n")
	fmt.Printf("📁 Root: %s\n", root)
	fmt.Printf("🎯 Pattern: %s\n", pattern)

	return startMonitor(root, pattern)
}

func handleMonitorWithInfo(info *CommandInfo) error {
	return handleMonitorCommand(info.Files)
}

// startMonitor initializes and runs the file system watcher
func startMonitor(root string, pattern string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer watcher.Close()

	err = addWatchRecursive(watcher, root)
	if err != nil {
		return fmt.Errorf("failed to setup watches: %w", err)
	}

	if logger != nil {
		logger.Printf(">>> MONITORING: [%s] Pattern: [%s] | Watched: %d dirs", root, pattern, len(watchedDirs))
	}
	fmt.Printf("✅ Monitoring %d directories\n", len(watchedDirs))
	fmt.Printf("⌨️  Press Ctrl+C to stop\n\n")

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			handleMonitorEvent(watcher, event, pattern)

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			if logger != nil {
				logger.Printf("Monitor error: %v", err)
			}
			fmt.Printf("%s⚠️  Warning: %v%s\n", ColorYellow, err, ColorReset)
		}
	}
}

// addWatchRecursive adds all subdirectories to the watcher
func addWatchRecursive(watcher *fsnotify.Watcher, root string) error {
	monitorMu.Lock()
	defer monitorMu.Unlock()

	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if logger != nil {
				logger.Printf("Walk error at %s: %v", path, err)
			}
			return nil
		}

		if info.IsDir() {
			name := info.Name()

			// CRITICAL: Skip .git and .pt directories immediately
			if name == ".git" || name == ".pt" {
				if logger != nil {
					logger.Printf("Skipping critical directory: %s", path)
				}
				return filepath.SkipDir
			}

			// Skip other common directories that shouldn't be monitored
			if name == "Diagnostics" || name == "node_modules" ||
				name == "__pycache__" || name == ".vscode" || name == ".idea" ||
				name == "vendor" || name == "dist" || name == "build" ||
				name == ".backups" || name == "target" || name == "bin" || name == "obj" {
				if logger != nil {
					logger.Printf("Skipping directory: %s", path)
				}
				return filepath.SkipDir
			}

			if watchedDirs[path] {
				return nil
			}

			err := watcher.Add(path)
			if err != nil {
				if logger != nil {
					logger.Printf("Failed to watch %s: %v", path, err)
				}
				return nil
			}

			watchedDirs[path] = true
			if logger != nil {
				logger.Printf("Watching: %s", path)
			}
		}
		return nil
	})
}

// handleMonitorEvent processes file system events
func handleMonitorEvent(watcher *fsnotify.Watcher, event fsnotify.Event, pattern string) {
	// Get the directory name to check for exclusions
	eventDir := filepath.Base(filepath.Dir(event.Name))
	eventBase := filepath.Base(event.Name)

	// CRITICAL: Ignore all events from .git and .pt directories
	if eventBase == ".git" || eventBase == ".pt" || eventDir == ".git" || eventDir == ".pt" {
		return
	}

	// Also check if path contains .git or .pt anywhere
	if containsExcludedDir(event.Name) {
		return
	}

	// Handle directory creation
	if event.Has(fsnotify.Create) {
		info, err := os.Stat(event.Name)
		if err == nil && info.IsDir() {
			dirName := info.Name()

			// Skip if it's .git or .pt directory
			if dirName == ".git" || dirName == ".pt" {
				if logger != nil {
					logger.Printf("Ignoring excluded directory creation: %s", event.Name)
				}
				return
			}

			monitorMu.Lock()

			if !watchedDirs[event.Name] {
				err := watcher.Add(event.Name)
				if err != nil {
					if logger != nil {
						logger.Printf("Failed to watch new dir %s: %v", event.Name, err)
					}
					monitorMu.Unlock()
					return
				}
				watchedDirs[event.Name] = true
				if logger != nil {
					logger.Printf("New directory watched: %s", event.Name)
				}
				fmt.Printf("📁 New directory: %s\n", event.Name)

				// Watch subdirectories
				filepath.Walk(event.Name, func(path string, info os.FileInfo, err error) error {
					if err != nil || path == event.Name {
						return nil
					}

					if info.IsDir() {
						subDirName := info.Name()

						// Skip .git and .pt subdirectories
						if subDirName == ".git" || subDirName == ".pt" {
							return filepath.SkipDir
						}

						if !watchedDirs[path] {
							err := watcher.Add(path)
							if err != nil {
								if logger != nil {
									logger.Printf("Failed to watch subdirectory %s: %v", path, err)
								}
							} else {
								watchedDirs[path] = true
								if logger != nil {
									logger.Printf("Subdirectory watched: %s", path)
								}
							}
						}
					}
					return nil
				})
			}

			monitorMu.Unlock()
		}
	}

	// Handle directory removal/rename
	if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
		monitorMu.Lock()
		if watchedDirs[event.Name] {
			delete(watchedDirs, event.Name)
			watcher.Remove(event.Name)
			if logger != nil {
				logger.Printf("Directory removed from watch: %s", event.Name)
			}
			fmt.Printf("📁 Directory removed: %s\n", event.Name)
		}
		monitorMu.Unlock()
	}

	// Handle file changes
	matched, _ := filepath.Match(pattern, filepath.Base(event.Name))
	if matched {
		if event.Has(fsnotify.Write) {
			triggerFileAction(event.Name, "modified")
		} else if event.Has(fsnotify.Create) {
			triggerFileAction(event.Name, "created")
		} else if event.Has(fsnotify.Remove) {
			fmt.Printf("🗑️  File deleted: %s\n", event.Name)
			if logger != nil {
				logger.Printf("File deleted: %s", event.Name)
			}
		}
	}
}

// containsExcludedDir checks if path contains .git or .pt directory
func containsExcludedDir(path string) bool {
	// Normalize path separators
	normalizedPath := filepath.ToSlash(path)
	
	// Check if path contains /.git/ or /.pt/
	if strings.Contains(normalizedPath, "/.git/") || 
	   strings.Contains(normalizedPath, "/.pt/") {
		return true
	}
	
	// Check if path ends with /.git or /.pt
	if strings.HasSuffix(normalizedPath, "/.git") || 
	   strings.HasSuffix(normalizedPath, "/.pt") {
		return true
	}
	
	// For Windows paths, also check with backslash
	if strings.Contains(path, string(filepath.Separator)+".git"+string(filepath.Separator)) ||
	   strings.Contains(path, string(filepath.Separator)+".pt"+string(filepath.Separator)) {
		return true
	}
	
	return false
}

// triggerFileAction handles debounced file change actions
func triggerFileAction(path string, action string) {
	monitorMu.Lock()
	defer monitorMu.Unlock()

	if timer, ok := debounceTimers[path]; ok {
		timer.Stop()
	}

	debounceTimers[path] = time.AfterFunc(300*time.Millisecond, func() {
		absPath, _ := filepath.Abs(path)
		timestamp := time.Now().Format("15:04:05")

		// Console output
		actionEmoji := "📝"
		if action == "created" {
			actionEmoji = "✨"
		}
		fmt.Printf("%s [%s] File %s: %s\n", actionEmoji, timestamp, action, absPath)
		if logger != nil {
			logger.Printf("File %s: %s", action, absPath)
		}

		// Send notification if available
		sendFileNotification(path, action, timestamp)

		// Auto-backup if configured (DEFAULT: enabled)
		// NOTE: After adding AutoBackup field to Config struct in main.go,
		// this will work automatically. Default is TRUE.
		if appConfig != nil && appConfig.AutoBackup {
			comment := fmt.Sprintf("Auto-backup on %s", action)
			err := autoBackupFile(absPath, comment)
			if err != nil {
				if logger != nil {
					logger.Printf("Auto-backup failed: %v", err)
				}
			} else {
				fmt.Printf("💾 Auto-backup created: %s\n", filepath.Base(absPath))
			}
		}
	})
}

// autoBackupFile creates a backup when file changes
func autoBackupFile(filePath string, comment string) error {
	// Check if file exists
	if _, err := os.Stat(filePath); err != nil {
		return err
	}

	// Get existing backups
	backups, err := listBackups(filePath)
	if err != nil {
		// If error getting backups, proceed with backup anyway
		if logger != nil {
			logger.Printf("Warning: could not list backups: %v", err)
		}
	}

	// Check if content is different from last backup
	if len(backups) > 0 {
		// Use checkIfDifferent with file paths (it handles file reading internally)
		if !checkIfDifferent(filePath, backups[0].Path) {
			// Content is identical, skip backup
			return nil
		}
	}

	// Create backup using autoRenameIfExists
	// This returns (string, error) not just error
	_, err = autoRenameIfExists(filePath, comment)
	return err
}

// sendFileNotification sends notification via Growl/GNTP
func sendFileNotification(path string, action string, timestamp string) {
	absPath, _ := filepath.Abs(path)
	title := "File Monitor - pt"
	message := fmt.Sprintf("[%s] File %s\n%s", timestamp, action, absPath)

	icon := findNotificationIcon()

	client := gntp.NewClient()
	client.AppName = "pt"

	// Register notification types
	events := []gntp.Notification{
		{Event: "file_changed", Enabled: true},
		{Event: "file_created", Enabled: true},
		{Event: "error", Enabled: true},
	}
	err := client.Register(events)
	if err != nil {
		if logger != nil {
			logger.Printf("Failed to register notifications: %v", err)
		}
		return
	}

	eventType := "file_changed"
	if action == "created" {
		eventType = "file_created"
	}

	msg := &gntp.Message{
		Event:  eventType,
		Title:  title,
		Text:   message,
		Sticky: false,
	}

	if icon != "" {
		if _, err := os.Stat(icon); err == nil {
			msg.Icon = icon
		}
	}

	err = client.Notify(msg)
	if err != nil {
		if logger != nil {
			logger.Printf("Failed to send notification: %v", err)
		}
	}
}

// findNotificationIcon finds the icon file for notifications
func findNotificationIcon() string {
	iconNames := []string{"filemonitor.png", "pt.png", "icon.png"}

	// Try current working directory
	cwd, err := os.Getwd()
	if err == nil {
		for _, iconName := range iconNames {
			candidate := filepath.Join(cwd, iconName)
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
	}

	// Try executable directory
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		for _, iconName := range iconNames {
			candidate := filepath.Join(exeDir, iconName)
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
	}

	return ""
}