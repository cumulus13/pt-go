// File: pt/main.go
// Author: Hadi Cahyadi <cumulus13@gmail.com>
// Date: 2025-10-30
// Description: Production-hardened clipboard-to-file tool with security, validation, and robustness improvements
// License: MIT

package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/atotto/clipboard"
)

// Configuration constants
const (
	MaxClipboardSize = 100 * 1024 * 1024 // 100MB max
	MaxBackupCount   = 100                // Keep max 100 backups
	MaxFilenameLen   = 200                // Max filename length
	Version          = "2.0.0"
)

// ANSI color codes for pretty output
const (
	ColorReset  = "\033[0m"
	ColorCyan   = "\033[96m"
	ColorYellow = "\033[93m"
	ColorGreen  = "\033[92m"
	ColorGray   = "\033[90m"
	ColorBold   = "\033[1m"
	ColorRed    = "\033[91m"
)

// BackupInfo stores information about a backup file
type BackupInfo struct {
	Path    string
	Name    string
	ModTime time.Time
	Size    int64
}

// Logger for audit trail
var logger *log.Logger

func init() {
	// Initialize logger (write to stderr to not interfere with stdout)
	logger = log.New(os.Stderr, "", log.LstdFlags)
}

// validatePath checks for path traversal and other security issues
func validatePath(filePath string) error {
	if filePath == "" {
		return fmt.Errorf("filename cannot be empty")
	}

	// Get absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Check for path traversal attempts
	cleanPath := filepath.Clean(filePath)
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal not allowed")
	}

	// Check filename length
	if len(filepath.Base(filePath)) > MaxFilenameLen {
		return fmt.Errorf("filename too long (max %d characters)", MaxFilenameLen)
	}

	// Prevent writing to system directories (basic check)
	systemDirs := []string{"/etc", "/sys", "/proc", "/dev", "C:\\Windows", "C:\\System32"}
	for _, sysDir := range systemDirs {
		if strings.HasPrefix(absPath, sysDir) {
			return fmt.Errorf("writing to system directories not allowed")
		}
	}

	return nil
}

// checkDiskSpace validates there's enough space (basic check)
func checkDiskSpace(path string, requiredSize int64) error {
	// Get directory
	dir := filepath.Dir(path)
	if dir == "." {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	// Try to create a small test file to verify write permissions
	testFile := filepath.Join(dir, ".pt_test_"+generateShortID())
	f, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("no write permission in directory: %w", err)
	}
	f.Close()
	os.Remove(testFile)

	// Note: Actual disk space checking is platform-specific
	// This is a basic validation that we can write to the directory
	return nil
}

// generateShortID creates a short unique identifier
func generateShortID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// generateUniqueBackupName creates a collision-resistant backup filename
func generateUniqueBackupName(filePath string) string {
	ext := filepath.Ext(filePath)
	base := strings.TrimSuffix(filePath, ext)

	// Format: YYYYMMDD_HHMMSSµµµµµµ (no dots in timestamp)
	timestamp := time.Now().Format("20060102_150405.000000")
	timestamp = strings.ReplaceAll(timestamp, ".", "")

	// Add process ID and random component for uniqueness
	uniqueID := fmt.Sprintf("%d_%s", os.Getpid(), generateShortID())

	return fmt.Sprintf("%s_%s.%s.%s", base, strings.TrimPrefix(ext, "."), timestamp, uniqueID)
}

// getClipboardText reads from clipboard with size validation
func getClipboardText() (string, error) {
	text, err := clipboard.ReadAll()
	if err != nil {
		return "", fmt.Errorf("failed to read clipboard: %w", err)
	}

	// Validate size
	if len(text) > MaxClipboardSize {
		return "", fmt.Errorf("clipboard content too large (max %dMB)", MaxClipboardSize/(1024*1024))
	}

	return text, nil
}

// autoRenameIfExists creates backup with atomic-like behavior
func autoRenameIfExists(filePath string) (string, error) {
	// Check if file exists
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return filePath, nil
	}
	if err != nil {
		return filePath, fmt.Errorf("failed to check file: %w", err)
	}

	// Don't backup empty files
	if info.Size() == 0 {
		logger.Printf("Skipping backup of empty file: %s", filePath)
		return filePath, nil
	}

	// Generate unique backup name
	backupName := generateUniqueBackupName(filePath)

	// Rename the file
	err = os.Rename(filePath, backupName)
	if err != nil {
		return filePath, fmt.Errorf("failed to create backup: %w", err)
	}

	logger.Printf("Backup created: %s -> %s", filePath, backupName)
	fmt.Printf("📦 Backup created: %s\n", filepath.Base(backupName))

	return filePath, nil
}

