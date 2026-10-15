//go:build windows || tray
// +build windows tray

// System tray support.
//
// Compiled on Windows by default (pure Go, no CGO required).
// On Linux/macOS the tray needs CGO (GTK3 + appindicator / Cocoa), so it is
// opt-in: build with `CGO_ENABLED=1 go build -tags tray ./pt`.
// Default Linux/macOS builds use tray_stub.go and stay fully static (CGO_ENABLED=0).

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/getlantern/systray"
)

var (
	menuStart     *systray.MenuItem
	menuStop      *systray.MenuItem
	menuPause     *systray.MenuItem
	menuResume    *systray.MenuItem
	menuTextNotif *systray.MenuItem
	menuQuit      *systray.MenuItem
)

// startTray launches the system tray in the background.
func startTray() {
	go systray.Run(onReady, onExit)
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
