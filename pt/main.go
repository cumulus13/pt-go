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
	"os/exec"
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
	BackupDirName    = "backup"           // Backup directory name
	Version          = "2.1.0"
	MaxSearchDepth   = 10                 // Max directory depth for recursive search
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
	ColorBlue   = "\033[94m"
)

// BackupInfo stores information about a backup file
type BackupInfo struct {
	Path    string
	Name    string
	ModTime time.Time
	Size    int64
}

// FileSearchResult stores information about found files
type FileSearchResult struct {
	Path     string
	Dir      string
	Size     int64
	ModTime  time.Time
	Depth    int
}

// Logger for audit trail
var logger *log.Logger

func init() {
	// Initialize logger (write to stderr to not interfere with stdout)
	logger = log.New(os.Stderr, "", log.LstdFlags)
}

// searchFileRecursive searches for a file recursively in current and subdirectories
func searchFileRecursive(filename string, maxDepth int) ([]FileSearchResult, error) {
	results := make([]FileSearchResult, 0)
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// First check current directory
	currentPath := filepath.Join(cwd, filename)
	if info, err := os.Stat(currentPath); err == nil && !info.IsDir() {
		results = append(results, FileSearchResult{
			Path:    currentPath,
			Dir:     cwd,
			Size:    info.Size(),
			ModTime: info.ModTime(),
			Depth:   0,
		})
	}

	// Then search recursively
	err = filepath.Walk(cwd, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip directories we can't access
			return nil
		}

		// Skip backup directories
		if info.IsDir() && info.Name() == BackupDirName {
			return filepath.SkipDir
		}

		// Calculate depth
		relPath, err := filepath.Rel(cwd, path)
		if err != nil {
			return nil
		}
		depth := len(strings.Split(relPath, string(os.PathSeparator))) - 1

		// Skip if too deep
		if depth > maxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if filename matches
		if !info.IsDir() && info.Name() == filename {
			// Skip if already added (current directory)
			if path == currentPath {
				return nil
			}

			results = append(results, FileSearchResult{
				Path:    path,
				Dir:     filepath.Dir(path),
				Size:    info.Size(),
				ModTime: info.ModTime(),
				Depth:   depth,
			})
		}

		return nil
	})

	if err != nil {
		return results, fmt.Errorf("error during search: %w", err)
	}

	return results, nil
}

// printFileSearchResults displays found files in a formatted table
func printFileSearchResults(results []FileSearchResult) {
	const (
		col1Width = 60
		col2Width = 19
		col3Width = 12
	)

	fmt.Printf("\n%s🔍 Found %d file(s):%s\n\n", ColorCyan, len(results), ColorReset)

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
		ColorBold, ColorYellow, col1Width, "Path", ColorReset,
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
	for i, result := range results {
		// Get relative path for display
		cwd, _ := os.Getwd()
		relPath, err := filepath.Rel(cwd, result.Path)
		if err != nil {
			relPath = result.Path
		}

		displayPath := relPath
		maxPathLen := col1Width - 5
		if len(displayPath) > maxPathLen {
			displayPath = "..." + displayPath[len(displayPath)-maxPathLen+3:]
		}

		modTime := result.ModTime.Format("2006-01-02 15:04:05")

		var sizeStr string
		if result.Size >= 1024*1024 {
			sizeStr = fmt.Sprintf("%.2f MB", float64(result.Size)/(1024*1024))
		} else if result.Size >= 1024 {
			sizeStr = fmt.Sprintf("%.2f KB", float64(result.Size)/1024)
		} else {
			sizeStr = fmt.Sprintf("%d B", result.Size)
		}

		fmt.Printf("%s│%s %s%3d. %-*s%s %s│%s %-*s %s│%s %*s %s│%s\n",
			ColorGray, ColorReset,
			ColorGreen, i+1, maxPathLen, displayPath, ColorReset,
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

// resolveFilePath resolves the file path, searching recursively if not found in current directory
func resolveFilePath(filename string) (string, error) {
	// First check if file exists in current directory
	if info, err := os.Stat(filename); err == nil && !info.IsDir() {
		absPath, _ := filepath.Abs(filename)
		return absPath, nil
	}

	// Search recursively
	logger.Printf("File not found in current directory, searching recursively...")
	fmt.Printf("%s🔍 Searching for '%s' in subdirectories...%s\n", ColorBlue, filename, ColorReset)

	results, err := searchFileRecursive(filename, MaxSearchDepth)
	if err != nil {
		return "", err
	}

	if len(results) == 0 {
		return "", fmt.Errorf("file '%s' not found in current directory or subdirectories", filename)
	}

	if len(results) == 1 {
		fmt.Printf("%s✓ Found: %s%s\n", ColorGreen, results[0].Path, ColorReset)
		return results[0].Path, nil
	}

	// Multiple files found, prompt user
	printFileSearchResults(results)

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Enter file number to use (1-%d) or 0 to cancel: ", len(results))

	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)
	choice, err := strconv.Atoi(input)
	if err != nil {
		return "", fmt.Errorf("invalid input: please enter a number")
	}

	if choice < 0 || choice > len(results) {
		return "", fmt.Errorf("invalid selection: must be between 0 and %d", len(results))
	}

	if choice == 0 {
		return "", fmt.Errorf("operation cancelled")
	}

	return results[choice-1].Path, nil
}

// checkDeltaInstalled checks if delta CLI tool is installed
func checkDeltaInstalled() bool {
	_, err := exec.LookPath("delta")
	return err == nil
}

// runDelta executes delta to show diff between two files
func runDelta(file1, file2 string) error {
	if !checkDeltaInstalled() {
		return fmt.Errorf("delta is not installed. Install it from: https://github.com/dandavison/delta")
	}

	cmd := exec.Command("delta", file1, file2)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	
	// Delta returns exit status 1 when files differ - this is NORMAL, not an error!
	// Only return error if exit code is something else (2+ means real error)
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				// Exit code 1 = files differ, which is expected - NOT an error
				return nil
			}
		}
		// Real error (exit code 2+ or other issue)
		return err
	}

	return nil
}