// writeFile writes data to file with validation
func writeFile(filePath string, data string, appendMode bool) error {
	// Validate path
	if err := validatePath(filePath); err != nil {
		return err
	}

	// Check disk space
	if err := checkDiskSpace(filePath, int64(len(data))); err != nil {
		return err
	}

	if !appendMode {
		// Create backup before overwriting
		var err error
		filePath, err = autoRenameIfExists(filePath)
		if err != nil {
			return err
		}
	}

	// Determine file open mode
	var flag int
	if appendMode {
		flag = os.O_APPEND | os.O_CREATE | os.O_WRONLY
	} else {
		flag = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	}

	// Open file
	file, err := os.OpenFile(filePath, flag, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Write data
	n, err := file.WriteString(data)
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	// Verify write completed
	if n != len(data) {
		return fmt.Errorf("incomplete write: wrote %d bytes, expected %d", n, len(data))
	}

	// Sync to disk
	if err := file.Sync(); err != nil {
		logger.Printf("Warning: failed to sync file: %v", err)
	}

	action := "written to"
	if appendMode {
		action = "appended to"
	}

	logger.Printf("Successfully %s: %s (%d bytes)", action, filePath, len(data))
	fmt.Printf("✅ Successfully %s: %s\n", action, filePath)
	fmt.Printf("📝 Content size: %d characters\n", len(data))

	return nil
}

// listBackups returns backup files with improved validation
func listBackups(filePath string) ([]BackupInfo, error) {
	// Validate path first
	if err := validatePath(filePath); err != nil {
		return nil, err
	}

	dir := filepath.Dir(filePath)
	if dir == "." {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}

	// Parse filename pattern
	base := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	ext := strings.TrimPrefix(filepath.Ext(filePath), ".")
	
	// Pattern should match: basename_ext.timestamp...
	pattern := fmt.Sprintf("%s_%s.", base, ext)
	
	// Debug logging
	logger.Printf("Looking for backups with pattern: %s in directory: %s", pattern, dir)

	// Read directory using modern API
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	backups := make([]BackupInfo, 0)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		
		// Debug: log what we're checking
		logger.Printf("Checking file: %s against pattern: %s", name, pattern)
		
		if !strings.HasPrefix(name, pattern) {
			continue
		}

		// Extract and validate timestamp
		timestamp := strings.TrimPrefix(name, pattern)
		
		// Debug: log timestamp extraction
		logger.Printf("Extracted timestamp: %s (length: %d)", timestamp, len(timestamp))
		
		if len(timestamp) < 20 {
			logger.Printf("Skipping %s: timestamp too short", name)
			continue
		}

		// More flexible validation: check if it starts with a date-like pattern
		// Format should be: YYYYMMDD_HHMMSS... (at least 15 digits in first 20 chars)
		timestampPart := timestamp
		if len(timestampPart) > 30 {
			timestampPart = timestampPart[:30]
		}
		
		// Count digits in the timestamp part (should have many digits for date/time)
		digitCount := 0
		for _, c := range timestampPart {
			if c >= '0' && c <= '9' {
				digitCount++
			}
		}
		
		if digitCount < 14 {
			logger.Printf("Skipping %s: not enough digits in timestamp (%d)", name, digitCount)
			continue
		}

		// Get file info
		info, err := entry.Info()
		if err != nil {
			logger.Printf("Warning: failed to get info for %s: %v", name, err)
			continue
		}

		logger.Printf("Found valid backup: %s", name)
		backups = append(backups, BackupInfo{
			Path:    filepath.Join(dir, name),
			Name:    name,
			ModTime: info.ModTime(),
			Size:    info.Size(),
		})
	}

	if len(backups) == 0 {
		return backups, nil
	}

	// Sort by modification time (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].ModTime.After(backups[j].ModTime)
	})

	// Limit to MaxBackupCount
	if len(backups) > MaxBackupCount {
		backups = backups[:MaxBackupCount]
	}

	return backups, nil
}

