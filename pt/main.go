// File: pt/main.go
// Author: Hadi Cahyadi <cumulus13@gmail.com>
// Date: 2025-11-18
// Description: Production-hardened clipboard-to-file tool with security, validation, Git-like .pt directory and smart version management
// License: MIT

package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
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
	"gopkg.in/yaml.v3"
)

// Configuration constants (defaults)
const (
	DefaultMaxClipboardSize = 100 * 1024 * 1024 // 100MB max
	DefaultMaxBackupCount   = 100                // Keep max 100 backups
	DefaultMaxFilenameLen   = 200                // Max filename length
	DefaultBackupDirName    = ".pt"              // Git-like hidden directory
	DefaultMaxSearchDepth   = 10                 // Max directory depth for recursive search
)

// Version will be loaded from VERSION file
var Version string = "dev"

// Config holds the application configuration
type Config struct {
	MaxClipboardSize int    `yaml:"max_clipboard_size"`
	MaxBackupCount   int    `yaml:"max_backup_count"`
	MaxFilenameLen   int    `yaml:"max_filename_length"`
	BackupDirName    string `yaml:"backup_dir_name"`
	MaxSearchDepth   int    `yaml:"max_search_depth"`
}

// Global config instance
var appConfig *Config
var debugMode bool = false

// ANSI color codes for pretty output
const (
    // Reset
    ColorReset = "\033[0m"

    // Regular Colors
    ColorBlack   = "\033[30m"
    ColorRed     = "\033[91m"
    ColorGreen   = "\033[92m"
    ColorGray    = "\033[90m"
    ColorYellow  = "\033[93m"
    ColorBlue    = "\033[34m"
    ColorMagenta = "\033[95m"
    ColorCyan    = "\033[96m"
    ColorWhite   = "\033[97m"

    // Bright Colors
    ColorBrightBlack   = "\033[90m"
    ColorBrightRed     = "\033[31m"
    ColorBrightGreen   = "\033[32m"
    ColorBrightYellow  = "\033[33m"
    ColorBrightBlue    = "\033[94m"
    ColorBrightMagenta = "\033[35m"
    ColorBrightCyan    = "\033[36m"
    ColorBrightWhite   = "\033[37m"

    // Background Colors
    BgBlack   = "\033[40m"
    BgRed     = "\033[41m"
    BgGreen   = "\033[42m"
    BgYellow  = "\033[43m"
    BgBlue    = "\033[44m"
    BgMagenta = "\033[45m"
    BgCyan    = "\033[46m"
    BgWhite   = "\033[47m"

    // Bright Backgrounds
    BgBrightBlack   = "\033[100m"
    BgBrightRed     = "\033[101m"
    BgBrightGreen   = "\033[102m"
    BgBrightYellow  = "\033[103m"
    BgBrightBlue    = "\033[104m"
    BgBrightMagenta = "\033[105m"
    BgBrightCyan    = "\033[106m"
    BgBrightWhite   = "\033[107m"

    // Text Effects
    ColorBold      = "\033[1m"
    ColorDim       = "\033[2m"
    ColorItalic    = "\033[3m"
    ColorUnderline = "\033[4m"
    ColorBlink     = "\033[5m"
    ColorReverse   = "\033[7m"
    ColorHidden    = "\033[8m"
    ColorStrike    = "\033[9m"
)


// BackupInfo stores information about a backup file
type BackupInfo struct {
	Path    string
	Name    string
	ModTime time.Time
	Size    int64
	Comment string
}

// BackupMetadata stores metadata for backup files
type BackupMetadata struct {
	Comment   string    `json:"comment"`
	Timestamp time.Time `json:"timestamp"`
	Size      int64     `json:"size"`
	Original  string    `json:"original_file"`
}

// FileStatus represents the status of a file compared to its last backup
type FileStatus int

const (
	FileStatusUnchanged FileStatus = iota
	FileStatusModified
	FileStatusNew
	FileStatusDeleted
)

func (fs FileStatus) String() string {
	switch fs {
	case FileStatusUnchanged:
		return "unchanged"
	case FileStatusModified:
		return "modified"
	case FileStatusNew:
		return "new"
	case FileStatusDeleted:
		return "deleted"
	default:
		return "unknown"
	}
}

func (fs FileStatus) Color() string {
	switch fs {
	case FileStatusUnchanged:
		return ColorGreen
	case FileStatusModified:
		return ColorYellow
	case FileStatusNew:
		return ColorCyan
	case FileStatusDeleted:
		return ColorRed
	default:
		return ColorReset
	}
}

// FileStatusInfo holds file status information
type FileStatusInfo struct {
	Path     string
	RelPath  string
	Status   FileStatus
	Size     int64
	ModTime  time.Time
	IsDir    bool
	Children []*FileStatusInfo
}

// compareFileWithBackup compares a file with its last backup
func compareFileWithBackup(filePath string) (FileStatus, error) {
	// Check if file exists
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return FileStatusDeleted, nil
	}
	if err != nil {
		return FileStatusUnchanged, err
	}

	// Get last backup
	backups, err := listBackups(filePath)
	if err != nil {
		return FileStatusUnchanged, err
	}

	// No backups = new file
	if len(backups) == 0 {
		return FileStatusNew, nil
	}

	// Get last backup content
	lastBackup := backups[0]
	backupContent, err := os.ReadFile(lastBackup.Path)
	if err != nil {
		return FileStatusUnchanged, fmt.Errorf("failed to read backup: %w", err)
	}

	// Get current file content
	currentContent, err := os.ReadFile(filePath)
	if err != nil {
		return FileStatusUnchanged, fmt.Errorf("failed to read file: %w", err)
	}

	// Compare content
	if string(backupContent) == string(currentContent) {
		return FileStatusUnchanged, nil
	}

	return FileStatusModified, nil
}

// buildStatusTree builds a tree with file status information
func buildStatusTree(path string, gitignore *GitIgnore, exceptions map[string]bool, depth int, maxDepth int) (*FileStatusInfo, error) {
	if depth > maxDepth {
		return nil, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	baseName := filepath.Base(path)
	
	if exceptions[baseName] {
		return nil, nil
	}

	if gitignore != nil && gitignore.shouldIgnore(path, info.IsDir()) {
		return nil, nil
	}

	relPath, _ := filepath.Rel(".", path)
	
	node := &FileStatusInfo{
		Path:    path,
		RelPath: relPath,
		IsDir:   info.IsDir(),
		Size:    info.Size(),
		ModTime: info.ModTime(),
		Status:  FileStatusUnchanged,
	}

	// Check status for files only
	if !info.IsDir() {
		status, err := compareFileWithBackup(path)
		if err != nil {
			logger.Printf("Warning: failed to check status for %s: %v", path, err)
			node.Status = FileStatusUnchanged
		} else {
			node.Status = status
		}
	}

	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return node, nil
		}

		for _, entry := range entries {
			childPath := filepath.Join(path, entry.Name())
			childNode, err := buildStatusTree(childPath, gitignore, exceptions, depth+1, maxDepth)
			if err != nil || childNode == nil {
				continue
			}
			node.Children = append(node.Children, childNode)
		}

		sort.Slice(node.Children, func(i, j int) bool {
			if node.Children[i].IsDir != node.Children[j].IsDir {
				return node.Children[i].IsDir
			}
			return node.Children[i].Path < node.Children[j].Path
		})
	}

	return node, nil
}

// printStatusTree prints tree with status information
func printStatusTree(node *FileStatusInfo, prefix string, isLast bool) {
	if node == nil {
		return
	}

	connector := "├── "
	if isLast {
		connector = "└── "
	}

	displayName := filepath.Base(node.Path)
	statusStr := ""
	sizeStr := ""

	if node.IsDir {
		displayName = ColorCyan + displayName + "/" + ColorReset
	} else {
		// Color based on status
		statusColor := node.Status.Color()
		
		if node.Status != FileStatusUnchanged {
			displayName = statusColor + displayName + ColorReset
			statusStr = fmt.Sprintf(" %s[%s]%s", statusColor, node.Status.String(), ColorReset)
		} else {
			displayName = ColorGreen + displayName + ColorReset
		}
		
		sizeStr = ColorGray + " (" + formatSize(node.Size) + ")" + ColorReset
	}

	fmt.Printf("%s%s%s%s%s\n", prefix, connector, displayName, sizeStr, statusStr)

	if node.IsDir && len(node.Children) > 0 {
		childPrefix := prefix
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}

		for i, child := range node.Children {
			printStatusTree(child, childPrefix, i == len(node.Children)-1)
		}
	}
}

// countStatusFiles counts files by status
func countStatusFiles(node *FileStatusInfo) map[FileStatus]int {
	counts := make(map[FileStatus]int)
	
	var count func(*FileStatusInfo)
	count = func(n *FileStatusInfo) {
		if !n.IsDir {
			counts[n.Status]++
		}
		for _, child := range n.Children {
			count(child)
		}
	}
	
	count(node)
	return counts
}