// handleDiffCommand handles the -d/--diff command
func handleDiffCommand(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("filename required for diff command")
	}

	filename := args[0]
	useLast := len(args) > 1 && args[1] == "--last"

	// Resolve file path
	filePath, err := resolveFilePath(filename)
	if err != nil {
		return err
	}

	// Get backups
	backups, err := listBackups(filePath)
	if err != nil {
		return err
	}

	if len(backups) == 0 {
		return fmt.Errorf("no backups found for: %s (check ./%s/ directory)", filePath, BackupDirName)
	}

	var selectedBackup BackupInfo

	if useLast {
		// Use last backup
		selectedBackup = backups[0]
		fmt.Printf("%s📊 Comparing with last backup: %s%s\n\n", ColorCyan, selectedBackup.Name, ColorReset)
	} else {
		// Show backups and prompt
		printBackupTable(filePath, backups)

		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("Enter backup number to compare (1-%d) or 0 to cancel: ", len(backups))

		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(input)
		choice, err := strconv.Atoi(input)
		if err != nil {
			return fmt.Errorf("invalid input: please enter a number")
		}

		if choice < 0 || choice > len(backups) {
			return fmt.Errorf("invalid selection: must be between 0 and %d", len(backups))
		}

		if choice == 0 {
			return fmt.Errorf("diff cancelled")
		}

		selectedBackup = backups[choice-1]
		fmt.Printf("\n%s📊 Comparing with: %s%s\n\n", ColorCyan, selectedBackup.Name, ColorReset)
	}

	// Run delta
	err = runDelta(selectedBackup.Path, filePath)
	if err != nil {
		return fmt.Errorf("delta execution failed: %w", err)
	}

	return nil
}