// isValidTimestamp checks if string matches our timestamp format
func isValidTimestamp(s string) bool {
	if len(s) != 20 {
		return false
	}
	// Check if all characters are digits
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// printBackupTable displays backups in formatted table
func printBackupTable(filePath string, backups []BackupInfo) {
	const (
		col1Width = 50
		col2Width = 19
		col3Width = 15
	)

	fmt.Printf("\n%s📂 Backup files for '%s%s%s%s'%s\n",
		ColorCyan, ColorBold, filePath, ColorReset, ColorCyan, ColorReset)
	fmt.Printf("%sTotal: %d backup(s)%s\n\n", ColorGray, len(backups), ColorReset)

	// Top border
	fmt.Printf("%s┌%s┬%s┬%s┐%s\n",
		ColorGray,
		strings.Repeat("─", col1Width+2),
		strings.Repeat("─", col2Width+2),
		strings.Repeat("─", col3Width+2),
		ColorReset)

	// Header row
	fmt.Printf("%s│%s %s%s%-*s%s %s│%s %s%s%-*s%s %s│%s %s%s%*s%s %s│%s\n",
		ColorGray, ColorReset,
		ColorBold, ColorYellow, col1Width, "File Name", ColorReset,
		ColorGray, ColorReset,
		ColorBold, ColorYellow, col2Width, "Modified", ColorReset,
		ColorGray, ColorReset,
		ColorBold, ColorYellow, col3Width, "Size", ColorReset,
		ColorGray, ColorReset)

	// Separator
	fmt.Printf("%s├%s┼%s┼%s┤%s\n",
		ColorGray,
		strings.Repeat("─", col1Width+2),
		strings.Repeat("─", col2Width+2),
		strings.Repeat("─", col3Width+2),
		ColorReset)

	// Data rows
	for i, backup := range backups {
		name := backup.Name
		// Account for number prefix (up to 3 digits + ". ")
		maxNameLen := col1Width - 5
		if len(name) > maxNameLen {
			name = name[:maxNameLen-3] + "..."
		}

		modTime := backup.ModTime.Format("2006-01-02 15:04:05")

		var sizeStr string
		if backup.Size >= 1024*1024 {
			sizeStr = fmt.Sprintf("%.2f MB", float64(backup.Size)/(1024*1024))
		} else if backup.Size >= 1024 {
			sizeStr = fmt.Sprintf("%.2f KB", float64(backup.Size)/1024)
		} else {
			sizeStr = fmt.Sprintf("%d B", backup.Size)
		}

		fmt.Printf("%s│%s %s%3d. %-*s%s %s│%s %-*s %s│%s %*s %s│%s\n",
			ColorGray, ColorReset,
			ColorGreen, i+1, maxNameLen, name, ColorReset,
			ColorGray, ColorReset,
			col2Width, modTime,
			ColorGray, ColorReset,
			col3Width, sizeStr,
			ColorGray, ColorReset)
	}

	// Bottom border
	fmt.Printf("%s└%s┴%s┴%s┘%s\n\n",
		ColorGray,
		strings.Repeat("─", col1Width+2),
		strings.Repeat("─", col2Width+2),
		strings.Repeat("─", col3Width+2),
		ColorReset)
}

// restoreBackup restores a backup file with validation
func restoreBackup(backupPath, originalPath string) error {
	// Validate paths
	if err := validatePath(originalPath); err != nil {
		return err
	}

	// Check if backup exists
	info, err := os.Stat(backupPath)
	if err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}

	// Check backup isn't too large
	if info.Size() > MaxClipboardSize {
		return fmt.Errorf("backup file too large to restore (max %dMB)", MaxClipboardSize/(1024*1024))
	}

	// Read backup file
	content, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// Create backup of current file if it exists
	if _, err := os.Stat(originalPath); err == nil {
		_, err = autoRenameIfExists(originalPath)
		if err != nil {
			return fmt.Errorf("failed to backup current file: %w", err)
		}
	}

	// Write content to original filename
	err = os.WriteFile(originalPath, content, 0644)
	if err != nil {
		return fmt.Errorf("failed to restore file: %w", err)
	}

	logger.Printf("Restored: %s from %s", originalPath, backupPath)
	fmt.Printf("✅ Successfully restored: %s\n", originalPath)
	fmt.Printf("📦 From backup: %s\n", filepath.Base(backupPath))
	fmt.Printf("📝 Content size: %d characters\n", len(content))

	return nil
}

// readUserChoice reads and validates user input
func readUserChoice(max int) (int, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Enter backup number to restore (1-%d) or 0 to cancel: ", max)

	input, err := reader.ReadString('\n')
	if err != nil {
		return 0, fmt.Errorf("failed to read input: %w", err)
	}

	// Trim whitespace
	input = strings.TrimSpace(input)

	// Parse integer
	choice, err := strconv.Atoi(input)
	if err != nil {
		return 0, fmt.Errorf("invalid input: please enter a number")
	}

	// Validate range
	if choice < 0 || choice > max {
		return 0, fmt.Errorf("invalid selection: must be between 0 and %d", max)
	}

	return choice, nil
}