// handleCheckCommand handles the check/status command
func handleCheckCommand(args []string) error {
	// If filename provided, check single file (existing behavior)
	if len(args) > 0 && args[0] != "" && args[0] != "-c" && args[0] != "--check" {
		filename := args[0]
		filePath, err := resolveFilePath(filename)
		if err != nil {
			return err
		}

		status, err := compareFileWithBackup(filePath)
		if err != nil {
			return err
		}

		fmt.Printf("\n%sFile Status:%s %s\n", ColorBold, ColorReset, filePath)
		statusColor := status.Color()
		fmt.Printf("Status: %s%s%s\n", statusColor, status.String(), ColorReset)

		if status == FileStatusModified {
			backups, _ := listBackups(filePath)
			if len(backups) > 0 {
				fmt.Printf("Last backup: %s\n", backups[0].ModTime.Format("2006-01-02 15:04:05"))
			}
		} else if status == FileStatusNew {
			fmt.Printf("No backups found (new file)\n")
		}

		return nil
	}

	// No filename = check all files (like git status)
	fmt.Printf("\n%s📊 PT Status (like git status)%s\n\n", ColorBold+ColorCyan, ColorReset)

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Load gitignore
	gitignore, err := loadGitIgnoreAndPtIgnore(cwd)
	if err != nil {
		logger.Printf("Warning: failed to load .gitignore: %v", err)
	}

	exceptions := make(map[string]bool)
	exceptions[appConfig.BackupDirName] = true

	// Build status tree
	tree, err := buildStatusTree(cwd, gitignore, exceptions, 0, appConfig.MaxSearchDepth)
	if err != nil {
		return fmt.Errorf("failed to build status tree: %w", err)
	}

	if tree == nil {
		return fmt.Errorf("no files to display")
	}

	// Print tree with status
	fmt.Printf("%s%s%s\n", ColorBold, filepath.Base(cwd), ColorReset)
	if tree.IsDir && len(tree.Children) > 0 {
		for i, child := range tree.Children {
			printStatusTree(child, "", i == len(tree.Children)-1)
		}
	}
	fmt.Println()

	// Count and display summary
	counts := countStatusFiles(tree)
	
	hasChanges := counts[FileStatusModified] > 0 || counts[FileStatusNew] > 0 || counts[FileStatusDeleted] > 0
	
	if hasChanges {
		fmt.Printf("%sSummary:%s\n", ColorBold, ColorReset)
		if counts[FileStatusModified] > 0 {
			fmt.Printf("  %s%d modified%s\n", ColorYellow, counts[FileStatusModified], ColorReset)
		}
		if counts[FileStatusNew] > 0 {
			fmt.Printf("  %s%d new%s\n", ColorCyan, counts[FileStatusNew], ColorReset)
		}
		if counts[FileStatusDeleted] > 0 {
			fmt.Printf("  %s%d deleted%s\n", ColorRed, counts[FileStatusDeleted], ColorReset)
		}
		if counts[FileStatusUnchanged] > 0 {
			fmt.Printf("  %s%d unchanged%s\n", ColorGreen, counts[FileStatusUnchanged], ColorReset)
		}
		fmt.Println()
		fmt.Printf("%sUse 'pt commit -m \"message\"' to backup all changes%s\n", ColorCyan, ColorReset)
	} else {
		fmt.Printf("%s✓ No changes detected. All files match their last backups.%s\n", ColorGreen, ColorReset)
	}

	return nil
}

// collectChangedFiles collects all files that need to be backed up
func collectChangedFiles(node *FileStatusInfo, changedFiles *[]string) {
	if !node.IsDir {
		if node.Status == FileStatusModified || node.Status == FileStatusNew {
			*changedFiles = append(*changedFiles, node.Path)
		}
	}
	
	for _, child := range node.Children {
		collectChangedFiles(child, changedFiles)
	}
}

// handleCommitCommand handles the commit command (backup all changed files)
func handleCommitCommand(args []string) error {
	// Parse commit message
	commitMessage := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "-m" || args[i] == "--message" {
			if i+1 < len(args) {
				commitMessage = args[i+1]
				break
			}
		}
	}

	if commitMessage == "" {
		return fmt.Errorf("commit message required. Use: pt commit -m \"your message\"")
	}

	// Add "commit: " prefix to message
	commitMessage = "commit: " + commitMessage

	fmt.Printf("\n%s📦 Committing changes...%s\n\n", ColorBold+ColorCyan, ColorReset)

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Load gitignore
	gitignore, err := loadGitIgnoreAndPtIgnore(cwd)
	if err != nil {
		logger.Printf("Warning: failed to load .gitignore: %v", err)
	}

	exceptions := make(map[string]bool)
	exceptions[appConfig.BackupDirName] = true

	// Build status tree to find changed files
	tree, err := buildStatusTree(cwd, gitignore, exceptions, 0, appConfig.MaxSearchDepth)
	if err != nil {
		return fmt.Errorf("failed to build status tree: %w", err)
	}

	if tree == nil {
		return fmt.Errorf("no files found")
	}

	// Collect all changed files
	var changedFiles []string
	collectChangedFiles(tree, &changedFiles)

	if len(changedFiles) == 0 {
		fmt.Printf("%s✓ No changes to commit. All files are up to date.%s\n", ColorGreen, ColorReset)
		return nil
	}

	fmt.Printf("Files to backup:\n")
	for i, file := range changedFiles {
		relPath, _ := filepath.Rel(cwd, file)
		status, _ := compareFileWithBackup(file)
		statusColor := status.Color()
		fmt.Printf("  %d. %s%s%s %s[%s]%s\n", 
			i+1, ColorGreen, relPath, ColorReset, 
			statusColor, status.String(), ColorReset)
	}
	fmt.Println()

	// Ask for confirmation
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Commit %d file(s) with message \"%s\"? (y/N): ", len(changedFiles), strings.TrimPrefix(commitMessage, "commit: "))
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	
	if input != "y" && input != "yes" {
		fmt.Println("❌ Commit cancelled")
		return nil
	}

	// Backup all changed files
	successCount := 0
	failCount := 0

	for _, file := range changedFiles {
		relPath, _ := filepath.Rel(cwd, file)
		
		// Create backup
		_, err := autoRenameIfExists(file, commitMessage)
		if err != nil {
			fmt.Printf("%s✗%s %s: %v\n", ColorRed, ColorReset, relPath, err)
			failCount++
		} else {
			fmt.Printf("%s✓%s %s\n", ColorGreen, ColorReset, relPath)
			successCount++
		}
	}

	fmt.Println()
	fmt.Printf("%s📦 Commit Summary:%s\n", ColorBold, ColorReset)
	fmt.Printf("  %s✓ %d files backed up%s\n", ColorGreen, successCount, ColorReset)
	if failCount > 0 {
		fmt.Printf("  %s✗ %d files failed%s\n", ColorRed, failCount, ColorReset)
	}
	fmt.Printf("  💬 Message: \"%s\"\n", strings.TrimPrefix(commitMessage, "commit: "))

	return nil
}

type FileSearchResult struct {
	Path     string
	Dir      string
	Size     int64
	ModTime  time.Time
	Depth    int
}

// TreeNode represents a node in the directory tree
type TreeNode struct {
	Name     string
	Path     string
	IsDir    bool
	Size     int64
	Children []*TreeNode
}

// GitIgnore holds gitignore patterns
type GitIgnore struct {
	patterns []string
}

// Logger for audit trail
var logger *log.Logger

// discardWriter implements io.Writer and discards all writes.
type discardWriter struct{}

func (d *discardWriter) Write(p []byte) (n int, err error) {
    return len(p), nil // Discard all data
}

// setupLogger initializes the global logger based on the debugMode flag.
func setupLogger() {
    if debugMode {
        logger = log.New(os.Stderr, "", log.LstdFlags)
    } else {
        logger = log.New(&discardWriter{}, "", log.LstdFlags)
    }
}

func init() {
    // Initialize logger to discard by default in init.
    // It will be set correctly in main() after flag parsing.
    logger = log.New(&discardWriter{}, "", log.LstdFlags)
    Version = loadVersion()
    appConfig = loadConfig()
}

// loadVersion loads version from VERSION file
func loadVersion() string {
	versionPaths := []string{
		"VERSION",
		filepath.Join(filepath.Dir(os.Args[0]), "VERSION"),
		"/usr/local/share/pt/VERSION",
		filepath.Join(os.Getenv("HOME"), ".local", "share", "pt", "VERSION"),
	}
	
	if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
		versionPaths = append(versionPaths, 
			filepath.Join(userProfile, ".pt", "VERSION"),
			filepath.Join(filepath.Dir(os.Args[0]), "VERSION"),
		)
	}
	
	for _, versionPath := range versionPaths {
		data, err := os.ReadFile(versionPath)
		if err == nil {
			content := strings.TrimSpace(string(data))
			
			if strings.HasPrefix(content, "version") {
				parts := strings.SplitN(content, "=", 2)
				if len(parts) == 2 {
					content = strings.TrimSpace(parts[1])
				}
			}
			
			content = strings.Trim(content, `"'`)
			content = strings.TrimPrefix(content, "v")
			
			if content != "" {
				logger.Printf("Version loaded from: %s (%s)", versionPath, content)
				return content
			}
		}
	}
	
	logger.Println("VERSION file not found, using 'dev'")
	return "dev"
}

func getDefaultConfig() *Config {
	return &Config{
		MaxClipboardSize: DefaultMaxClipboardSize,
		MaxBackupCount:   DefaultMaxBackupCount,
		MaxFilenameLen:   DefaultMaxFilenameLen,
		BackupDirName:    DefaultBackupDirName,
		MaxSearchDepth:   DefaultMaxSearchDepth,
	}
}

func findConfigFile() string {
	configNames := []string{"pt.yml", "pt.yaml", ".pt.yml", ".pt.yaml"}
	
	searchPaths := []string{
		".",
		filepath.Join(os.Getenv("HOME"), ".config", "pt"),
		os.Getenv("HOME"),
	}
	
	if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
		searchPaths = append(searchPaths, userProfile, filepath.Join(userProfile, ".pt"))
	}
	
	for _, basePath := range searchPaths {
		for _, configName := range configNames {
			configPath := filepath.Join(basePath, configName)
			if _, err := os.Stat(configPath); err == nil {
				return configPath
			}
		}
	}
	
	return ""
}

func loadConfig() *Config {
	config := getDefaultConfig()
	
	configPath := findConfigFile()
	if configPath == "" {
		logger.Println("No config file found, using defaults")
		return config
	}
	
	logger.Printf("Loading config from: %s", configPath)
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		logger.Printf("Warning: failed to read config file: %v, using defaults", err)
		return config
	}
	
	err = yaml.Unmarshal(data, config)
	if err != nil {
		logger.Printf("Warning: failed to parse config file: %v, using defaults", err)
		return config
	}
	
	if config.MaxClipboardSize <= 0 || config.MaxClipboardSize > 1024*1024*1024 {
		logger.Printf("Warning: invalid max_clipboard_size, using default")
		config.MaxClipboardSize = DefaultMaxClipboardSize
	}
	
	if config.MaxBackupCount <= 0 || config.MaxBackupCount > 10000 {
		logger.Printf("Warning: invalid max_backup_count, using default")
		config.MaxBackupCount = DefaultMaxBackupCount
	}
	
	if config.MaxFilenameLen <= 0 || config.MaxFilenameLen > 1000 {
		logger.Printf("Warning: invalid max_filename_length, using default")
		config.MaxFilenameLen = DefaultMaxFilenameLen
	}
	
	if config.BackupDirName == "" {
		logger.Printf("Warning: empty backup_dir_name, using default")
		config.BackupDirName = DefaultBackupDirName
	}
	
	if config.MaxSearchDepth <= 0 || config.MaxSearchDepth > 100 {
		logger.Printf("Warning: invalid max_search_depth, using default")
		config.MaxSearchDepth = DefaultMaxSearchDepth
	}
	
	logger.Printf("Config loaded successfully: clipboard=%dMB, backups=%d, depth=%d", 
		config.MaxClipboardSize/(1024*1024), config.MaxBackupCount, config.MaxSearchDepth)
	
	return config
}