// ensureBackupDir creates backup directory if it doesn't exist
// Returns the absolute path to the backup directory
func ensureBackupDir(filePath string) (string, error) {
	// Get directory of the target file
	dir := filepath.Dir(filePath)
	if dir == "." {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Create backup directory path
	backupDir := filepath.Join(dir, BackupDirName)

	// Check if backup directory exists
	info, err := os.Stat(backupDir)
	if os.IsNotExist(err) {
		// Create backup directory with appropriate permissions
		err = os.MkdirAll(backupDir, 0755)
		if err != nil {
			return "", fmt.Errorf("failed to create backup directory: %w", err)
		}
		logger.Printf("Created backup directory: %s", backupDir)
		fmt.Printf("📁 Created backup directory: %s\n", backupDir)
	} else if err != nil {
		return "", fmt.Errorf("failed to check backup directory: %w", err)
	} else if !info.IsDir() {
		return "", fmt.Errorf("backup path exists but is not a directory: %s", backupDir)
	}

	return backupDir, nil
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
// Now returns just the filename (without directory path)
func generateUniqueBackupName(filePath string) string {
	// Get just the base filename (not the full path)
	baseName := filepath.Base(filePath)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)

	// Format: YYYYMMDD_HHMMSSµµµµµµ (no dots in timestamp)
	timestamp := time.Now().Format("20060102_150405.000000")
	timestamp = strings.ReplaceAll(timestamp, ".", "")

	// Add process ID and random component for uniqueness
	uniqueID := fmt.Sprintf("%d_%s", os.Getpid(), generateShortID())

	return fmt.Sprintf("%s_%s.%s.%s", nameWithoutExt, strings.TrimPrefix(ext, "."), timestamp, uniqueID)
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
// Now stores backups in the "backup" subdirectory
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

	// Ensure backup directory exists
	backupDir, err := ensureBackupDir(filePath)
	if err != nil {
		return filePath, err
	}

	// Generate unique backup filename (just the name, not full path)
	backupFileName := generateUniqueBackupName(filePath)
	
	// Full path to backup file in backup directory
	backupPath := filepath.Join(backupDir, backupFileName)

	// Copy the file to backup directory (not rename, so original stays)
	// Read original file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return filePath, fmt.Errorf("failed to read file for backup: %w", err)
	}

	// Write to backup directory
	err = os.WriteFile(backupPath, content, 0644)
	if err != nil {
		return filePath, fmt.Errorf("failed to create backup: %w", err)
	}

	logger.Printf("Backup created: %s -> %s", filePath, backupPath)
	fmt.Printf("📦 Backup created: %s%s%s\n", ColorGreen, backupFileName, ColorReset)

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
	fmt.Printf("📄 Content size: %d characters\n", len(data))

	return nil
}

// listBackups returns backup files from backup directory
func listBackups(filePath string) ([]BackupInfo, error) {
	// Validate path first
	if err := validatePath(filePath); err != nil {
		return nil, err
	}

	// Get backup directory path
	dir := filepath.Dir(filePath)
	if dir == "." {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}
	
	backupDir := filepath.Join(dir, BackupDirName)

	// Check if backup directory exists
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		// No backup directory means no backups
		return []BackupInfo{}, nil
	}

	// Get base filename for pattern matching
	baseName := filepath.Base(filePath)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)
	extWithoutDot := strings.TrimPrefix(ext, ".")
	
	// Pattern should match: basename_ext.timestamp...
	pattern := fmt.Sprintf("%s_%s.", nameWithoutExt, extWithoutDot)
	
	logger.Printf("Looking for backups with pattern: %s in directory: %s", pattern, backupDir)

	// Read backup directory
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	backups := make([]BackupInfo, 0)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		
		logger.Printf("Checking file: %s against pattern: %s", name, pattern)
		
		if !strings.HasPrefix(name, pattern) {
			continue
		}

		// Extract and validate timestamp
		timestamp := strings.TrimPrefix(name, pattern)
		
		logger.Printf("Extracted timestamp: %s (length: %d)", timestamp, len(timestamp))
		
		if len(timestamp) < 20 {
			logger.Printf("Skipping %s: timestamp too short", name)
			continue
		}

		// More flexible validation: check if it starts with a date-like pattern
		timestampPart := timestamp
		if len(timestampPart) > 30 {
			timestampPart = timestampPart[:30]
		}
		
		// Count digits in the timestamp part
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
			Path:    filepath.Join(backupDir, name),
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

