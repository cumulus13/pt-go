package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/getlantern/systray"
	"github.com/mattn/go-gntp"
)

var (
	debounceTimers = make(map[string]*time.Timer)
	watchedDirs    = make(map[string]bool)
	watchedFiles   = make(map[string]bool)
	monitorMu      sync.Mutex
	
	monitorPaused  = false
	monitorRunning = false
	stopMonitorCh  = make(chan bool)
	
	menuStart      *systray.MenuItem
	menuStop       *systray.MenuItem
	menuPause      *systray.MenuItem
	menuResume     *systray.MenuItem
	menuTextNotif  *systray.MenuItem
	menuQuit       *systray.MenuItem
	
	savedArgs       []string
	savedExceptions []string  // Store exceptions for restart
)

func checkDebug() bool {
	if os.Getenv("DEBUG") == "1" {
		return true
	}

	return false
}

var DEBUG = checkDebug()

func containsString(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}

func removeArg(s []string, value string) []string {
    result := []string{}
    for _, v := range s {
        if v != value {
            result = append(result, v)
        }
    }
    return result
}

func handleMonitorCommand(args []string) error {
	// savedArgs = args

	if (containsString(os.Args, "-e") && !containsString(args, "-e")) || (containsString(os.Args, "--exception") && !containsString(args, "-e")) {
		args = os.Args[2:]
	}
	// Parse exception flags
	var exceptions []string
	var paths []string
	
	if DEBUG { fmt.Printf("args: %v\n", args) }

	for i := 0; i < len(args); i++ {
		if (args[i] == "-e" || args[i] == "--exception") && i+1 < len(args) {
			// Next arg is the exception pattern
			next_arg := args[i+1]
			if DEBUG { fmt.Printf("next_arg: %s", next_arg)}
			if string(next_arg[0]) != "-"  && next_arg != "-e" && next_arg != "--exception" {
					exceptions = append(exceptions, args[i+1])
				}
				i++ // Skip next arg with '-'
			
		} else {
			paths = append(paths, args[i])
		}
	}

	if DEBUG {
		fmt.Printf("exceptions: %v\n", exceptions)
		fmt.Printf("paths: %v\n", paths)
	}

	if len(paths) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		paths = []string{cwd}
		fmt.Printf("%sℹ️  No files specified, monitoring current directory%s\n", ColorYellow, ColorReset)
	}

	// Display exceptions if any
	if len(exceptions) > 0 {
		fmt.Printf("%sℹ️  Exceptions: %v%s\n", ColorYellow, exceptions, ColorReset)
	}

	var expandedPaths []string
	for _, pattern := range paths {
		if strings.ContainsAny(pattern, "*?[]") {
			matches, err := filepath.Glob(pattern)
			if err == nil && len(matches) > 0 {
				expandedPaths = append(expandedPaths, matches...)
				fmt.Printf("%sℹ️  Pattern '%s' matched %d file(s)%s\n", ColorCyan, pattern, len(matches), ColorReset)
			} else {
				dir := filepath.Dir(pattern)
				if dir != "." && dir != "" {
					absDir, err := filepath.Abs(dir)
					if err == nil {
						if info, err := os.Stat(absDir); err == nil && info.IsDir() {
							expandedPaths = append(expandedPaths, absDir)
							fmt.Printf("%sℹ️  Pattern '%s' → monitoring directory: %s%s\n", ColorCyan, pattern, absDir, ColorReset)
							continue
						}
					}
				}
				cwd, err := os.Getwd()
				if err == nil {
					expandedPaths = append(expandedPaths, cwd)
					fmt.Printf("%sℹ️  Pattern '%s' → monitoring current directory%s\n", ColorCyan, pattern, ColorReset)
				}
			}
		} else {
			expandedPaths = append(expandedPaths, pattern)
		}
	}

	if len(expandedPaths) == 0 {
		return fmt.Errorf("no valid paths to monitor")
	}

	fmt.Printf("\n🔍 Starting monitor...\n")
	fmt.Printf("📁 Monitoring %d path(s):\n", len(expandedPaths))
	for i, path := range expandedPaths {
		absPath, _ := filepath.Abs(path)
		fmt.Printf("   %d. %s\n", i+1, absPath)
	}

	go systray.Run(onReady, onExit)

	return startMonitorMultiple(expandedPaths, exceptions)
}