// findPTRoot searches for .pt directory in current and parent directories (like .git)
// It starts from the given path and walks up the directory tree until it finds .pt or reaches root
func findPTRoot(startPath string) (string, error) {
	// If startPath is a file, get its directory
	info, err := os.Stat(startPath)
	if err == nil && !info.IsDir() {
		startPath = filepath.Dir(startPath)
	}

	absPath, err := filepath.Abs(startPath)
	if err != nil {
		return "", err
	}

	current := absPath
	
	// Search up the directory tree until we find .pt or reach filesystem root
	for {
		ptDir := filepath.Join(current, appConfig.BackupDirName)
		if info, err := os.Stat(ptDir); err == nil && info.IsDir() {
			logger.Printf("Found %s directory at: %s", appConfig.BackupDirName, ptDir)
			return ptDir, nil
		}

		parent := filepath.Dir(current)
		
		// Reached filesystem root (parent == current means we can't go up anymore)
		if parent == current {
			break
		}
		
		current = parent
	}

	// No .pt directory found in any parent
	logger.Printf("No %s directory found in tree from: %s", appConfig.BackupDirName, absPath)
	return "", nil
}

// ensurePTDir creates .pt directory if it doesn't exist
// Returns the absolute path to the .pt directory (could be in parent dir)
// This function mimics git behavior - searches upward for existing .pt
func ensurePTDir(filePath string) (string, error) {
	// Get directory of the target file (or use current dir if it's already a dir)
	dir := filePath
	info, err := os.Stat(filePath)
	if err == nil && !info.IsDir() {
		dir = filepath.Dir(filePath)
	} else if err != nil {
		// File doesn't exist yet, get its directory
		dir = filepath.Dir(filePath)
	}
	
	if dir == "." || dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Try to find existing .pt directory by walking up the tree
	ptRoot, err := findPTRoot(dir)
	if err != nil {
		return "", err
	}

	// If found in parent directory, use that (like git)
	if ptRoot != "" {
		ptParent := filepath.Dir(ptRoot)
		cwd, _ := os.Getwd()
		relPath, _ := filepath.Rel(cwd, ptParent)
		if relPath != "" && relPath != "." {
			logger.Printf("Using existing %s from parent: %s", appConfig.BackupDirName, ptParent)
			fmt.Printf("📁 Using %s from: %s%s/%s\n", appConfig.BackupDirName, ColorCyan, relPath, ColorReset)
		}
		return ptRoot, nil
	}

	// No .pt directory found in tree, create one in the file's immediate directory
	// Get the absolute path of the directory where we'll create .pt
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	
	ptDir := filepath.Join(absDir, appConfig.BackupDirName)

	// Check if .pt directory exists at this level
	info, err = os.Stat(ptDir)
	if os.IsNotExist(err) {
		// Create .pt directory with appropriate permissions
		err = os.MkdirAll(ptDir, 0755)
		if err != nil {
			return "", fmt.Errorf("failed to create %s directory: %w", appConfig.BackupDirName, err)
		}
		logger.Printf("Created %s directory: %s", appConfig.BackupDirName, ptDir)
		fmt.Printf("📁 Created %s directory: %s\n", appConfig.BackupDirName, ptDir)
		
		// Create .gitignore to ignore .pt directory
		createPTGitignore(absDir)
	} else if err != nil {
		return "", fmt.Errorf("failed to check %s directory: %w", appConfig.BackupDirName, err)
	} else if !info.IsDir() {
		return "", fmt.Errorf("%s exists but is not a directory: %s", appConfig.BackupDirName, ptDir)
	}

	return ptDir, nil
}

// createPTGitignore creates/updates .gitignore to exclude .pt directory
func createPTGitignore(dir string) {
	gitignorePath := filepath.Join(dir, ".gitignore")
	
	// Check if .gitignore exists
	content, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return // Skip on error
	}

	gitignoreContent := string(content)
	
	// Check if .pt is already ignored
	ptPattern := appConfig.BackupDirName + "/"
	if strings.Contains(gitignoreContent, ptPattern) || strings.Contains(gitignoreContent, appConfig.BackupDirName+"\n") {
		return // Already ignored
	}

	// Append .pt to .gitignore
	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return // Skip on error
	}
	defer f.Close()

	// Add newline if file doesn't end with one
	if len(content) > 0 && content[len(content)-1] != '\n' {
		f.WriteString("\n")
	}

	f.WriteString("# PT backup directory\n")
	f.WriteString(ptPattern + "\n")
	
	logger.Printf("Added %s to .gitignore", ptPattern)
}

// getRelativePath gets relative path from .pt root to file
func getRelativePath(ptRoot, filePath string) (string, error) {
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return "", err
	}

	// Get the directory containing .pt
	ptParent := filepath.Dir(ptRoot)
	
	relPath, err := filepath.Rel(ptParent, absFilePath)
	if err != nil {
		return "", err
	}

	return relPath, nil
}

// getBackupDir returns the backup directory path for a file within .pt
// The backup directory name is based on the file's relative path from .pt parent
// Examples:
//   ./main.go          -> .pt/main.go/
//   ./pt/main.go       -> .pt/pt_main.go/
//   ./src/lib/util.go  -> .pt/src_lib_util.go/
func getBackupDir(ptRoot, filePath string) (string, error) {
	relPath, err := getRelativePath(ptRoot, filePath)
	if err != nil {
		return "", err
	}

	// Clean the relative path
	relPath = filepath.Clean(relPath)
	
	// Get the base filename
	baseName := filepath.Base(relPath)
	
	// Get the directory part (if any)
	dirPart := filepath.Dir(relPath)
	
	var backupSubdir string
	
	// If file is directly in .pt parent (no subdirectory)
	if dirPart == "." {
		// Just use the filename
		backupSubdir = baseName
	} else {
		// File is in a subdirectory, preserve the path structure
		// Replace path separators with underscores
		// e.g., pt/main.go -> pt_main.go
		//       src/lib/util.go -> src_lib_util.go
		fullPath := relPath
		fullPath = strings.ReplaceAll(fullPath, string(os.PathSeparator), "_")
		fullPath = strings.ReplaceAll(fullPath, "/", "_")  // Unix
		fullPath = strings.ReplaceAll(fullPath, "\\", "_") // Windows
		backupSubdir = fullPath
	}
	
	backupDir := filepath.Join(ptRoot, backupSubdir)
	
	logger.Printf("Backup dir for %s: %s (relative: %s)", filePath, backupDir, relPath)
	
	return backupDir, nil
}

// loadIgnorePatterns loads patterns from .ptignore and .gitignore
func loadIgnorePatterns(startPath string) []string {
	patterns := make([]string, 0)
	
	// Try to find .pt root first
	ptRoot, _ := findPTRoot(startPath)
	var searchDir string
	if ptRoot != "" {
		searchDir = filepath.Dir(ptRoot)
	} else {
		searchDir = startPath
	}

	// Load .ptignore (higher priority)
	ptignorePath := filepath.Join(searchDir, ".ptignore")
	if content, err := os.ReadFile(ptignorePath); err == nil {
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				patterns = append(patterns, line)
			}
		}
		logger.Printf("Loaded %d patterns from .ptignore", len(patterns))
	}

	// Load .gitignore
	gitignorePath := filepath.Join(searchDir, ".gitignore")
	if content, err := os.ReadFile(gitignorePath); err == nil {
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				patterns = append(patterns, line)
			}
		}
		logger.Printf("Loaded patterns from .gitignore")
	}

	// Always ignore .pt directory
	patterns = append(patterns, appConfig.BackupDirName, appConfig.BackupDirName+"/")

	return patterns
}

// shouldIgnore checks if a path matches ignore patterns
func shouldIgnore(path string, patterns []string) bool {
	baseName := filepath.Base(path)
	
	for _, pattern := range patterns {
		// Simple pattern matching
		if pattern == baseName {
			return true
		}
		if strings.HasSuffix(pattern, "/") && baseName == strings.TrimSuffix(pattern, "/") {
			return true
		}
		if strings.Contains(path, pattern) {
			return true
		}
	}
	
	return false
}