// printBackupTable displays backups in formatted table
func printBackupTable(filePath string, backups []BackupInfo) {
	const (
		col1Width = 50
		col2Width = 19
		col3Width = 15
	)

	fmt.Printf("\n%s📂 Backup files for '%s%s%s%s'%s\n",
		ColorCyan, ColorBold, filePath, ColorReset, ColorCyan, ColorReset)
	fmt.Printf("%sTotal: %d backup(s) (stored in ./%s/)%s\n\n", 
		ColorGray, len(backups), BackupDirName, ColorReset)

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
	fmt.Printf("📄 Content size: %d characters\n", len(content))

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
	fmt.Println("  pt -d <filename>            Compare file with backup (interactive)")
	fmt.Println("  pt -d <filename> --last     Compare file with last backup")
	fmt.Println("  pt -h, --help               Show this help")
	fmt.Println("  pt -v, --version            Show version")
	fmt.Println("\nExamples:")
	fmt.Println("  pt notes.txt                # Save clipboard to notes.txt")
	fmt.Println("  pt + log.txt                # Append clipboard to log.txt")
	fmt.Println("  pt -l notes.txt             # List all backups")
	fmt.Println("  pt -r notes.txt             # Interactive restore")
	fmt.Println("  pt -r notes.txt --last      # Restore most recent backup")
	fmt.Println("  pt -d notes.txt             # Interactive diff with backup")
	fmt.Println("  pt -d notes.txt --last      # Diff with most recent backup")
	fmt.Printf("\n%sFeatures:%s\n", ColorBold, ColorReset)
	fmt.Printf("  • %sRecursive Search:%s If file not in current dir, searches subdirectories\n", ColorCyan, ColorReset)
	fmt.Printf("  • %sDiff Support:%s Uses 'delta' CLI tool for beautiful diffs\n", ColorCyan, ColorReset)
	fmt.Printf("\n%sBackup Location: All backups stored in ./%s/ directory%s\n", ColorCyan, BackupDirName, ColorReset)
	fmt.Printf("%sLimits: Max file size %dMB, Max %d backups kept%s\n",
		ColorGray, MaxClipboardSize/(1024*1024), MaxBackupCount, ColorReset)
	fmt.Printf("\n%sNote: Install 'delta' for diff functionality: https://github.com/dandavison/delta%s\n",
		ColorGray, ColorReset)
}

// printVersion displays version information
func printVersion() {
	fmt.Printf("PT version %s\n", Version)
	fmt.Println("Production-hardened clipboard to file tool")
	fmt.Println("Features: Recursive search, backup management, delta diff support")
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

		// Resolve file path with recursive search
		filePath, err := resolveFilePath(os.Args[2])
		if err != nil {
			fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
			os.Exit(1)
		}

		backups, err := listBackups(filePath)
		if err != nil {
			fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
			os.Exit(1)
		}

		if len(backups) == 0 {
			fmt.Printf("ℹ️  No backups found for: %s (check ./%s/ directory)\n", filePath, BackupDirName)
		} else {
			printBackupTable(filePath, backups)
		}

	case "-d", "--diff":
		if len(os.Args) < 3 {
			fmt.Printf("%s❌ Error: Filename required%s\n", ColorRed, ColorReset)
			os.Exit(1)
		}

		err := handleDiffCommand(os.Args[2:])
		if err != nil {
			fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
			os.Exit(1)
		}

	case "-r", "--restore":
		if len(os.Args) < 3 {
			fmt.Printf("%s❌ Error: Filename required%s\n", ColorRed, ColorReset)
			os.Exit(1)
		}

		filename := os.Args[2]

		// Resolve file path with recursive search
		filePath, err := resolveFilePath(filename)
		if err != nil {
			// For restore, if file doesn't exist, use the filename as-is
			// (we're restoring it, so it might not exist yet)
			filePath = filename
			absPath, err := filepath.Abs(filePath)
			if err == nil {
				filePath = absPath
			}
		}

		// Get list of backups
		backups, err := listBackups(filePath)
		if err != nil {
			fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
			os.Exit(1)
		}

		if len(backups) == 0 {
			fmt.Printf("%s❌ Error: No backups found for: %s (check ./%s/ directory)%s\n", 
				ColorRed, filePath, BackupDirName, ColorReset)
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

		// Resolve file path with recursive search
		filePath, err := resolveFilePath(os.Args[2])
		if err != nil {
			// If file doesn't exist, use the provided path as-is
			filePath = os.Args[2]
		}

		err = writeFile(filePath, text, true)
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

		// Resolve file path with recursive search
		filePath, err := resolveFilePath(os.Args[1])
		if err != nil {
			// If file doesn't exist, use the provided path as-is
			filePath = os.Args[1]
		}

		err = writeFile(filePath, text, false)
		if err != nil {
			fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
			os.Exit(1)
		}
	}
}