// Daftar Perbaikan yang Dilakukan pada PT Tool:

// 1. BUG FIX: Fungsi parseWriteArgs tidak pernah dipanggil
//    - Fungsi ini didefinisikan tapi tidak digunakan di main()
//    - Seharusnya digunakan untuk parsing argumen -m dan -c

// 2. BUG FIX: Handling argumen -m (message/comment) tidak konsisten
//    - Di beberapa command sudah ada, di command lain belum
//    - Perlu standardisasi parsing argumen

// 3. BUG FIX: restoreBackup() dipanggil tanpa parameter comment
//    - Fungsi signature: restoreBackup(backupPath, originalPath, comment string)
//    - Dipanggil dengan: restoreBackup(backups[0].Path, filePath)
//    - Missing parameter comment

// 4. IMPROVEMENT: Error handling kurang informatif
//    - Beberapa error tidak memberikan context yang cukup

// 5. IMPROVEMENT: Validasi input user kurang ketat
//    - Perlu validasi lebih baik untuk user input

// 6. CODE SMELL: Duplicate code dalam parsing argumen
//    - Banyak duplikasi logic parsing -m, -c, dll

// KODE YANG DIPERBAIKI:

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
	DefaultBackupDirName    = "backup"           // Backup directory name
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
	Comment string
}

// BackupMetadata stores metadata for backup files
type BackupMetadata struct {
	Comment   string    `json:"comment"`
	Timestamp time.Time `json:"timestamp"`
	Size      int64     `json:"size"`
	Original  string    `json:"original_file"`
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
	logger = log.New(os.Stderr, "", log.LstdFlags)
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
		fmt.Printf("%sBackup Directory:%s %s/\n", ColorCyan, ColorReset, appConfig.BackupDirName)
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

func loadGitIgnore(rootPath string) (*GitIgnore, error) {
	gitignorePath := filepath.Join(rootPath, ".gitignore")
	gi := &GitIgnore{patterns: make([]string, 0)}
	
	file, err := os.Open(gitignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return gi, nil
		}
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		gi.patterns = append(gi.patterns, line)
	}

	return gi, scanner.Err()
}

func (gi *GitIgnore) shouldIgnore(path string, isDir bool) bool {
	baseName := filepath.Base(path)
	
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
		gitignore, err = loadGitIgnore(absPath)
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
		fmt.Printf("%sUsing .gitignore (%d patterns)%s\n", ColorGray, len(gitignore.patterns), ColorReset)
	}

	return nil
}

// FIX: Tambahkan parsing comment untuk handleRemoveCommand
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
	fmt.Printf("ℹ️  Original content (%d bytes) backed up to ./%s/\n", len(content), appConfig.BackupDirName)

	return nil
}