func generateSampleConfig(path string) error {
	config := getDefaultConfig()
	
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	header := `# PT Configuration File
# This file configures the behavior of the PT tool
# All values are optional - if not specified, defaults will be used

# Maximum clipboard content size in bytes (default: 104857600 = 100MB)
# Range: 1 - 1073741824 (1GB)
`
	
	fullContent := header + string(data)
	
	err = os.WriteFile(path, []byte(fullContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

func handleConfigCommand(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("config subcommand required: 'init', 'show', or 'path'")
	}
	
	subcommand := args[0]
	
	switch subcommand {
	case "init":
		var configPath string
		if len(args) > 1 {
			configPath = args[1]
		} else {
			configPath = "pt.yml"
		}
		
		if _, err := os.Stat(configPath); err == nil {
			fmt.Printf("%s⚠️  Warning: Config file already exists: %s%s\n", ColorYellow, configPath, ColorReset)
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Overwrite? (y/N): ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(strings.ToLower(input))
			if input != "y" && input != "yes" {
				fmt.Println("❌ Cancelled")
				return nil
			}
		}
		
		err := generateSampleConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to generate config: %w", err)
		}
		
		fmt.Printf("✅ Sample config file created: %s%s%s\n", ColorGreen, configPath, ColorReset)
		fmt.Println("📝 Edit this file to customize PT behavior")
		
	case "show":
		fmt.Printf("\n%sCurrent PT Configuration:%s\n\n", ColorBold, ColorReset)
		fmt.Printf("%sMax Clipboard Size:%s %d bytes (%.1f MB)\n", 
			ColorCyan, ColorReset, appConfig.MaxClipboardSize, float64(appConfig.MaxClipboardSize)/(1024*1024))
		fmt.Printf("%sMax Backup Count:%s %d\n", ColorCyan, ColorReset, appConfig.MaxBackupCount)
		fmt.Printf("%sMax Filename Length:%s %d characters\n", ColorCyan, ColorReset, appConfig.MaxFilenameLen)
		fmt.Printf("%sBackup Directory:%s %s/ (Git-like structure)\n", ColorCyan, ColorReset, appConfig.BackupDirName)
		fmt.Printf("%sMax Search Depth:%s %d levels\n\n", ColorCyan, ColorReset, appConfig.MaxSearchDepth)
		
		configPath := findConfigFile()
		if configPath != "" {
			fmt.Printf("%sConfig loaded from:%s %s\n", ColorGray, ColorReset, configPath)
		} else {
			fmt.Printf("%sUsing default configuration (no config file found)%s\n", ColorGray, ColorReset)
		}
		
	case "path":
		configPath := findConfigFile()
		if configPath != "" {
			fmt.Printf("📄 Config file: %s%s%s\n", ColorGreen, configPath, ColorReset)
		} else {
			fmt.Printf("%sℹ️  No config file found%s\n", ColorGray, ColorReset)
			fmt.Println("\nSearched in:")
			fmt.Println("  • ./pt.yml or ./pt.yaml")
			fmt.Println("  • ~/.config/pt/pt.yml or ~/.config/pt/pt.yaml")
			fmt.Println("  • ~/pt.yml or ~/pt.yaml")
			fmt.Printf("\n%sCreate one with:%s pt config init\n", ColorCyan, ColorReset)
		}
		
	default:
		return fmt.Errorf("unknown config subcommand: %s (use 'init', 'show', or 'path')", subcommand)
	}
	
	return nil
}

func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func saveBackupMetadata(backupPath, comment, originalFile string, size int64) error {
	metadataPath := backupPath + ".meta.json"
	
	metadata := BackupMetadata{
		Comment:   comment,
		Timestamp: time.Now(),
		Size:      size,
		Original:  originalFile,
	}
	
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	
	err = os.WriteFile(metadataPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}
	
	return nil
}

func loadBackupMetadata(backupPath string) (string, error) {
	metadataPath := backupPath + ".meta.json"
	
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	
	var metadata BackupMetadata
	err = json.Unmarshal(data, &metadata)
	if err != nil {
		return "", err
	}
	
	return metadata.Comment, nil
}

// func loadGitIgnore(rootPath string) (*GitIgnore, error) {
// 	gitignorePath := filepath.Join(rootPath, ".gitignore")
// 	gi := &GitIgnore{patterns: make([]string, 0)}
	
// 	file, err := os.Open(gitignorePath)
// 	if err != nil {
// 		if os.IsNotExist(err) {
// 			return gi, nil
// 		}
// 		return nil, err
// 	}
// 	defer file.Close()

// 	scanner := bufio.NewScanner(file)
// 	for scanner.Scan() {
// 		line := strings.TrimSpace(scanner.Text())
// 		if line == "" || strings.HasPrefix(line, "#") {
// 			continue
// 		}
// 		gi.patterns = append(gi.patterns, line)
// 	}

// 	return gi, scanner.Err()
// }

// loadGitIgnoreAndPtIgnore loads patterns from .gitignore and .ptignore in the root path
func loadGitIgnoreAndPtIgnore(rootPath string) (*GitIgnore, error) {
    gitignorePath := filepath.Join(rootPath, ".gitignore")
    ptignorePath := filepath.Join(rootPath, ".ptignore")

    gi := &GitIgnore{patterns: make([]string, 0)}

    // Load .gitignore
    file, err := os.Open(gitignorePath)
    if err != nil {
        if !os.IsNotExist(err) {
            logger.Printf("Warning: failed to read .gitignore: %v", err)
        }
        // Continue to load .ptignore even if .gitignore fails
    } else {
        defer file.Close()
        scanner := bufio.NewScanner(file)
        for scanner.Scan() {
            line := strings.TrimSpace(scanner.Text())
            if line == "" || strings.HasPrefix(line, "#") {
                continue
            }
            gi.patterns = append(gi.patterns, line)
        }
        if err := scanner.Err(); err != nil {
            logger.Printf("Warning: error reading .gitignore: %v", err)
        }
    }

    // Load .ptignore
    ptFile, err := os.Open(ptignorePath)
    if err != nil {
        if !os.IsNotExist(err) {
            logger.Printf("Warning: failed to read .ptignore: %v", err)
        }
        // Continue even if .ptignore fails
    } else {
        defer ptFile.Close()
        scanner := bufio.NewScanner(ptFile)
        for scanner.Scan() {
            line := strings.TrimSpace(scanner.Text())
            if line == "" || strings.HasPrefix(line, "#") {
                continue
            }
            gi.patterns = append(gi.patterns, line)
        }
        if err := scanner.Err(); err != nil {
            logger.Printf("Warning: error reading .ptignore: %v", err)
        }
    }

    return gi, nil
}

func (gi *GitIgnore) shouldIgnore(path string, isDir bool) bool {
	baseName := filepath.Base(path)
	
	// Always ignore .pt directory
	if baseName == appConfig.BackupDirName {
		return true
	}

	// Always ignore .git directory
    if baseName == ".git" {
        return true
    }
	
	for _, pattern := range gi.patterns {
		if strings.HasSuffix(pattern, "/") {
			dirPattern := strings.TrimSuffix(pattern, "/")
			if isDir && (baseName == dirPattern || strings.HasPrefix(baseName, dirPattern)) {
				return true
			}
			continue
		}
		
		if strings.Contains(pattern, "*") {
			matched, _ := filepath.Match(pattern, baseName)
			if matched {
				return true
			}
			continue
		}
		
		if baseName == pattern {
			return true
		}
		
		if strings.Contains(path, "/"+pattern+"/") || strings.Contains(path, "\\"+pattern+"\\") {
			return true
		}
	}
	
	return false
}

func buildTree(path string, gitignore *GitIgnore, exceptions map[string]bool, depth int, maxDepth int) (*TreeNode, error) {
	if depth > maxDepth {
		return nil, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	baseName := filepath.Base(path)
	
	if exceptions[baseName] {
		return nil, nil
	}

	if gitignore != nil && gitignore.shouldIgnore(path, info.IsDir()) {
		return nil, nil
	}

	node := &TreeNode{
		Name:  baseName,
		Path:  path,
		IsDir: info.IsDir(),
		Size:  info.Size(),
	}

	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return node, nil
		}

		for _, entry := range entries {
			childPath := filepath.Join(path, entry.Name())
			childNode, err := buildTree(childPath, gitignore, exceptions, depth+1, maxDepth)
			if err != nil || childNode == nil {
				continue
			}
			node.Children = append(node.Children, childNode)
		}

		sort.Slice(node.Children, func(i, j int) bool {
			if node.Children[i].IsDir != node.Children[j].IsDir {
				return node.Children[i].IsDir
			}
			return node.Children[i].Name < node.Children[j].Name
		})
	}

	return node, nil
}

func printTree(node *TreeNode, prefix string, isLast bool, showSize bool) {
	if node == nil {
		return
	}

	connector := "├── "
	if isLast {
		connector = "└── "
	}

	displayName := node.Name
	if node.IsDir {
		displayName = ColorCyan + displayName + "/" + ColorReset
	} else {
		displayName = ColorGreen + displayName + ColorReset
	}

	sizeStr := ""
	if showSize && !node.IsDir {
		sizeStr = ColorGray + " (" + formatSize(node.Size) + ")" + ColorReset
	}

	fmt.Printf("%s%s%s%s\n", prefix, connector, displayName, sizeStr)

	if node.IsDir && len(node.Children) > 0 {
		childPrefix := prefix
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}

		for i, child := range node.Children {
			printTree(child, childPrefix, i == len(node.Children)-1, showSize)
		}
	}
}

func handleTreeCommand(args []string) error {
	exceptions := make(map[string]bool)
	startPath := "."
	
	i := 0
	for i < len(args) {
		if args[i] == "-e" || args[i] == "--exception" {
			if i+1 >= len(args) {
				return fmt.Errorf("-e/--exception requires a value")
			}
			i++
			for _, exc := range strings.Split(args[i], ",") {
				exceptions[strings.TrimSpace(exc)] = true
			}
			i++
		} else {
			startPath = args[i]
			i++
		}
	}

	absPath, err := filepath.Abs(startPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("path does not exist: %w", err)
	}

	var gitignore *GitIgnore
	if info.IsDir() {
		gitignore, err = loadGitIgnoreAndPtIgnore(absPath)
		if err != nil {
			logger.Printf("Warning: failed to load .gitignore: %v", err)
		}
	}

	tree, err := buildTree(absPath, gitignore, exceptions, 0, appConfig.MaxSearchDepth)
	if err != nil {
		return fmt.Errorf("failed to build tree: %w", err)
	}

	if tree == nil {
		return fmt.Errorf("no files to display")
	}

	fmt.Printf("\n%s%s%s\n", ColorBold, tree.Name, ColorReset)
	if tree.IsDir && len(tree.Children) > 0 {
		for i, child := range tree.Children {
			printTree(child, "", i == len(tree.Children)-1, true)
		}
	}
	fmt.Println()

	fileCount := 0
	dirCount := 0
	var totalSize int64

	var countNodes func(*TreeNode)
	countNodes = func(n *TreeNode) {
		if n.IsDir {
			dirCount++
			for _, child := range n.Children {
				countNodes(child)
			}
		} else {
			fileCount++
			totalSize += n.Size
		}
	}
	countNodes(tree)

	fmt.Printf("%s%d directories, %d files, %s total%s\n", 
		ColorGray, dirCount, fileCount, formatSize(totalSize), ColorReset)

	if len(exceptions) > 0 {
		excList := make([]string, 0, len(exceptions))
		for exc := range exceptions {
			excList = append(excList, exc)
		}
		fmt.Printf("%sExceptions: %s%s\n", ColorGray, strings.Join(excList, ", "), ColorReset)
	}

	if gitignore != nil && len(gitignore.patterns) > 0 {
		fmt.Printf("%sUsing .gitignore (%d patterns) + %s is always excluded%s\n", 
			ColorGray, len(gitignore.patterns), appConfig.BackupDirName, ColorReset)
	}

	return nil
}

// parsing comment for handleRemoveCommand
func handleRemoveCommand(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("filename required for remove command")
	}

	filename := args[0]
	comment := ""
	
	for i := 1; i < len(args); i++ {
		if args[i] == "-m" || args[i] == "--message" {
			if i+1 >= len(args) {
				return fmt.Errorf("-m/--message requires a value")
			}
			i++
			comment = args[i]
			break
		}
	}

	filePath, err := resolveFilePath(filename)
	if err != nil {
		return err
	}

	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", filePath)
		}
		return fmt.Errorf("failed to check file: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("cannot remove directories, only files")
	}

	if info.Size() > 0 {
		if comment == "" {
			comment = "Deleted file backup"
		}
		_, err = autoRenameIfExists(filePath, comment)
		if err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	err = os.Remove(filePath)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	logger.Printf("File deleted: %s (%d bytes)", filePath, len(content))
	fmt.Printf("🗑️  File deleted: %s\n", filePath)

	emptyFile, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create empty placeholder: %w", err)
	}
	emptyFile.Close()

	logger.Printf("Created empty placeholder: %s", filePath)
	fmt.Printf("📄 Created empty placeholder: %s\n", filePath)
	fmt.Printf("ℹ️  Original content (%d bytes) backed up to %s/\n", len(content), appConfig.BackupDirName)

	return nil
}