func handleMonitorWithInfo(info *CommandInfo) error {
	return handleMonitorCommand(info.Files)
}

func startMonitorMultiple(paths []string, exceptions []string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer watcher.Close()

	monitorRunning = true
	defer func() { monitorRunning = false }()
	
	// Save exceptions for restart
	savedExceptions = exceptions

	for _, path := range paths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			fmt.Printf("%s⚠️  Warning: failed to resolve %s: %v%s\n", ColorYellow, path, err, ColorReset)
			continue
		}

		info, err := os.Stat(absPath)
		if err != nil {
			fmt.Printf("%s⚠️  Warning: path not found %s%s\n", ColorYellow, absPath, ColorReset)
			continue
		}

		baseName := filepath.Base(absPath)
		if baseName == ".git" || baseName == ".pt" {
			fmt.Printf("%s⚠️  Skipping excluded directory: %s%s\n", ColorYellow, absPath, ColorReset)
			continue
		}
		
		// Check if path matches any exception pattern
		if matchesException(absPath, exceptions) {
			fmt.Printf("%s⚠️  Skipping exception: %s%s\n", ColorYellow, absPath, ColorReset)
			continue
		}

		if info.IsDir() {
			err = addWatchRecursive(watcher, absPath, exceptions)
			if err != nil {
				fmt.Printf("%s⚠️  Warning: failed to watch directory %s: %v%s\n", ColorYellow, absPath, err, ColorReset)
			}
		} else {
			parentDir := filepath.Dir(absPath)
			err = watcher.Add(parentDir)
			if err != nil {
				fmt.Printf("%s⚠️  Warning: failed to watch file %s: %v%s\n", ColorYellow, absPath, err, ColorReset)
			} else {
				monitorMu.Lock()
				watchedDirs[parentDir] = true
				watchedFiles[absPath] = true
				monitorMu.Unlock()
				if logger != nil {
					logger.Printf("Watching file: %s", absPath)
				}
			}
		}
	}

	if logger != nil {
		logger.Printf(">>> MONITORING: %d dirs, %d files | Exceptions: %v", len(watchedDirs), len(watchedFiles), exceptions)
	}
	fmt.Printf("\n✅ Monitoring %d directories and %d specific files\n", len(watchedDirs), len(watchedFiles))
	if len(exceptions) > 0 {
		fmt.Printf("🚫 Excluding patterns: %v\n", exceptions)
	}
	fmt.Printf("⌨️  Press Ctrl+C to stop or use system tray menu\n\n")

	for {
		select {
		case <-stopMonitorCh:
			fmt.Println("🛑 Monitoring stopped by system tray")
			return nil

		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if !monitorPaused {
				handleMonitorEventMultiple(watcher, event, paths, exceptions)
			}

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

func addWatchRecursive(watcher *fsnotify.Watcher, root string, exceptions []string) error {
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

			if name == ".git" || name == ".pt" {
				if logger != nil {
					logger.Printf("Skipping critical directory: %s", path)
				}
				return filepath.SkipDir
			}

			if name == "Diagnostics" || name == "node_modules" ||
				name == "__pycache__" || name == ".vscode" || name == ".idea" ||
				name == "vendor" || name == "dist" || name == "build" ||
				name == ".backups" || name == "target" || name == "bin" || name == "obj" {
				if logger != nil {
					logger.Printf("Skipping directory: %s", path)
				}
				return filepath.SkipDir
			}
			
			// Check if directory matches exception pattern
			if matchesException(path, exceptions) {
				if logger != nil {
					logger.Printf("Skipping exception directory: %s", path)
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

func handleMonitorEventMultiple(watcher *fsnotify.Watcher, event fsnotify.Event, monitoredPaths []string, exceptions []string) {
	eventDir := filepath.Base(filepath.Dir(event.Name))
	eventBase := filepath.Base(event.Name)

	if eventBase == ".git" || eventBase == ".pt" || eventDir == ".git" || eventDir == ".pt" {
		return
	}

	if containsExcludedDir(event.Name) {
		return
	}
	
	// Check if event matches exception pattern
	if matchesException(event.Name, exceptions) {
		return
	}

	absEvent, _ := filepath.Abs(event.Name)
	isMonitored := false

	monitorMu.Lock()
	if watchedFiles[absEvent] {
		isMonitored = true
	} else {
		for _, path := range monitoredPaths {
			absPath, _ := filepath.Abs(path)
			
			if strings.HasPrefix(absEvent, absPath) {
				isMonitored = true
				break
			}
			
			if absEvent == absPath {
				isMonitored = true
				break
			}
		}
	}
	monitorMu.Unlock()

	if !isMonitored {
		return
	}

	if event.Has(fsnotify.Create) {
		info, err := os.Stat(event.Name)
		if err == nil && info.IsDir() {
			dirName := info.Name()

			if dirName == ".git" || dirName == ".pt" {
				if logger != nil {
					logger.Printf("Ignoring excluded directory creation: %s", event.Name)
				}
				return
			}
			
			// Check exception for new directory
			if matchesException(event.Name, exceptions) {
				if logger != nil {
					logger.Printf("Ignoring exception directory creation: %s", event.Name)
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

				filepath.Walk(event.Name, func(path string, info os.FileInfo, err error) error {
					if err != nil || path == event.Name {
						return nil
					}

					if info.IsDir() {
						subDirName := info.Name()

						if subDirName == ".git" || subDirName == ".pt" {
							return filepath.SkipDir
						}
						
						if matchesException(path, exceptions) {
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
		if watchedFiles[event.Name] {
			delete(watchedFiles, event.Name)
			if logger != nil {
				logger.Printf("File removed from watch: %s", event.Name)
			}
		}
		monitorMu.Unlock()
	}

	if event.Has(fsnotify.Write) {
		triggerFileAction(event.Name, "modified")
	} else if event.Has(fsnotify.Create) {
		info, err := os.Stat(event.Name)
		if err == nil && !info.IsDir() {
			triggerFileAction(event.Name, "created")
		}
	} else if event.Has(fsnotify.Remove) {
		info, _ := os.Stat(event.Name)
		if info == nil || !info.IsDir() {
			fmt.Printf("🗑️  File deleted: %s\n", event.Name)
			if logger != nil {
				logger.Printf("File deleted: %s", event.Name)
			}
		}
	}
}

// matchesException checks if path matches any exception pattern
func matchesException(path string, exceptions []string) bool {
	if len(exceptions) == 0 {
		return false
	}
	
	for _, pattern := range exceptions {
		// Check exact match
		if filepath.Base(path) == pattern {
			return true
		}
		
		// Check wildcard match
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err == nil && matched {
			return true
		}
		
		// Check if path contains pattern
		if strings.Contains(filepath.ToSlash(path), pattern) {
			return true
		}
	}
	
	return false
}

func containsExcludedDir(path string) bool {
	normalizedPath := filepath.ToSlash(path)
	
	if strings.Contains(normalizedPath, "/.git/") || 
	   strings.Contains(normalizedPath, "/.pt/") {
		return true
	}
	
	if strings.HasSuffix(normalizedPath, "/.git") || 
	   strings.HasSuffix(normalizedPath, "/.pt") {
		return true
	}
	
	if strings.Contains(path, string(filepath.Separator)+".git"+string(filepath.Separator)) ||
	   strings.Contains(path, string(filepath.Separator)+".pt"+string(filepath.Separator)) {
		return true
	}
	
	return false
}

func triggerFileAction(path string, action string) {
	monitorMu.Lock()
	defer monitorMu.Unlock()

	if timer, ok := debounceTimers[path]; ok {
		timer.Stop()
	}

	debounceTimers[path] = time.AfterFunc(300*time.Millisecond, func() {
		absPath, _ := filepath.Abs(path)
		timestamp := time.Now().Format("15:04:05")

		actionEmoji := "📝"
		if action == "created" {
			actionEmoji = "✨"
		}
		fmt.Printf("%s [%s] File %s: %s\n", actionEmoji, timestamp, action, absPath)
		if logger != nil {
			logger.Printf("File %s: %s", action, absPath)
		}

		sendFileNotification(path, action, timestamp)

		if appConfig.AutoBackup == nil || *appConfig.AutoBackup {
			comment := ""
			status, err := autoBackupFile(absPath, comment)
			if err != nil {
				if logger != nil {
					logger.Printf("Auto-backup failed: %v", err)
				}
			} else {
				if status != "identical" {
					fmt.Printf("💾 Auto-backup created: %s\n", filepath.Base(absPath))
				}
			}
		}
	})
}

func autoBackupFile(filePath string, comment string) (string, error) {
	backups, err := listBackups(filePath)
	if err != nil {
		fmt.Printf("%s❌ Error autoBackupFile [1]: %v%s\n", ColorRed, err, ColorReset)
		sendFileNotification(filePath, "error", time.Now().Format("15:04:05"), err)
		return "", err
	}

	if !isFile(filePath) {
		return "", fmt.Errorf("%s not a file", filePath)
	}
	text, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("%s❌ Error autoBackupFile [2]: %v%s\n", ColorRed, err, ColorReset)
		sendFileNotification(filePath, "error", time.Now().Format("15:04:05"), err)
		return "", err
	}

	if len(backups) == 0 {
		fmt.Printf("No backups found for: %s (check %s/ directory)\n", filePath, appConfig.BackupDirName)
		_, err = autoRenameIfExists(filePath, comment, false)
		if err != nil {
			fmt.Printf("%s❌ Error autoBackupFile [3]: %v%s\n", ColorRed, err, ColorReset)
			return "", err
		}
	} else {
		selectedBackup := backups[0]
		fmt.Printf("%s📊 Comparing with last backup: %s%s\n\n", ColorCyan, selectedBackup.Name, ColorReset)

		if !checkIfDifferent(selectedBackup.Path, text) {
			fmt.Printf(" ⚠ %sLast backup:%s %s%s%s%s %sand%s %s'content'%s %sis%s %s%sidentical%s\n", ColorYellow, ColorReset, ColorWhite, ColorBlue, selectedBackup.Name, ColorReset, ColorYellow, ColorReset, ColorCyan, ColorReset, ColorYellow, ColorReset, ColorWhite, BgMagenta, ColorReset)
			return "identical", nil
		}

		_, err = autoRenameIfExists(filePath, comment, false)
		if err != nil {
			fmt.Printf("%s❌ Error autoBackupFile [4]: %v%s\n", ColorRed, err, ColorReset)
			return "", err
		}
	}

	return "", nil
}

func sendFileNotification(path string, action string, timestamp string, optionalErr ...error) {
	absPath, _ := filepath.Abs(path)
	title := "File Monitor - pt"
	message := fmt.Sprintf("[%s] File %s\n%s", timestamp, action, absPath)

	icon := findNotificationIcon()

	client := gntp.NewClient()
	client.AppName = "pt"

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

	if len(optionalErr) > 0 && optionalErr[0] != nil {
		client.Notify(&gntp.Message{
			Event:  "error",
			Title:  title,
			Text:   fmt.Sprintf("pt monitoring Error: %v", optionalErr[0]),
			Sticky: true,
		})
	}
}

func findNotificationIcon() string {
	iconNames := []string{
		"pt.ico",
		"pt.png",
		"filemonitor.ico",
		"filemonitor.png",
		"icon.ico",
		"icon.png",
		"tray.ico",
		"tray.png",
	}

	cwd, err := os.Getwd()
	if err == nil {
		for _, iconName := range iconNames {
			candidate := filepath.Join(cwd, iconName)
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
	}

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

func onReady() {
	// Helper for safe min (Go <1.21 compatible)
	minInt := func(a, b int) int {
		if a < b {
			return a
		}
		return b
	}

	currentDir, err := os.Getwd()
	if err != nil {fmt.Printf("Error: os.Getwd !")}

	// parentDir := ""
	// if currentDir != "" {
	// 	parentDir = filepath.Dir(currentDir)
	// }

	iconData := getTrayIconData()

	// Debug output (optional)
	if os.Getenv("DEBUG") == "1" {
		fmt.Printf("Icon data length: %d bytes\n", len(iconData))
		if len(iconData) > 0 {
			n := minInt(16, len(iconData))
			fmt.Printf("Icon header (first %d bytes): %x\n", n, iconData[:n])
		} else {
			fmt.Println("Icon data is empty")
		}
	}

	if len(iconData) >= 4 && iconData[0] == 0 && iconData[1] == 0 && iconData[2] == 1 && iconData[3] == 0 {
	    // EXTRA SAFETY: ensure it's not just header — e.g., must be ≥ 22 bytes (min ICO size)
	    if len(iconData) < 22 {
	        if logger != nil {
	            logger.Printf("⚠️ ICO too small (%d bytes)", len(iconData))
	        }
	    } else {
	        systray.SetIcon(iconData)   // ← now much less likely to trigger false error
	        if logger != nil {
	            logger.Printf("Tray icon set")
	        }
	    }
	}

	// Validate ICO header: must be at least 4 bytes and match 00 00 01 00
	isValidICO := len(iconData) >= 4 &&
		iconData[0] == 0 &&
		iconData[1] == 0 &&
		iconData[2] == 1 &&
		iconData[3] == 0

	if isValidICO {
		systray.SetIcon(iconData) // ✅ No error — correct usage
		if logger != nil {
			logger.Printf("Tray icon set (%d bytes)", len(iconData))
		}
	} else {
		if logger != nil {
			if len(iconData) == 0 {
				logger.Printf("⚠️ No icon data found — using emoji title only")
			} else {
				n := minInt(8, len(iconData))
				hdr := iconData[:n]
				logger.Printf("⚠️ Invalid icon format (not ICO). Header: %x", hdr)
			}
		}
		// Proceed without icon — rely on emoji title & tooltip
	}
	
	// Always set title with emoji (works as fallback)
	systray.SetTitle("📁 File Monitor - Running " + string(currentDir))
	if iconTray := getTrayIconData(); len(iconTray) > 0 {
		systray.SetIcon(iconTray)
	}
	systray.SetTooltip("File Monitor - Running " + string(currentDir))

	menuStart = systray.AddMenuItem("▶️ Start", "Start monitoring")
	if iconStart := getMenuIcon("start"); len(iconStart) > 0 {
		menuStart.SetIcon(iconStart)
	}
	
	menuStop = systray.AddMenuItem("⏹️ Stop", "Stop monitoring")
	if iconStop := getMenuIcon("stop"); len(iconStop) > 0 {
		menuStop.SetIcon(iconStop)
	}
	
	menuPause = systray.AddMenuItem("⏸️ Pause", "Pause monitoring")
	if iconPause := getMenuIcon("pause"); len(iconPause) > 0 {
		menuPause.SetIcon(iconPause)
	}
	
	menuResume = systray.AddMenuItem("⏯️ Resume", "Resume monitoring")
	if iconResume := getMenuIcon("resume"); len(iconResume) > 0 {
		menuResume.SetIcon(iconResume)
	}
	
	systray.AddSeparator()
	
	menuTextNotif = systray.AddMenuItemCheckbox("🔔 Test Notifications", "Toggle text notifications", false)
	if iconNotif := getMenuIcon("notification"); len(iconNotif) > 0 {
		menuTextNotif.SetIcon(iconNotif)
	}
	
	systray.AddSeparator()
	
	menuQuit = systray.AddMenuItem("🚪 Exit", "Exit the application")
	if iconExit := getMenuIcon("exit"); len(iconExit) > 0 {
		menuQuit.SetIcon(iconExit)
	}

	menuStart.Disable()
	menuStop.Enable()
	menuPause.Enable()
	menuResume.Hide()

	go func() {
		for {
			select {
			case <-menuStart.ClickedCh:
				handleTrayStart()
			case <-menuStop.ClickedCh:
				handleTrayStop()
			case <-menuPause.ClickedCh:
				handleTrayPause()
			case <-menuResume.ClickedCh:
				handleTrayResume()
			case <-menuTextNotif.ClickedCh:
				if menuTextNotif.Checked() {
					menuTextNotif.Uncheck()
					fmt.Println("🔕 Text notifications disabled")
				} else {
					menuTextNotif.Check()
					fmt.Println("🔔 Text notifications enabled")
				}
			case <-menuQuit.ClickedCh:
				fmt.Println("👋 Exiting file monitor...")
				systray.Quit()
				os.Exit(0)
			}
		}
	}()
}

func handleTrayStart() {
	if monitorRunning {
		fmt.Println("⚠️  Monitor already running")
		return
	}

	if len(savedArgs) == 0 {
		fmt.Println("❌ No saved configuration. Please restart the program.")
		return
	}

	fmt.Println("▶️  Starting monitor from tray...")
	menuStart.Disable()
	menuStop.Enable()
	menuPause.Enable()
	menuResume.Hide()
	menuPause.Show()
	systray.SetTooltip("File Monitor - Running")

	go func() {
		monitorMu.Lock()
		watchedDirs = make(map[string]bool)
		watchedFiles = make(map[string]bool)
		monitorMu.Unlock()

		// Parse args again for exceptions
		var exceptions []string
		var paths []string
		
		for i := 0; i < len(savedArgs); i++ {
			if savedArgs[i] == "-e" || savedArgs[i] == "--exception" {
				if i+1 < len(savedArgs) {
					exceptions = append(exceptions, savedArgs[i+1])
					i++
				}
			} else {
				paths = append(paths, savedArgs[i])
			}
		}

		err := startMonitorMultiple(paths, exceptions)
		if err != nil {
			fmt.Printf("❌ Monitor error: %v\n", err)
			menuStart.Enable()
			menuStop.Disable()
			menuPause.Disable()
		}
	}()
}

func handleTrayStop() {
	if !monitorRunning {
		fmt.Println("⚠️  Monitor not running")
		return
	}

	fmt.Println("⏹️  Stopping monitor...")
	stopMonitorCh <- true
	monitorRunning = false
	monitorPaused = false

	menuStart.Enable()
	menuStop.Disable()
	menuPause.Disable()
	menuResume.Hide()
	menuPause.Show()
	systray.SetTooltip("File Monitor - Stopped")
}

func handleTrayPause() {
	if !monitorRunning {
		fmt.Println("⚠️  Monitor not running")
		return
	}

	fmt.Println("⏸️  Pausing monitor...")
	monitorPaused = true
	menuPause.Hide()
	menuResume.Show()
	menuResume.Enable()
	systray.SetTooltip("File Monitor - Paused")
}

func handleTrayResume() {
	fmt.Println("⏯️  Resuming monitor...")
	monitorPaused = false
	menuResume.Hide()
	menuPause.Show()
	menuPause.Enable()
	systray.SetTooltip("File Monitor - Running")
}

func onExit() {
	fmt.Println("System tray exited")
}

func getTrayIconData() []byte {
	if appConfig == nil {
		return nil
	}
	
	if appConfig.TrayIcon != "" {
		if data, err := os.ReadFile(appConfig.TrayIcon); err == nil && len(data) > 0 {
			if logger != nil {
				logger.Printf("Using tray icon from config: %s", appConfig.TrayIcon)
			}
			return data
		}
		
		exePath, err := os.Executable()
		if err == nil {
			exeDir := filepath.Dir(exePath)
			iconPath := filepath.Join(exeDir, appConfig.TrayIcon)
			if os.Getenv("DEBUG") == "1" { fmt.Printf("iconPath: %s\n", iconPath)}
			if data, err := os.ReadFile(iconPath); err == nil && len(data) > 0 {
				if logger != nil {
					logger.Printf("Using tray icon from config (exe dir): %s", iconPath)
				}
				return data
			}
		}
		
		cwd, err := os.Getwd()
		if err == nil {
			iconPath := filepath.Join(cwd, appConfig.TrayIcon)
			if data, err := os.ReadFile(iconPath); err == nil && len(data) > 0 {
				if logger != nil {
					logger.Printf("Using tray icon from config (cwd): %s", iconPath)
				}
				return data
			}
		}
	}
	
	iconPath := findNotificationIcon()
	if iconPath != "" {
		data, err := os.ReadFile(iconPath)
		if err == nil && len(data) > 0 {
			if logger != nil {
				logger.Printf("Using tray icon from default location: %s", iconPath)
			}
			return data
		}
	}
	
	return nil
}

func getMenuIcon(menuType string) []byte {
	if appConfig == nil {
		return nil
	}
	
	var iconName string
	
	switch menuType {
	case "start":
		iconName = "start.ico"
		if appConfig.MenuIcons.Start != "" {
			iconName = appConfig.MenuIcons.Start
		}
	case "stop":
		iconName = "stop.ico"
		if appConfig.MenuIcons.Stop != "" {
			iconName = appConfig.MenuIcons.Stop
		}
	case "pause":
		iconName = "pause.ico"
		if appConfig.MenuIcons.Pause != "" {
			iconName = appConfig.MenuIcons.Pause
		}
	case "resume":
		iconName = "resume.ico"
		if appConfig.MenuIcons.Resume != "" {
			iconName = appConfig.MenuIcons.Resume
		}
	case "notification":
		iconName = "notification.ico"
		if appConfig.MenuIcons.Notification != "" {
			iconName = appConfig.MenuIcons.Notification
		}
	case "exit":
		iconName = "exit.ico"
		if appConfig.MenuIcons.Exit != "" {
			iconName = appConfig.MenuIcons.Exit
		}
	default:
		return nil
	}
	
	var iconPaths []string
	
	if appConfig.MenuIconsDir != "" {
		iconPaths = append(iconPaths, filepath.Join(appConfig.MenuIconsDir, iconName))
	}
	
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	iconPaths = append(iconPaths, filepath.Join(exeDir, "menu_icons", iconName))
	iconPaths = append(iconPaths, filepath.Join(exeDir, "icons", iconName))
	iconPaths = append(iconPaths, filepath.Join(exeDir, iconName))
	
	cwd, _ := os.Getwd()
	iconPaths = append(iconPaths, filepath.Join(cwd, "menu_icons", iconName))
	iconPaths = append(iconPaths, filepath.Join(cwd, "icons", iconName))
	iconPaths = append(iconPaths, filepath.Join(cwd, iconName))
	
	for _, path := range iconPaths {
		if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
			if logger != nil {
				logger.Printf("Using menu icon for %s: %s", menuType, path)
			}
			return data
		}
	}
	
	return nil
}