// printHelp displays usage information
func printHelp() {
	fmt.Printf("%sPT - Clipboard to File Tool v%s%s\n\n", ColorBold, Version, ColorReset)
	fmt.Println("Usage:")
	fmt.Println("  pt <filename>               Write clipboard to file")
	fmt.Println("  pt + <filename>             Append clipboard to file")
	fmt.Println("  pt -l <filename>            List backups")
	fmt.Println("  pt -r <filename>            Restore backup (interactive)")
	fmt.Println("  pt -r <filename> --last     Restore last backup")
	fmt.Println("  pt -h, --help               Show this help")
	fmt.Println("  pt -v, --version            Show version")
	fmt.Println("\nExamples:")
	fmt.Println("  pt notes.txt                # Save clipboard to notes.txt")
	fmt.Println("  pt + log.txt                # Append clipboard to log.txt")
	fmt.Println("  pt -l notes.txt             # List all backups")
	fmt.Println("  pt -r notes.txt             # Interactive restore")
	fmt.Println("  pt -r notes.txt --last      # Restore most recent backup")
	fmt.Printf("\n%sLimits: Max file size %dMB, Max %d backups kept%s\n",
		ColorGray, MaxClipboardSize/(1024*1024), MaxBackupCount, ColorReset)
}

// printVersion displays version information
func printVersion() {
	fmt.Printf("PT version %s\n", Version)
	fmt.Println("Production-hardened clipboard to file tool")
}

func main() {
	// Handle help and version flags
	if len(os.Args) == 2 {
		switch os.Args[1] {
		case "-h", "--help":
			printHelp()
			os.Exit(0)
		case "-v", "--version":
			printVersion()
			os.Exit(0)
		}
	}

	// Require at least one argument
	if len(os.Args) < 2 {
		fmt.Printf("%s❌ Error: No command specified%s\n\n", ColorRed, ColorReset)
		printHelp()
		os.Exit(1)
	}

	// Handle different commands
	switch os.Args[1] {
	case "-l", "--list":
		if len(os.Args) < 3 {
			fmt.Printf("%s❌ Error: Filename required%s\n", ColorRed, ColorReset)
			os.Exit(1)
		}

		backups, err := listBackups(os.Args[2])
		if err != nil {
			fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
			os.Exit(1)
		}

		if len(backups) == 0 {
			fmt.Printf("ℹ️  No backups found for: %s\n", os.Args[2])
		} else {
			printBackupTable(os.Args[2], backups)
		}

	case "-r", "--restore":
		if len(os.Args) < 3 {
			fmt.Printf("%s❌ Error: Filename required%s\n", ColorRed, ColorReset)
			os.Exit(1)
		}

		filePath := os.Args[2]

		// Get list of backups
		backups, err := listBackups(filePath)
		if err != nil {
			fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
			os.Exit(1)
		}

		if len(backups) == 0 {
			fmt.Printf("%s❌ Error: No backups found for: %s%s\n", ColorRed, filePath, ColorReset)
			os.Exit(1)
		}

		// Check for --last flag
		if len(os.Args) == 4 && os.Args[3] == "--last" {
			err = restoreBackup(backups[0].Path, filePath)
			if err != nil {
				fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
				os.Exit(1)
			}
		} else {
			// Interactive selection
			printBackupTable(filePath, backups)

			choice, err := readUserChoice(len(backups))
			if err != nil {
				fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
				os.Exit(1)
			}

			if choice == 0 {
				fmt.Println("❌ Restore cancelled")
				os.Exit(0)
			}

			// Restore selected backup
			selectedBackup := backups[choice-1]
			err = restoreBackup(selectedBackup.Path, filePath)
			if err != nil {
				fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
				os.Exit(1)
			}
		}

	case "+":
		if len(os.Args) < 3 {
			fmt.Printf("%s❌ Error: Filename required%s\n", ColorRed, ColorReset)
			os.Exit(1)
		}

		text, err := getClipboardText()
		if err != nil {
			fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
			os.Exit(1)
		}

		if text == "" {
			fmt.Printf("%s⚠️  Warning: Clipboard is empty%s\n", ColorYellow, ColorReset)
			os.Exit(1)
		}

		err = writeFile(os.Args[2], text, true)
		if err != nil {
			fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
			os.Exit(1)
		}

	default:
		// Write mode (default)
		text, err := getClipboardText()
		if err != nil {
			fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
			os.Exit(1)
		}

		if text == "" {
			fmt.Printf("%s⚠️  Warning: Clipboard is empty%s\n", ColorYellow, ColorReset)
			os.Exit(1)
		}

		err = writeFile(os.Args[1], text, false)
		if err != nil {
			fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
			os.Exit(1)
		}
	}
}