func searchFileRecursive(filename string, maxDepth int) ([]FileSearchResult, error) {
	results := make([]FileSearchResult, 0)
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Load ignore patterns
	ignorePatterns := loadIgnorePatterns(cwd)

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

	err = filepath.Walk(cwd, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Check ignore patterns
		if shouldIgnore(path, ignorePatterns) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(cwd, path)
		if err != nil {
			return nil
		}
		depth := len(strings.Split(relPath, string(os.PathSeparator))) - 1

		if depth > maxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !info.IsDir() && info.Name() == filename {
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

func printFileSearchResults(results []FileSearchResult) {
	const (
		col1Width = 60
		col2Width = 19
		col3Width = 12
	)

	fmt.Printf("\n%s🔍 Found %d file(s):%s\n\n", ColorCyan, len(results), ColorReset)

	fmt.Printf("%s┌%s┬%s┬%s┐%s\n",
		ColorGray,
		strings.Repeat("─", col1Width+2),
		strings.Repeat("─", col2Width+2),
		strings.Repeat("─", col3Width+2),
		ColorReset)

	fmt.Printf("%s│%s %s%s%-*s%s %s│%s %s%s%-*s%s %s│%s %s%s%*s%s %s│%s\n",
		ColorGray, ColorReset,
		ColorBold, ColorYellow, col1Width, "Path", ColorReset,
		ColorGray, ColorReset,
		ColorBold, ColorYellow, col2Width, "Modified", ColorReset,
		ColorGray, ColorReset,
		ColorBold, ColorYellow, col3Width, "Size", ColorReset,
		ColorGray, ColorReset)

	fmt.Printf("%s├%s┼%s┼%s┤%s\n",
		ColorGray,
		strings.Repeat("─", col1Width+2),
		strings.Repeat("─", col2Width+2),
		strings.Repeat("─", col3Width+2),
		ColorReset)

	for i, result := range results {
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
		sizeStr := formatSize(result.Size)

		fmt.Printf("%s│%s %s%3d. %-*s%s %s│%s %-*s %s│%s %*s %s│%s\n",
			ColorGray, ColorReset,
			ColorGreen, i+1, maxPathLen, displayPath, ColorReset,
			ColorGray, ColorReset,
			col2Width, modTime,
			ColorGray, ColorReset,
			col3Width, sizeStr,
			ColorGray, ColorReset)
	}

	fmt.Printf("%s└%s┴%s┴%s┘%s\n\n",
		ColorGray,
		strings.Repeat("─", col1Width+2),
		strings.Repeat("─", col2Width+2),
		strings.Repeat("─", col3Width+2),
		ColorReset)
}

func resolveFilePath(filename string) (string, error) {
	if info, err := os.Stat(filename); err == nil && !info.IsDir() {
		absPath, _ := filepath.Abs(filename)
		return absPath, nil
	}

	logger.Printf("File not found in current directory, searching recursively...")
	fmt.Printf("%s🔍 Searching for '%s' in subdirectories...%s\n", ColorBlue, filename, ColorReset)

	results, err := searchFileRecursive(filename, appConfig.MaxSearchDepth)
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

func checkDeltaInstalled() bool {
	_, err := exec.LookPath("delta")
	return err == nil
}

func runDelta(file1, file2 string) error {
	if !checkDeltaInstalled() {
		return fmt.Errorf("delta is not installed. Install it from: https://github.com/dandavison/delta")
	}

	cmd := exec.Command("delta", file1, file2)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	
	// Delta exit code 1 is NORMAL when files are different
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return nil
			}
		}
		return err
	}

	return nil
}

func handleDiffCommand(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("filename required for diff command")
	}

	filename := args[0]
	useLast := len(args) > 1 && args[1] == "--last"

	filePath, err := resolveFilePath(filename)
	if err != nil {
		return err
	}

	backups, err := listBackups(filePath)
	if err != nil {
		return err
	}

	if len(backups) == 0 {
		return fmt.Errorf("no backups found for: %s (check %s/ directory)", filePath, appConfig.BackupDirName)
	}

	var selectedBackup BackupInfo

	if useLast {
		selectedBackup = backups[0]
		fmt.Printf("%s📊 Comparing with last backup: %s%s\n\n", ColorCyan, selectedBackup.Name, ColorReset)
	} else {
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

	err = runDelta(selectedBackup.Path, filePath)
	if err != nil {
		return fmt.Errorf("delta execution failed: %w", err)
	}

	return nil
}

func validatePath(filePath string) error {
	if filePath == "" {
		return fmt.Errorf("filename cannot be empty")
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	cleanPath := filepath.Clean(filePath)
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal not allowed")
	}

	if len(filepath.Base(filePath)) > appConfig.MaxFilenameLen {
		return fmt.Errorf("filename too long (max %d characters)", appConfig.MaxFilenameLen)
	}

	systemDirs := []string{"/etc", "/sys", "/proc", "/dev", "C:\\Windows", "C:\\System32"}
	for _, sysDir := range systemDirs {
		if strings.HasPrefix(absPath, sysDir) {
			return fmt.Errorf("writing to system directories not allowed")
		}
	}

	return nil
}

func checkDiskSpace(path string, requiredSize int64) error {
	dir := filepath.Dir(path)
	if dir == "." {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	testFile := filepath.Join(dir, ".pt_test_"+generateShortID())
	f, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("no write permission in directory: %w", err)
	}
	f.Close()
	os.Remove(testFile)

	return nil
}

func generateShortID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generateUniqueBackupName(filePath string) string {
	baseName := filepath.Base(filePath)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)

	timestamp := time.Now().Format("20060102_150405.000000")
	timestamp = strings.ReplaceAll(timestamp, ".", "")

	uniqueID := fmt.Sprintf("%d_%s", os.Getpid(), generateShortID())

	return fmt.Sprintf("%s_%s.%s.%s", nameWithoutExt, strings.TrimPrefix(ext, "."), timestamp, uniqueID)
}

func getClipboardText() (string, error) {
	text, err := clipboard.ReadAll()
	if err != nil {
		return "", fmt.Errorf("failed to read clipboard: %w", err)
	}

	if len(text) > appConfig.MaxClipboardSize {
		return "", fmt.Errorf("clipboard content too large (max %dMB)", appConfig.MaxClipboardSize/(1024*1024))
	}

	return text, nil
}

func autoRenameIfExists(filePath, comment string) (string, error) {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return filePath, nil
	}
	if err != nil {
		return filePath, fmt.Errorf("failed to check file: %w", err)
	}

	if info.Size() == 0 {
		logger.Printf("Skipping backup of empty file: %s", filePath)
		return filePath, nil
	}

	// Ensure .pt directory exists (searches parent dirs)
	ptRoot, err := ensurePTDir(filePath)
	if err != nil {
		return filePath, err
	}

	backupFileName := generateUniqueBackupName(filePath)
	
	// Get backup directory for this file within .pt
	backupDir, err := getBackupDir(ptRoot, filePath)
	if err != nil {
		return filePath, err
	}

	// Create subdirectory if needed
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return filePath, fmt.Errorf("failed to create backup subdirectory: %w", err)
	}

	backupPath := filepath.Join(backupDir, backupFileName)

	content, err := os.ReadFile(filePath)
	if err != nil {
		return filePath, fmt.Errorf("failed to read file for backup: %w", err)
	}

	err = os.WriteFile(backupPath, content, 0644)
	if err != nil {
		return filePath, fmt.Errorf("failed to create backup: %w", err)
	}

	err = saveBackupMetadata(backupPath, comment, filePath, info.Size())
	if err != nil {
		logger.Printf("Warning: failed to save backup metadata: %v", err)
	}

	logger.Printf("Backup created: %s -> %s", filePath, backupPath)
	if comment != "" {
		logger.Printf("Backup comment: %s", comment)
		fmt.Printf("📦 Backup created: %s%s%s\n", ColorGreen, backupFileName, ColorReset)
		fmt.Printf("💬 Comment: \"%s%s%s\"\n", ColorCyan, comment, ColorReset)
	} else {
		fmt.Printf("📦 Backup created: %s%s%s\n", ColorGreen, backupFileName, ColorReset)
	}

	return filePath, nil
}