func searchFileRecursive(filename string, maxDepth int) ([]FileSearchResult, error) {
	results := make([]FileSearchResult, 0)
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

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

		if info.IsDir() && info.Name() == appConfig.BackupDirName {
			return filepath.SkipDir
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
	
	// FIX: Delta exit code 1 adalah NORMAL saat file berbeda
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
		return fmt.Errorf("no backups found for: %s (check ./%s/ directory)", filePath, appConfig.BackupDirName)
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

func ensureBackupDir(filePath string) (string, error) {
	dir := filepath.Dir(filePath)
	if dir == "." {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	backupDir := filepath.Join(dir, appConfig.BackupDirName)

	info, err := os.Stat(backupDir)
	if os.IsNotExist(err) {
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

	backupDir, err := ensureBackupDir(filePath)
	if err != nil {
		return filePath, err
	}

	backupFileName := generateUniqueBackupName(filePath)
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

	dir := filepath.Dir(filePath)
	if dir == "." {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}
	
	backupDir := filepath.Join(dir, appConfig.BackupDirName)

	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return []BackupInfo{}, nil
	}

	baseName := filepath.Base(filePath)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)
	extWithoutDot := strings.TrimPrefix(ext, ".")
	
	pattern := fmt.Sprintf("%s_%s.", nameWithoutExt, extWithoutDot)
	
	logger.Printf("Looking for backups with pattern: %s in directory: %s", pattern, backupDir)

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
		
		if strings.HasSuffix(name, ".meta.json") {
			continue
		}
		
		logger.Printf("Checking file: %s against pattern: %s", name, pattern)
		
		if !strings.HasPrefix(name, pattern) {
			continue
		}

		timestamp := strings.TrimPrefix(name, pattern)
		
		logger.Printf("Extracted timestamp: %s (length: %d)", timestamp, len(timestamp))
		
		if len(timestamp) < 20 {
			logger.Printf("Skipping %s: timestamp too short", name)
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
		if err != nil {
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
		return backups, nil
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].ModTime.After(backups[j].ModTime)
	})

	if len(backups) > appConfig.MaxBackupCount {
		backups = backups[:appConfig.MaxBackupCount]
	}

	return backups, nil
}

func printBackupTable(filePath string, backups []BackupInfo) {
	const (
		col1Width = 40  // Lebih lebar untuk filename
		col2Width = 19
		col3Width = 12
		col4Width = 30  // Lebih kecil untuk comment
	)

	fmt.Printf("\n%s📂 Backup files for '%s%s%s%s'%s\n",
		ColorCyan, ColorBold, filePath, ColorReset, ColorCyan, ColorReset)
	fmt.Printf("%sTotal: %d backup(s) (stored in ./%s/)%s\n\n", 
		ColorGray, len(backups), appConfig.BackupDirName, ColorReset)

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
		// Hitung lebar untuk nomor (misal "  10. " = 6 karakter)
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
			// Hitung lebar yang tersedia untuk comment (tanpa warna)
			if len(comment) > col4Width {
				comment = comment[:col4Width-3] + "..."
			}
		}

		// Format row dengan padding yang konsisten
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

// FIX: Tambahkan parameter comment yang hilang
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

// FIX: Fungsi parseWriteArgs sekarang benar-benar digunakan
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
	fmt.Printf("%sPT - Clipboard to File Tool with Smart Version Management v%s%s\n\n", ColorBold, Version, ColorReset)
	fmt.Println("Usage:")
	fmt.Println("  pt <filename>                    Write clipboard to file")
	fmt.Println("  pt <filename> -c                 Write only if content differs (check mode)")
	fmt.Println("  pt <filename> -m \"comment\"       Write with comment")
	fmt.Println("  pt <filename> -c -m \"comment\"   Check mode with comment")
	fmt.Println("  pt + <filename>                  Append clipboard to file")
	fmt.Println("  pt + <filename> -m \"comment\"    Append with comment")
	fmt.Println("  pt -l <filename>                 List backups (with comments)")
	fmt.Println("  pt -r <filename>                 Restore backup (interactive)")
	fmt.Println("  pt -r <filename> --last          Restore last backup")
	fmt.Println("  pt -r <filename> -m \"comment\"   Restore with comment")
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

func printVersion() {
	fmt.Printf("PT version %s\n", Version)
	fmt.Println("Production-hardened clipboard to file tool")
	fmt.Println("Features: Recursive search, backup management, delta diff, tree view, safe delete, configurable")
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
		comment := ""
		useLast := false

		// FIX: Parse argumen untuk restore dengan benar
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
			// Untuk restore, file mungkin belum ada
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
			fmt.Printf("%s❌ Error: No backups found for: %s (check ./%s/ directory)%s\n", 
				ColorRed, filePath, appConfig.BackupDirName, ColorReset)
			os.Exit(1)
		}

		// FIX: Gunakan parameter comment saat memanggil restoreBackup
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

		// FIX: Parse argumen untuk append dengan benar
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

	default:
		// FIX: Gunakan parseWriteArgs untuk write mode default
		text, err := getClipboardText()
		if err != nil {
			fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
			os.Exit(1)
		}

		if text == "" {
			fmt.Printf("%s⚠️  Warning: Clipboard is empty%s\n", ColorYellow, ColorReset)
			os.Exit(1)
		}

		// Parse argumen menggunakan parseWriteArgs
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
	}
}