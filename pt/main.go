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
	"gopkg.in/yaml.v3"
)

// Configuration constants (defaults)
const (
	DefaultMaxClipboardSize = 100 * 1024 * 1024 // 100MB max
	DefaultMaxBackupCount   = 100                // Keep max 100 backups
	DefaultMaxFilenameLen   = 200                // Max filename length
	DefaultBackupDirName    = "backup"           // Backup directory name
	DefaultMaxSearchDepth   = 10                 // Max directory depth for recursive search
)

// Version will be loaded from VERSION file
var Version string = "dev"

// Config holds the application configuration
type Config struct {
	MaxClipboardSize int    `yaml:"max_clipboard_size"` // in bytes
	MaxBackupCount   int    `yaml:"max_backup_count"`
	MaxFilenameLen   int    `yaml:"max_filename_length"`
	BackupDirName    string `yaml:"backup_dir_name"`
	MaxSearchDepth   int    `yaml:"max_search_depth"`
}

// Global config instance
var appConfig *Config

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

func init() {
	// Initialize logger (write to stderr to not interfere with stdout)
	logger = log.New(os.Stderr, "", log.LstdFlags)
	
	// Load version from VERSION file
	Version = loadVersion()
	
	// Load configuration
	appConfig = loadConfig()
}

// loadVersion loads version from VERSION file
func loadVersion() string {
	// Try to find VERSION file in multiple locations
	versionPaths := []string{
		"VERSION",                                    // Current directory
		filepath.Join(filepath.Dir(os.Args[0]), "VERSION"), // Same directory as executable
		"/usr/local/share/pt/VERSION",               // Linux system location
		filepath.Join(os.Getenv("HOME"), ".local", "share", "pt", "VERSION"), // User location
	}
	
	// Windows locations
	if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
		versionPaths = append(versionPaths, 
			filepath.Join(userProfile, ".pt", "VERSION"),
			filepath.Join(filepath.Dir(os.Args[0]), "VERSION"),
		)
	}
	
	for _, versionPath := range versionPaths {
		data, err := os.ReadFile(versionPath)
		if err == nil {
			// Parse version from file content
			content := strings.TrimSpace(string(data))
			
			// Support formats:
			// 1. version = "1.0.12"
			// 2. 1.0.12
			// 3. v1.0.12
			
			// Remove 'version = ' prefix if exists
			if strings.HasPrefix(content, "version") {
				parts := strings.SplitN(content, "=", 2)
				if len(parts) == 2 {
					content = strings.TrimSpace(parts[1])
				}
			}
			
			// Remove quotes
			content = strings.Trim(content, `"'`)
			
			// Remove 'v' prefix if exists
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

// getDefaultConfig returns default configuration
func getDefaultConfig() *Config {
	return &Config{
		MaxClipboardSize: DefaultMaxClipboardSize,
		MaxBackupCount:   DefaultMaxBackupCount,
		MaxFilenameLen:   DefaultMaxFilenameLen,
		BackupDirName:    DefaultBackupDirName,
		MaxSearchDepth:   DefaultMaxSearchDepth,
	}
}

// findConfigFile searches for pt.yml or pt.yaml in multiple locations
func findConfigFile() string {
	// Config file names to search for
	configNames := []string{"pt.yml", "pt.yaml", ".pt.yml", ".pt.yaml"}
	
	// Search locations (in order of priority)
	searchPaths := []string{
		".",                                    // Current directory
		filepath.Join(os.Getenv("HOME"), ".config", "pt"), // ~/.config/pt/
		os.Getenv("HOME"),                      // Home directory
	}
	
	// Windows home directory
	if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
		searchPaths = append(searchPaths, userProfile, filepath.Join(userProfile, ".pt"))
	}
	
	// Search for config file
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

// loadConfig loads configuration from pt.yml/pt.yaml or uses defaults
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
	
	// Validate loaded config and apply bounds
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

// generateSampleConfig creates a sample pt.yml file
func generateSampleConfig(path string) error {
	config := getDefaultConfig()
	
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	// Add comments to the generated file
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

// handleConfigCommand handles config-related commands
func handleConfigCommand(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("config subcommand required: 'init', 'show', or 'path'")
	}
	
	subcommand := args[0]
	
	switch subcommand {
	case "init":
		// Generate sample config file
		var configPath string
		if len(args) > 1 {
			configPath = args[1]
		} else {
			configPath = "pt.yml"
		}
		
		// Check if file already exists
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
		// Show current configuration
		fmt.Printf("\n%sCurrent PT Configuration:%s\n\n", ColorBold, ColorReset)
		fmt.Printf("%sMax Clipboard Size:%s %d bytes (%.1f MB)\n", 
			ColorCyan, ColorReset, appConfig.MaxClipboardSize, float64(appConfig.MaxClipboardSize)/(1024*1024))
		fmt.Printf("%sMax Backup Count:%s %d\n", ColorCyan, ColorReset, appConfig.MaxBackupCount)
		fmt.Printf("%sMax Filename Length:%s %d characters\n", ColorCyan, ColorReset, appConfig.MaxFilenameLen)
		fmt.Printf("%sBackup Directory:%s %s/\n", ColorCyan, ColorReset, appConfig.BackupDirName)
		fmt.Printf("%sMax Search Depth:%s %d levels\n\n", ColorCyan, ColorReset, appConfig.MaxSearchDepth)
		
		configPath := findConfigFile()
		if configPath != "" {
			fmt.Printf("%sConfig loaded from:%s %s\n", ColorGray, ColorReset, configPath)
		} else {
			fmt.Printf("%sUsing default configuration (no config file found)%s\n", ColorGray, ColorReset)
		}
		
	case "path":
		// Show config file path
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

// formatSize formats file size in human-readable format
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

// loadGitIgnore loads .gitignore patterns from a file
func loadGitIgnore(rootPath string) (*GitIgnore, error) {
	gitignorePath := filepath.Join(rootPath, ".gitignore")
	gi := &GitIgnore{patterns: make([]string, 0)}
	
	file, err := os.Open(gitignorePath)
	if err != nil {
		// No .gitignore file is okay
		if os.IsNotExist(err) {
			return gi, nil
		}
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		gi.patterns = append(gi.patterns, line)
	}

	return gi, scanner.Err()
}

// shouldIgnore checks if a path should be ignored based on gitignore patterns
func (gi *GitIgnore) shouldIgnore(path string, isDir bool) bool {
	baseName := filepath.Base(path)
	
	for _, pattern := range gi.patterns {
		// Handle directory patterns (ending with /)
		if strings.HasSuffix(pattern, "/") {
			dirPattern := strings.TrimSuffix(pattern, "/")
			if isDir && (baseName == dirPattern || strings.HasPrefix(baseName, dirPattern)) {
				return true
			}
			continue
		}
		
		// Handle wildcard patterns
		if strings.Contains(pattern, "*") {
			matched, _ := filepath.Match(pattern, baseName)
			if matched {
				return true
			}
			continue
		}
		
		// Exact match
		if baseName == pattern {
			return true
		}
		
		// Check if path contains pattern (for nested paths)
		if strings.Contains(path, "/"+pattern+"/") || strings.Contains(path, "\\"+pattern+"\\") {
			return true
		}
	}
	
	return false
}

// buildTree recursively builds a directory tree
func buildTree(path string, gitignore *GitIgnore, exceptions map[string]bool, depth int, maxDepth int) (*TreeNode, error) {
	if depth > maxDepth {
		return nil, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	baseName := filepath.Base(path)
	
	// Check exceptions
	if exceptions[baseName] {
		return nil, nil
	}

	// Check gitignore
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
			return node, nil // Return node but skip children if can't read
		}

		for _, entry := range entries {
			childPath := filepath.Join(path, entry.Name())
			childNode, err := buildTree(childPath, gitignore, exceptions, depth+1, maxDepth)
			if err != nil || childNode == nil {
				continue
			}
			node.Children = append(node.Children, childNode)
		}

		// Sort children: directories first, then files, alphabetically
		sort.Slice(node.Children, func(i, j int) bool {
			if node.Children[i].IsDir != node.Children[j].IsDir {
				return node.Children[i].IsDir
			}
			return node.Children[i].Name < node.Children[j].Name
		})
	}

	return node, nil
}

// printTree prints the directory tree
func printTree(node *TreeNode, prefix string, isLast bool, showSize bool) {
	if node == nil {
		return
	}

	// Print current node
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

	// Print children
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

// handleTreeCommand handles the -t/--tree command
func handleTreeCommand(args []string) error {
	// Parse arguments
	exceptions := make(map[string]bool)
	startPath := "."
	
	// Check for -e/--exception flags
	i := 0
	for i < len(args) {
		if args[i] == "-e" || args[i] == "--exception" {
			if i+1 >= len(args) {
				return fmt.Errorf("-e/--exception requires a value")
			}
			i++
			// Split comma-separated exceptions
			for _, exc := range strings.Split(args[i], ",") {
				exceptions[strings.TrimSpace(exc)] = true
			}
			i++
		} else {
			// This should be the path
			startPath = args[i]
			i++
		}
	}

	// Get absolute path
	absPath, err := filepath.Abs(startPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Check if path exists
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("path does not exist: %w", err)
	}

	// Load .gitignore if exists
	var gitignore *GitIgnore
	if info.IsDir() {
		gitignore, err = loadGitIgnore(absPath)
		if err != nil {
			logger.Printf("Warning: failed to load .gitignore: %v", err)
		}
	}

	// Build tree
	tree, err := buildTree(absPath, gitignore, exceptions, 0, appConfig.MaxSearchDepth)
	if err != nil {
		return fmt.Errorf("failed to build tree: %w", err)
	}

	if tree == nil {
		return fmt.Errorf("no files to display")
	}

	// Print tree
	fmt.Printf("\n%s%s%s\n", ColorBold, tree.Name, ColorReset)
	if tree.IsDir && len(tree.Children) > 0 {
		for i, child := range tree.Children {
			printTree(child, "", i == len(tree.Children)-1, true)
		}
	}
	fmt.Println()

	// Print summary
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
		fmt.Printf("%sUsing .gitignore (%d patterns)%s\n", ColorGray, len(gitignore.patterns), ColorReset)
	}

	return nil
}

// handleRemoveCommand handles the -rm/--remove command
func handleRemoveCommand(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("filename required for remove command")
	}

	filename := args[0]

	// Resolve file path with recursive search
	filePath, err := resolveFilePath(filename)
	if err != nil {
		return err
	}

	// Check if file exists
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

	// Create backup before removing
	if info.Size() > 0 {
		_, err = autoRenameIfExists(filePath)
		if err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	// Read file content for logging
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Delete the file
	err = os.Remove(filePath)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	logger.Printf("File deleted: %s (%d bytes)", filePath, len(content))
	fmt.Printf("🗑️  File deleted: %s\n", filePath)

	// Create empty placeholder file with same name
	emptyFile, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create empty placeholder: %w", err)
	}
	emptyFile.Close()

	logger.Printf("Created empty placeholder: %s", filePath)
	fmt.Printf("📄 Created empty placeholder: %s\n", filePath)
	fmt.Printf("ℹ️  Original content (%d bytes) backed up to ./%s/\n", len(content), appConfig.BackupDirName)

	return nil
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
		if info.IsDir() && info.Name() == appConfig.BackupDirName {
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
		return fmt.Errorf("no backups found for: %s (check ./%s/ directory)", filePath, appConfig.BackupDirName)
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
	backupDir := filepath.Join(dir, appConfig.BackupDirName)

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
	if len(filepath.Base(filePath)) > appConfig.MaxFilenameLen {
		return fmt.Errorf("filename too long (max %d characters)", appConfig.MaxFilenameLen)
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
	if len(text) > appConfig.MaxClipboardSize {
		return "", fmt.Errorf("clipboard content too large (max %dMB)", appConfig.MaxClipboardSize/(1024*1024))
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
func writeFile(filePath string, data string, appendMode bool, checkMode bool) error {
	// Validate path
	if err := validatePath(filePath); err != nil {
		return err
	}

	// Check mode: compare with existing file first
	if checkMode && !appendMode {
		if existingData, err := os.ReadFile(filePath); err == nil {
			// File exists, compare content
			if string(existingData) == data {
				logger.Printf("Content identical, skipping write: %s", filePath)
				fmt.Printf("ℹ️  Content identical to current file, no changes needed\n")
				fmt.Printf("📄 File: %s\n", filePath)
				return nil
			}
			fmt.Printf("🔍 Content differs, proceeding with backup and write\n")
		}
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
	
	backupDir := filepath.Join(dir, appConfig.BackupDirName)

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
	if len(backups) > appConfig.MaxBackupCount {
		backups = backups[:appConfig.MaxBackupCount]
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
		ColorGray, len(backups), appConfig.BackupDirName, ColorReset)

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
		sizeStr := formatSize(backup.Size)

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
	if info.Size() > int64(appConfig.MaxClipboardSize) {
		return fmt.Errorf("backup file too large to restore (max %dMB)", appConfig.MaxClipboardSize/(1024*1024))
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
	fmt.Printf("%sPT - Clipboard to File Tool with Smart Version Management v%s%s\n\n", ColorBold, Version, ColorReset)
	fmt.Println("Usage:")
	fmt.Println("  pt <filename>                    Write clipboard to file")
	fmt.Println("  pt <filename> -c                 Write only if content differs (check mode)")
	fmt.Println("  pt + <filename>                  Append clipboard to file")
	fmt.Println("  pt -l <filename>                 List backups")
	fmt.Println("  pt -r <filename>                 Restore backup (interactive)")
	fmt.Println("  pt -r <filename> --last          Restore last backup")
	fmt.Println("  pt -d <filename>                 Compare file with backup (interactive)")
	fmt.Println("  pt -d <filename> --last          Compare file with last backup")
	fmt.Println("  pt -rm <filename>                Delete file (with backup) and create empty placeholder")
	fmt.Println("  pt -t [path]                     Show directory tree with file sizes")
	fmt.Println("  pt -t [path] -e file1,file2      Tree with exceptions (comma-separated)")
	fmt.Println("  pt config <subcommand>           Manage configuration")
	fmt.Println("  pt -h, --help                    Show this help")
	fmt.Println("  pt -v, --version                 Show version")
	fmt.Println("\nConfiguration Commands:")
	fmt.Println("  pt config init [path]            Create sample config file (default: pt.yml)")
	fmt.Println("  pt config show                   Show current configuration")
	fmt.Println("  pt config path                   Show config file location")
	fmt.Println("\nExamples:")
	fmt.Println("  pt notes.txt                     # Save clipboard to notes.txt")
	fmt.Println("  pt notes.txt -c                  # Save only if content changed")
	fmt.Println("  pt notes.txt --check             # Same as above")
	fmt.Println("  pt + log.txt                     # Append clipboard to log.txt")
	fmt.Println("  pt -l notes.txt                  # List all backups")
	fmt.Println("  pt -r notes.txt                  # Interactive restore")
	fmt.Println("  pt -r notes.txt --last           # Restore most recent backup")
	fmt.Println("  pt -d notes.txt                  # Interactive diff with backup")
	fmt.Println("  pt -d notes.txt --last           # Diff with most recent backup")
	fmt.Println("  pt -rm notes.txt                 # Delete notes.txt (backup first)")
	fmt.Println("  pt -t                            # Show tree of current directory")
	fmt.Println("  pt -t /path/to/dir               # Show tree of specific directory")
	fmt.Println("  pt -t -e node_modules,.git       # Tree excluding node_modules and .git")
	fmt.Println("  pt config init                   # Create pt.yml config file")
	fmt.Println("  pt config show                   # View current settings")
	fmt.Printf("\n%sFeatures:%s\n", ColorBold, ColorReset)
	fmt.Printf("  • %sRecursive Search:%s If file not in current dir, searches subdirectories\n", ColorCyan, ColorReset)
	fmt.Printf("  • %sDiff Support:%s Uses 'delta' CLI tool for beautiful diffs\n", ColorCyan, ColorReset)
	fmt.Printf("  • %sTree View:%s Display directory structure with file sizes\n", ColorCyan, ColorReset)
	fmt.Printf("  • %sGitignore Support:%s Respects .gitignore patterns in tree view\n", ColorCyan, ColorReset)
	fmt.Printf("  • %sSafe Delete:%s Backup before delete, create empty placeholder\n", ColorCyan, ColorReset)
	fmt.Printf("  • %sConfigurable:%s Customize via pt.yml config file\n", ColorCyan, ColorReset)
	fmt.Printf("  • %sCheck Mode:%s Skip write if content unchanged (save disk space)\n", ColorCyan, ColorReset)
	fmt.Printf("\n%sBackup Location: All backups stored in ./%s/ directory%s\n", ColorCyan, appConfig.BackupDirName, ColorReset)
	fmt.Printf("%sLimits: Max file size %dMB, Max %d backups kept%s\n",
		ColorGray, appConfig.MaxClipboardSize/(1024*1024), appConfig.MaxBackupCount, ColorReset)
	fmt.Printf("\n%sConfig File Locations (searched in order):%s\n", ColorGray, ColorReset)
	fmt.Println("  • ./pt.yml or ./pt.yaml (current directory)")
	fmt.Println("  • ~/.config/pt/pt.yml or ~/.config/pt/pt.yaml")
	fmt.Println("  • ~/pt.yml or ~/pt.yaml (home directory)")
	fmt.Printf("\n%sNote: Install 'delta' for diff functionality: https://github.com/dandavison/delta%s\n",
		ColorGray, ColorReset)
}

// printVersion displays version information
func printVersion() {
	fmt.Printf("PT version %s\n", Version)
	fmt.Println("Production-hardened clipboard to file tool")
	fmt.Println("Features: Recursive search, backup management, delta diff, tree view, safe delete, configurable")
	fmt.Println()
	
	// Show version file location if found
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
		// fmt.Printf("%s❌ Error: No command specified%s\n\n", ColorRed, ColorReset)
		printHelp()
		os.Exit(1)
	}

	// Handle different commands
	switch os.Args[1] {
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
			fmt.Printf("ℹ️  No backups found for: %s (check ./%s/ directory)\n", filePath, appConfig.BackupDirName)
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
				ColorRed, filePath, appConfig.BackupDirName, ColorReset)
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

		err = writeFile(filePath, text, true, false)
		if err != nil {
			fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
			os.Exit(1)
		}

	default:
		// Write mode (default)
		// Check for -c/--check flag
		checkMode := false
		filename := os.Args[1]
		
		if len(os.Args) > 2 {
			for i := 2; i < len(os.Args); i++ {
				if os.Args[i] == "-c" || os.Args[i] == "--check" {
					checkMode = true
				}
			}
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
		filePath, err := resolveFilePath(filename)
		if err != nil {
			// If file doesn't exist, use the provided path as-is
			filePath = filename
		}

		if checkMode {
			fmt.Printf("🔍 Check mode enabled - will skip if content identical\n")
		}

		err = writeFile(filePath, text, false, checkMode)
		if err != nil {
			fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
			os.Exit(1)
		}
	}
}