func writeFile(filePath string, data string, appendMode bool, checkMode bool, comment string) error {
	if err := validatePath(filePath); err != nil {
		return err
	}

	if checkMode && !appendMode {
		if existingData, err := os.ReadFile(filePath); err == nil {
			if string(existingData) == data {
				logger.Printf("Content identical, skipping write: %s", filePath)
				fmt.Printf("ℹ️  Content identical to current file, no changes needed\n")
				fmt.Printf("📄 File: %s\n", filePath)
				return nil
			}
			fmt.Printf("🔍 Content differs, proceeding with backup and write\n")
		}
	}

	if err := checkDiskSpace(filePath, int64(len(data))); err != nil {
		return err
	}

	if !appendMode {
		var err error
		filePath, err = autoRenameIfExists(filePath, comment)
		if err != nil {
			return err
		}
	}

	var flag int
	if appendMode {
		flag = os.O_APPEND | os.O_CREATE | os.O_WRONLY
	} else {
		flag = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	}

	file, err := os.OpenFile(filePath, flag, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	n, err := file.WriteString(data)
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	if n != len(data) {
		return fmt.Errorf("incomplete write: wrote %d bytes, expected %d", n, len(data))
	}

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

func listBackups(filePath string) ([]BackupInfo, error) {
	if err := validatePath(filePath); err != nil {
		return nil, err
	}

	// Get absolute path of the file
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}
	
	logger.Printf("Listing backups for: %s", absFilePath)
	
	// Get the directory of the file (or use current if file doesn't exist yet)
	dir := filepath.Dir(absFilePath)
	
	// Find .pt root (searches parent directories like git)
	ptRoot, err := findPTRoot(dir)
	if err != nil {
		return nil, err
	}

	if ptRoot == "" {
		// No .pt directory exists yet in the entire tree
		logger.Printf("No .pt directory found in tree")
		return []BackupInfo{}, nil
	}

	logger.Printf("Found .pt root: %s", ptRoot)

	// Get file basename and extension once
	fileBaseName := filepath.Base(absFilePath)
	fileExt := filepath.Ext(fileBaseName)
	fileNameWithoutExt := strings.TrimSuffix(fileBaseName, fileExt)
	fileExtWithoutDot := strings.TrimPrefix(fileExt, ".")
	
	// Get backup directory for this file within .pt
	backupDir, err := getBackupDir(ptRoot, absFilePath)
	if err != nil {
		return nil, err
	}

	logger.Printf("Expected backup directory: %s", backupDir)

	// Check if expected backup directory exists
	backupDirExists := false
	if stat, err := os.Stat(backupDir); err == nil && stat.IsDir() {
		backupDirExists = true
		logger.Printf("Backup directory exists: %s", backupDir)
	} else {
		logger.Printf("Backup directory does not exist: %s (error: %v)", backupDir, err)
	}

	// If expected directory doesn't exist, try fallback to base filename only
	if !backupDirExists {
		alternateBackupDir := filepath.Join(ptRoot, fileBaseName)
		
		logger.Printf("Trying alternate backup directory (base filename only): %s", alternateBackupDir)
		
		if stat, err := os.Stat(alternateBackupDir); err == nil && stat.IsDir() {
			logger.Printf("Found backups using base filename: %s", alternateBackupDir)
			fmt.Printf("%sℹ️  Note: Using backups from '%s/' (file may have been moved)%s\n", 
				ColorYellow, fileBaseName, ColorReset)
			backupDir = alternateBackupDir
			backupDirExists = true
		} else {
			logger.Printf("Alternate backup directory also not found: %s (error: %v)", alternateBackupDir, err)
		}
	}

	// If still no backup directory found, return empty
	if !backupDirExists {
		logger.Printf("No backup directory found for file")
		return []BackupInfo{}, nil
	}

	// Pattern for backup files: filename_ext.timestamp...
	pattern := fmt.Sprintf("%s_%s.", fileNameWithoutExt, fileExtWithoutDot)
	
	logger.Printf("Looking for backup files with pattern: %s", pattern)

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		logger.Printf("Failed to read backup directory: %v", err)
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	logger.Printf("Found %d entries in backup directory", len(entries))

	backups := make([]BackupInfo, 0)

	for _, entry := range entries {
		if entry.IsDir() {
			logger.Printf("Skipping directory: %s", entry.Name())
			continue
		}

		name := entry.Name()
		
		if strings.HasSuffix(name, ".meta.json") {
			logger.Printf("Skipping metadata file: %s", name)
			continue
		}
		
		logger.Printf("Checking file: %s against pattern: %s", name, pattern)
		
		if !strings.HasPrefix(name, pattern) {
			logger.Printf("Skipping (doesn't match pattern '%s'): %s", pattern, name)
			continue
		}

		timestamp := strings.TrimPrefix(name, pattern)
		
		logger.Printf("Extracted timestamp: %s (length: %d)", timestamp, len(timestamp))
		
		if len(timestamp) < 20 {
			logger.Printf("Skipping (timestamp too short): %s", name)
			continue
		}

		timestampPart := timestamp
		if len(timestampPart) > 30 {
			timestampPart = timestampPart[:30]
		}
		
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

		info, err := entry.Info()
		if err != nil {
			logger.Printf("Warning: failed to get info for %s: %v", name, err)
			continue
		}

		backupPath := filepath.Join(backupDir, name)
		comment, err := loadBackupMetadata(backupPath)
		if err != nil && !os.IsNotExist(err) {
			logger.Printf("Warning: failed to load metadata for %s: %v", name, err)
		}

		logger.Printf("Found valid backup: %s (comment: %s)", name, comment)
		backups = append(backups, BackupInfo{
			Path:    backupPath,
			Name:    name,
			ModTime: info.ModTime(),
			Size:    info.Size(),
			Comment: comment,
		})
	}

	if len(backups) == 0 {
		logger.Printf("No valid backups found matching pattern: %s", pattern)
		return backups, nil
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].ModTime.After(backups[j].ModTime)
	})

	if len(backups) > appConfig.MaxBackupCount {
		backups = backups[:appConfig.MaxBackupCount]
	}

	logger.Printf("Returning %d backup(s)", len(backups))
	return backups, nil
}

func printBackupTable(filePath string, backups []BackupInfo) {
	const (
		col1Width = 40  // More width for filename
		col2Width = 19
		col3Width = 12
		col4Width = 30  // Smaller for comments
	)

	// Find .pt root to show in message
	dir := filepath.Dir(filePath)
	ptRoot, _ := findPTRoot(dir)
	ptLocation := appConfig.BackupDirName
	if ptRoot != "" {
		relPT, _ := filepath.Rel(".", ptRoot)
		if relPT != "" {
			ptLocation = relPT
		}
	}

	fmt.Printf("\n%s📂 Backup files for '%s%s%s%s'%s\n",
		ColorCyan, ColorBold, filePath, ColorReset, ColorCyan, ColorReset)
	fmt.Printf("%sTotal: %d backup(s) (stored in %s/)%s\n\n", 
		ColorGray, len(backups), ptLocation, ColorReset)

	fmt.Printf("%s┌%s┬%s┬%s┬%s┐%s\n",
		ColorGray,
		strings.Repeat("─", col1Width+2),
		strings.Repeat("─", col2Width+2),
		strings.Repeat("─", col3Width+2),
		strings.Repeat("─", col4Width+2),
		ColorReset)

	fmt.Printf("%s│%s %s%s%-*s%s %s│%s %s%s%-*s%s %s│%s %s%s%*s%s %s│%s %s%s%-*s%s %s│%s\n",
		ColorGray, ColorReset,
		ColorBold, ColorYellow, col1Width, "File Name", ColorReset,
		ColorGray, ColorReset,
		ColorBold, ColorYellow, col2Width, "Modified", ColorReset,
		ColorGray, ColorReset,
		ColorBold, ColorYellow, col3Width, "Size", ColorReset,
		ColorGray, ColorReset,
		ColorBold, ColorYellow, col4Width, "Comment", ColorReset,
		ColorGray, ColorReset)

	fmt.Printf("%s├%s┼%s┼%s┼%s┤%s\n",
		ColorGray,
		strings.Repeat("─", col1Width+2),
		strings.Repeat("─", col2Width+2),
		strings.Repeat("─", col3Width+2),
		strings.Repeat("─", col4Width+2),
		ColorReset)

	for i, backup := range backups {
		name := backup.Name
		numWidth := len(fmt.Sprintf("%3d. ", i+1))
		maxNameLen := col1Width - numWidth
		if len(name) > maxNameLen {
			name = name[:maxNameLen-3] + "..."
		}

		modTime := backup.ModTime.Format("2006-01-02 15:04:05")
		sizeStr := formatSize(backup.Size)
		
		comment := backup.Comment
		if comment == "" {
			comment = "-"
		} else {
			if len(comment) > col4Width {
				comment = comment[:col4Width-3] + "..."
			}
		}

		fmt.Printf("%s│%s %3d. %-*s %s│%s %-*s %s│%s %*s %s│%s %-*s %s│%s\n",
			ColorGray, ColorReset,
			i+1, maxNameLen, name,
			ColorGray, ColorReset,
			col2Width, modTime,
			ColorGray, ColorReset,
			col3Width, sizeStr,
			ColorGray, ColorReset,
			col4Width, comment,
			ColorGray, ColorReset)
	}

	fmt.Printf("%s└%s┴%s┴%s┴%s┘%s\n\n",
		ColorGray,
		strings.Repeat("─", col1Width+2),
		strings.Repeat("─", col2Width+2),
		strings.Repeat("─", col3Width+2),
		strings.Repeat("─", col4Width+2),
		ColorReset)
}

// Add the missing comment parameter
func restoreBackup(backupPath, originalPath, comment string) error {
	if err := validatePath(originalPath); err != nil {
		return err
	}

	info, err := os.Stat(backupPath)
	if err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}

	if info.Size() > int64(appConfig.MaxClipboardSize) {
		return fmt.Errorf("backup file too large to restore (max %dMB)", appConfig.MaxClipboardSize/(1024*1024))
	}

	content, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	if _, err := os.Stat(originalPath); err == nil {
		if comment == "" {
			comment = "Backup before restore"
		}
		_, err = autoRenameIfExists(originalPath, comment)
		if err != nil {
			return fmt.Errorf("failed to backup current file: %w", err)
		}
	}

	err = os.WriteFile(originalPath, content, 0644)
	if err != nil {
		return fmt.Errorf("failed to restore file: %w", err)
	}

	logger.Printf("Restored: %s from %s", originalPath, backupPath)
	fmt.Printf("✅ Successfully restored: %s\n", originalPath)
	fmt.Printf("📦 From backup: %s\n", filepath.Base(backupPath))
	fmt.Printf("📄 Content size: %d characters\n", len(content))
	
	if comment != "" {
		fmt.Printf("💬 Restore comment: \"%s\"\n", comment)
	}

	return nil
}

func parseWriteArgs(args []string) (filename string, comment string, checkMode bool, err error) {
	if len(args) == 0 {
		return "", "", false, fmt.Errorf("filename required")
	}
	
	filename = args[0]
	comment = ""
	checkMode = false
	
	i := 1
	for i < len(args) {
		switch args[i] {
		case "-m", "--message":
			if i+1 >= len(args) {
				return "", "", false, fmt.Errorf("-m/--message requires a value")
			}
			i++
			comment = args[i]
		case "-c", "--check":
			checkMode = true
		default:
			return "", "", false, fmt.Errorf("unknown flag: %s", args[i])
		}
		i++
	}
	
	return filename, comment, checkMode, nil
}

func readUserChoice(max int) (int, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Enter backup number to restore (1-%d) or 0 to cancel: ", max)

	input, err := reader.ReadString('\n')
	if err != nil {
		return 0, fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)

	choice, err := strconv.Atoi(input)
	if err != nil {
		return 0, fmt.Errorf("invalid input: please enter a number")
	}

	if choice < 0 || choice > max {
		return 0, fmt.Errorf("invalid selection: must be between 0 and %d", max)
	}

	return choice, nil
}

func printHelp() {
	fmt.Printf("\n%s╔══════════════════════════════════════════════════════════╗%s\n", ColorCyan, ColorReset)
	fmt.Printf("%s║%s          %sPT - Clipboard to File Tool v%s%s             %s║%s\n", 
		ColorCyan, ColorReset, ColorBold, Version, ColorReset, ColorCyan, ColorReset)
	fmt.Printf("%s║                                                          ║%s\n", ColorCyan, ColorReset)
    fmt.Printf("%s║                     by cumulus13                         ║%s\n", ColorCyan, ColorReset)
	fmt.Printf("%s╚══════════════════════════════════════════════════════════╝%s\n\n", ColorCyan, ColorReset)
	
	fmt.Printf("%s📝 BASIC OPERATIONS:%s\n", ColorBold+ColorYellow, ColorReset)
	fmt.Printf("  %spt <filename>%s               Write clipboard to file\n", ColorGreen, ColorReset)
	fmt.Printf("  %spt <filename> -c%s            Write only if content differs\n", ColorGreen, ColorReset)
	fmt.Printf("  %spt <filename> -m \"msg\"%s      Write with comment\n", ColorGreen, ColorReset)
	fmt.Printf("  %spt + <filename>%s             Append clipboard to file\n", ColorGreen, ColorReset)
	
	fmt.Printf("\n%s🎯 GIT-LIKE WORKFLOW (NEW!):%s\n", ColorBold+ColorYellow, ColorReset)
	fmt.Printf("  %spt check%s                    Show status of all files (like git status)\n", ColorGreen, ColorReset)
	fmt.Printf("  %spt check <filename>%s         Check single file status\n", ColorGreen, ColorReset)
	fmt.Printf("  %spt commit -m \"message\"%s      Backup all changed files (like git commit)\n", ColorGreen, ColorReset)
	
	fmt.Printf("\n%s📦 BACKUP OPERATIONS:%s\n", ColorBold+ColorYellow, ColorReset)
	fmt.Printf("  %spt -l <filename>%s            List all backups (with comments)\n", ColorGreen, ColorReset)
	fmt.Printf("  %spt -r <filename>%s            Restore backup (interactive)\n", ColorGreen, ColorReset)
	fmt.Printf("  %spt -r <filename> --last%s     Restore most recent backup\n", ColorGreen, ColorReset)
	
	fmt.Printf("\n%s📊 DIFF OPERATIONS:%s\n", ColorBold+ColorYellow, ColorReset)
	fmt.Printf("  %spt -d <filename>%s            Compare with backup (interactive)\n", ColorGreen, ColorReset)
	fmt.Printf("  %spt -d <filename> --last%s     Compare with most recent backup\n", ColorGreen, ColorReset)
	
	fmt.Printf("\n%s🌳 TREE & UTILITIES:%s\n", ColorBold+ColorYellow, ColorReset)
	fmt.Printf("  %spt -t [path]%s                Show directory tree\n", ColorGreen, ColorReset)
	fmt.Printf("  %spt -t [path] -e items,items%s       Tree with exceptions\n", ColorGreen, ColorReset)
	fmt.Printf("  %spt -rm <filename>%s           Safe delete (backup first)\n", ColorGreen, ColorReset)
	
	fmt.Printf("\n%s⚙️ CONFIGURATION:%s\n", ColorBold+ColorYellow, ColorReset)
	fmt.Printf("  %spt config init%s              Create sample config file\n", ColorGreen, ColorReset)
	fmt.Printf("  %spt config show%s              Show current configuration\n", ColorGreen, ColorReset)
	fmt.Printf("  %spt config path%s              Show config file location\n", ColorGreen, ColorReset)
	
	fmt.Printf("\n%sℹ️ INFORMATION:%s\n", ColorBold+ColorYellow, ColorReset)
	fmt.Printf("  %spt -h, --help%s               Show this help message\n", ColorGreen, ColorReset)
	fmt.Printf("  %spt -v, --version%s            Show version information\n", ColorGreen, ColorReset)

	fmt.Printf("\n%s🪲 DEBUGGING:%s\n", ColorBold+ColorYellow, ColorReset)
	fmt.Printf("  %spt --debug%s                  Show debug/logging\n", ColorGreen, ColorReset)
	
	fmt.Printf("\n%s💡 EXAMPLES:%s\n", ColorBold+ColorCyan, ColorReset)
	fmt.Printf("  %s$%s pt notes.txt                %s# Save clipboard%s\n", ColorGray, ColorReset, ColorGray, ColorReset)
	fmt.Printf("  %s$%s pt check                    %s# Show all file statuses%s\n", ColorGray, ColorReset, ColorGray, ColorReset)
	fmt.Printf("  %s$%s pt commit -m \"fix bugs\"     %s# Backup all changes%s\n", ColorGray, ColorReset, ColorGray, ColorReset)
	fmt.Printf("  %s$%s pt -l notes.txt             %s# List backups%s\n", ColorGray, ColorReset, ColorGray, ColorReset)
	fmt.Printf("  %s$%s pt -d notes.txt --last      %s# Diff with last backup%s\n", ColorGray, ColorReset, ColorGray, ColorReset)
	
	fmt.Printf("\n%s🎯 GIT-LIKE WORKFLOW:%s\n", ColorBold+ColorCyan, ColorReset)
	fmt.Printf("  1. %spt check%s                  - See what files changed (like git status)\n", ColorYellow, ColorReset)
	fmt.Printf("  2. %spt commit -m \"msg\"%s        - Backup all changes (like git commit)\n", ColorYellow, ColorReset)
	fmt.Printf("  3. %spt -l <file>%s              - View commit history\n", ColorYellow, ColorReset)
	fmt.Printf("  4. %spt -d <file> --last%s       - See what changed\n", ColorYellow, ColorReset)
	fmt.Printf("  5. %spt -r <file> --last%s       - Rollback if needed\n", ColorYellow, ColorReset)
	
	fmt.Printf("\n%s📊 CHECK/STATUS OUTPUT:%s\n", ColorBold+ColorCyan, ColorReset)
	fmt.Printf("  • %sGreen%s   = Unchanged (matches last backup)\n", ColorGreen, ColorReset)
	fmt.Printf("  • %sYellow%s  = Modified (content changed)\n", ColorYellow, ColorReset)
	fmt.Printf("  • %sCyan%s    = New (no backup exists yet)\n", ColorCyan, ColorReset)
	fmt.Printf("  • %sRed%s     = Deleted (backup exists but file gone)\n", ColorRed, ColorReset)
	
	fmt.Printf("\n%s📦 COMMIT BEHAVIOR:%s\n", ColorBold+ColorCyan, ColorReset)
	fmt.Printf("  • Only backs up %smodified%s and %snew%s files\n", ColorYellow, ColorReset, ColorCyan, ColorReset)
	fmt.Printf("  • Skips %sunchanged%s files (no backup needed)\n", ColorGreen, ColorReset)
	fmt.Printf("  • All backups tagged with \"commit: message\"\n")
	fmt.Printf("  • Confirmation prompt before backing up\n")
	
	fmt.Printf("\n%s🔍 RECURSIVE SEARCH:%s\n", ColorBold+ColorCyan, ColorReset)
	fmt.Printf("  • If file not in current directory, searches recursively\n")
	fmt.Printf("  • Maximum search depth: %d levels\n", appConfig.MaxSearchDepth)
	fmt.Printf("  • If multiple files found, prompts for selection\n")
	fmt.Printf("  • Respects %s.ptignore%s and %s.gitignore%s patterns\n", ColorYellow, ColorReset, ColorYellow, ColorReset)
	
	fmt.Printf("\n%s📂 %s DIRECTORY (Git-like structure):%s\n", ColorBold+ColorCyan, appConfig.BackupDirName, ColorReset)
	fmt.Printf("  • Location: %s%s/%s directory (like .git)\n", ColorYellow, appConfig.BackupDirName, ColorReset)
	fmt.Printf("  • Searches parent directories for existing %s%s/%s\n", ColorYellow, appConfig.BackupDirName, ColorReset)
	fmt.Printf("  • If found in parent, uses that (like git)\n")
	fmt.Printf("  • If not found, creates %s%s/%s in current directory\n", ColorYellow, appConfig.BackupDirName, ColorReset)
	fmt.Printf("  • Automatically added to %s.gitignore%s\n", ColorYellow, ColorReset)
	fmt.Printf("  • Backups organized by file path inside %s%s/%s\n", ColorYellow, appConfig.BackupDirName, ColorReset)
	
	fmt.Printf("\n%s📄 IGNORE FILES:%s\n", ColorBold+ColorCyan, ColorReset)
	fmt.Printf("  • %s.ptignore%s: PT-specific ignore patterns (higher priority)\n", ColorYellow, ColorReset)
	fmt.Printf("  • %s.gitignore%s: Also respected for recursive search\n", ColorYellow, ColorReset)
	fmt.Printf("  • Format: One pattern per line, # for comments\n")
	fmt.Printf("  • %s%s/%s directory always excluded from search\n", ColorYellow, appConfig.BackupDirName, ColorReset)
	
	fmt.Printf("\n%s⚙️  SYSTEM LIMITS:%s\n", ColorBold+ColorCyan, ColorReset)
	fmt.Printf("  • Max file size: %s%dMB%s\n", ColorYellow, appConfig.MaxClipboardSize/(1024*1024), ColorReset)
	fmt.Printf("  • Max filename: %s%d characters%s\n", ColorYellow, appConfig.MaxFilenameLen, ColorReset)
	fmt.Printf("  • Max backups: %s%d per file%s\n", ColorYellow, appConfig.MaxBackupCount, ColorReset)
	fmt.Printf("  • Search depth: %s%d levels%s\n", ColorYellow, appConfig.MaxSearchDepth, ColorReset)
	
	fmt.Printf("\n%s🔧 REQUIREMENTS:%s\n", ColorBold+ColorCyan, ColorReset)
	fmt.Printf("  • %sdelta%s: Required for diff operations\n", ColorYellow, ColorReset)
	fmt.Printf("    Install: %shttps://github.com/dandavison/delta%s\n", ColorGray, ColorReset)
	fmt.Printf("    %s- macOS:%s     brew install git-delta\n", ColorGray, ColorReset)
	fmt.Printf("    %s- Linux:%s     cargo install git-delta\n", ColorGray, ColorReset)
	fmt.Printf("    %s- Windows:%s   scoop install delta\n", ColorGray, ColorReset)
	
	fmt.Printf("\n%s🛡️  SECURITY FEATURES:%s\n", ColorBold+ColorCyan, ColorReset)
	fmt.Printf("  • Path traversal protection (blocks '..' in paths)\n")
	fmt.Printf("  • System directory protection (blocks /etc, /sys, etc.)\n")
	fmt.Printf("  • Write permission validation\n")
	fmt.Printf("  • File size validation\n")
	fmt.Printf("  • Atomic-like backup operations\n")
	
	fmt.Printf("\n%s📋 NOTES:%s\n", ColorBold+ColorCyan, ColorReset)
	fmt.Printf("  • All operations are logged to stderr for audit trail\n")
	fmt.Printf("  • Backup timestamps use microsecond precision\n")
	fmt.Printf("  • Files are synced to disk after writing\n")
	fmt.Printf("  • Supports cross-platform operation (Linux, macOS, Windows)\n")
	fmt.Printf("  • %s%s/%s directory works like %s.git/%s - searches upward\n", 
		ColorYellow, appConfig.BackupDirName, ColorReset, ColorYellow, ColorReset)
	
	fmt.Printf("\n%s📄 LICENSE:%s MIT | %sAUTHOR:%s Hadi Cahyadi <cumulus13@gmail.com>\n", 
		ColorBold, ColorReset, ColorBold, ColorReset)
	fmt.Println()
	
	fmt.Printf("\n%s🔍 RECURSIVE SEARCH:%s\n", ColorBold+ColorCyan, ColorReset)
	fmt.Printf("  • If file not in current directory, searches recursively\n")
	fmt.Printf("  • Maximum search depth: %d levels\n", appConfig.MaxSearchDepth)
	fmt.Printf("  • If multiple files found, prompts for selection\n")
	fmt.Printf("  • Respects %s.ptignore%s and %s.gitignore%s patterns\n", ColorYellow, ColorReset, ColorYellow, ColorReset)
	
	fmt.Printf("\n%s📂 %s DIRECTORY (Git-like structure):%s\n", ColorBold+ColorCyan, appConfig.BackupDirName, ColorReset)
	fmt.Printf("  • Location: %s%s/%s directory (like .git)\n", ColorYellow, appConfig.BackupDirName, ColorReset)
	fmt.Printf("  • Searches parent directories for existing %s%s/%s\n", ColorYellow, appConfig.BackupDirName, ColorReset)
	fmt.Printf("  • If found in parent, uses that (like git)\n")
	fmt.Printf("  • If not found, creates %s%s/%s in current directory\n", ColorYellow, appConfig.BackupDirName, ColorReset)
	fmt.Printf("  • Automatically added to %s.gitignore%s\n", ColorYellow, ColorReset)
	fmt.Printf("  • Backups organized by file path inside %s%s/%s\n", ColorYellow, appConfig.BackupDirName, ColorReset)
	
	fmt.Printf("\n%s📄 IGNORE FILES:%s\n", ColorBold+ColorCyan, ColorReset)
	fmt.Printf("  • %s.ptignore%s: PT-specific ignore patterns (higher priority)\n", ColorYellow, ColorReset)
	fmt.Printf("  • %s.gitignore%s: Also respected for recursive search\n", ColorYellow, ColorReset)
	fmt.Printf("  • Format: One pattern per line, # for comments\n")
	fmt.Printf("  • %s%s/%s directory always excluded from search\n", ColorYellow, appConfig.BackupDirName, ColorReset)
	
	fmt.Printf("\n%s⚙️  SYSTEM LIMITS:%s\n", ColorBold+ColorCyan, ColorReset)
	fmt.Printf("  • Max file size: %s%dMB%s\n", ColorYellow, appConfig.MaxClipboardSize/(1024*1024), ColorReset)
	fmt.Printf("  • Max filename: %s%d characters%s\n", ColorYellow, appConfig.MaxFilenameLen, ColorReset)
	fmt.Printf("  • Max backups: %s%d per file%s\n", ColorYellow, appConfig.MaxBackupCount, ColorReset)
	fmt.Printf("  • Search depth: %s%d levels%s\n", ColorYellow, appConfig.MaxSearchDepth, ColorReset)
	
	fmt.Printf("\n%s🔧 REQUIREMENTS:%s\n", ColorBold+ColorCyan, ColorReset)
	fmt.Printf("  • %sdelta%s: Required for diff operations\n", ColorYellow, ColorReset)
	fmt.Printf("    Install: %shttps://github.com/dandavison/delta%s\n", ColorGray, ColorReset)
	fmt.Printf("    %s- macOS:%s     brew install git-delta\n", ColorGray, ColorReset)
	fmt.Printf("    %s- Linux:%s     cargo install git-delta\n", ColorGray, ColorReset)
	fmt.Printf("    %s- Windows:%s   scoop install delta\n", ColorGray, ColorReset)
	
	fmt.Printf("\n%s🛡️  SECURITY FEATURES:%s\n", ColorBold+ColorCyan, ColorReset)
	fmt.Printf("  • Path traversal protection (blocks '..' in paths)\n")
	fmt.Printf("  • System directory protection (blocks /etc, /sys, etc.)\n")
	fmt.Printf("  • Write permission validation\n")
	fmt.Printf("  • File size validation\n")
	fmt.Printf("  • Atomic-like backup operations\n")
	
	fmt.Printf("\n%s📋 NOTES:%s\n", ColorBold+ColorCyan, ColorReset)
	fmt.Printf("  • All operations are logged to stderr for audit trail\n")
	fmt.Printf("  • Backup timestamps use microsecond precision\n")
	fmt.Printf("  • Files are synced to disk after writing\n")
	fmt.Printf("  • Supports cross-platform operation (Linux, macOS, Windows)\n")
	fmt.Printf("  • %s%s/%s directory works like %s.git/%s - searches upward\n", 
		ColorYellow, appConfig.BackupDirName, ColorReset, ColorYellow, ColorReset)
	
	fmt.Printf("\n%s📄 LICENSE:%s MIT | %sAUTHOR:%s %s%sHadi Cahyadi%s %s%s<cumulus13@gmail.com>%s\n", 
		ColorBrightGreen, ColorReset, ColorBrightBlue, ColorReset, ColorWhite, BgBlue, ColorReset, ColorWhite, ColorMagenta, ColorReset)
	fmt.Println()
}

func printVersion() {
	fmt.Printf("PT version %s\n", Version)
	fmt.Printf("Production-hardened clipboard to file tool\n")
	fmt.Printf("Features: Git-like %s structure, recursive search, backup management, delta diff\n", appConfig.BackupDirName)
	fmt.Println()
	
	versionPaths := []string{
		"VERSION",
		filepath.Join(filepath.Dir(os.Args[0]), "VERSION"),
	}
	
	for _, versionPath := range versionPaths {
		if _, err := os.Stat(versionPath); err == nil {
			absPath, _ := filepath.Abs(versionPath)
			fmt.Printf("Version file: %s\n", absPath)
			break
		}
	}
	
	configPath := findConfigFile()
	if configPath != "" {
		fmt.Printf("Config file: %s\n", configPath)
	} else {
		fmt.Println("Config: Using defaults (no config file)")
	}
}

func main() {
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

	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	// Parse global flags first
    for _, arg := range os.Args[1:] {
        if arg == "--debug" {
            debugMode = true
            break
        }
    }
    // Setup logger based on the parsed debug flag
    setupLogger()

	switch os.Args[1] {
		case "check", "-c", "--check":
			// Handle both single file check and full status
			err := handleCheckCommand(os.Args[2:])
			if err != nil {
				fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
				os.Exit(1)
			}

		case "commit":
			err := handleCommitCommand(os.Args[2:])
			if err != nil {
				fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
				os.Exit(1)
			}

		case "config":
			if len(os.Args) < 3 {
				fmt.Printf("%s❌ Error: Config subcommand required%s\n", ColorRed, ColorReset)
				fmt.Println("\nAvailable subcommands:")
				fmt.Println("  pt config init [path]  - Create sample config file")
				fmt.Println("  pt config show         - Show current configuration")
				fmt.Println("  pt config path         - Show config file location")
				os.Exit(1)
			}
			
			err := handleConfigCommand(os.Args[2:])
			if err != nil {
				fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
				os.Exit(1)
			}

		
		default:
			// Use parseWriteArgs for the default write mode
			text, err := getClipboardText()
			if err != nil {
				fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
				os.Exit(1)
			}

			if text == "" {
				fmt.Printf("%s⚠️  Warning: Clipboard is empty%s\n", ColorYellow, ColorReset)
				os.Exit(1)
			}

			// Parse arguments using parseWriteArgs
			filename, comment, checkMode, err := parseWriteArgs(os.Args[1:])
			if err != nil {
				fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
				os.Exit(1)
			}

			filePath, err := resolveFilePath(filename)
			if err != nil {
				filePath = filename
			}

			if checkMode {
				fmt.Printf("🔍 Check mode enabled - will skip if content identical\n")
			}

			err = writeFile(filePath, text, false, checkMode, comment)
			if err != nil {
				fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
				os.Exit(1)
			}

		case "-t", "--tree":
			err := handleTreeCommand(os.Args[2:])
			if err != nil {
				fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
				os.Exit(1)
			}

		case "-rm", "--remove":
			if len(os.Args) < 3 {
				fmt.Printf("%s❌ Error: Filename required%s\n", ColorRed, ColorReset)
				os.Exit(1)
			}

			err := handleRemoveCommand(os.Args[2:])
			if err != nil {
				fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
				os.Exit(1)
			}

		case "-l", "--list":
			if len(os.Args) < 3 {
				fmt.Printf("%s❌ Error: Filename required%s\n", ColorRed, ColorReset)
				os.Exit(1)
			}

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
				fmt.Printf("ℹ️  No backups found for: %s (check %s/ directory)\n", filePath, appConfig.BackupDirName)
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
			comment := ""
			useLast := false

			for i := 3; i < len(os.Args); i++ {
				if os.Args[i] == "--last" {
					useLast = true
				} else if os.Args[i] == "-m" || os.Args[i] == "--message" {
					if i+1 < len(os.Args) {
						i++
						comment = os.Args[i]
					}
				}
			}

			filePath, err := resolveFilePath(filename)
			if err != nil {
				filePath = filename
				absPath, err := filepath.Abs(filePath)
				if err == nil {
					filePath = absPath
				}
			}

			backups, err := listBackups(filePath)
			if err != nil {
				fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
				os.Exit(1)
			}

			if len(backups) == 0 {
				fmt.Printf("%s❌ Error: No backups found for: %s (check %s/ directory)%s\n", 
					ColorRed, filePath, appConfig.BackupDirName, ColorReset)
				os.Exit(1)
			}

			if useLast {
				if comment == "" {
					comment = "Restored from last backup"
				}
				err = restoreBackup(backups[0].Path, filePath, comment)
				if err != nil {
					fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
					os.Exit(1)
				}
			} else {
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

				selectedBackup := backups[choice-1]
				if comment == "" {
					comment = "Restored from backup"
				}
				err = restoreBackup(selectedBackup.Path, filePath, comment)
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

			// Parse the arguments for append correctly
			filename := os.Args[2]
			comment := ""
			
			for i := 3; i < len(os.Args); i++ {
				if os.Args[i] == "-m" || os.Args[i] == "--message" {
					if i+1 < len(os.Args) {
						i++
						comment = os.Args[i]
					}
				}
			}

			filePath, err := resolveFilePath(filename)
			if err != nil {
				filePath = filename
			}

			err = writeFile(filePath, text, true, false, comment)
			if err != nil {
				fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
				os.Exit(1)
			}

	}
}	
