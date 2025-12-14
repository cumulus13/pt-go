// File: pt/main.go
// Author: Hadi Cahyadi <cumulus13@gmail.com>
// Date: 2025-11-18
// Description: Production-hardened clipboard-to-file tool with security, validation, Git-like .pt directory and smart version management
// License: MIT

package main

import (
	"bufio"
	"bytes"
	"runtime"
	// "syscall"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	// "golang.org/x/sys/windows"
	"github.com/atotto/clipboard"
	"gopkg.in/yaml.v3"
	// "github.com/alecthomas/chroma/v2/quick" // Import chroma quick for syntax highlighting
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"golang.org/x/term"

	// "github.com/gdamore/tcell/v2"
	// "github.com/acarl005/stripansi"
	// "github.com/rivo/tview"
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
	DiffTool         string `yaml:"diff_tool"`
}

// Global config instance
var appConfig *Config
var debugMode bool = false
var difftool string = "delta"
var foundZ bool = false

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

// FileSearchResult for recursive file search
type FileSearchResult struct {
	Path    string
	Dir     string
	Size    int64
	ModTime time.Time
	Depth   int
}

// OrphanedBackup represents a backup directory whose original file is missing
type OrphanedBackup struct {
	BackupDir    string
	ExpectedPath string
	ActualFiles  []string
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

// discardWriter implements io.Writer and discards all writes
type discardWriter struct{}

func (d *discardWriter) Write(p []byte) (n int, err error) {
    return len(p), nil // Discard all data
}

func init() {
    // Initialize logger to discard by default in init.
    // It will be set correctly in main() after flag parsing.
    logger = log.New(&discardWriter{}, "", log.LstdFlags)
    Version = loadVersion()
    appConfig = loadConfig()
}

// setupLogger initializes the global logger based on the debugMode flag.
func setupLogger() {
	if debugMode {
		logger = log.New(os.Stderr, "", log.LstdFlags)
	} else {
		logger = log.New(&discardWriter{}, "", log.LstdFlags)
	}
}

func getTerminalWidth() int {
    width, _, err := term.GetSize(int(os.Stdout.Fd()))
    if err != nil {
        return 80 // fallback
    }
    return width
}

// ============================================================================
// SHOW COMMAND - Display file content with syntax highlighting (like bat)
// ============================================================================

func handleShowCommand(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("filename required for show command")
	}

	filename := args[0]
	lexerName := ""
	themeName := "fruity"
	showLineNumbers := true
	showGrid := true
	usePager := true

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--lexer", "-l":
			if i+1 < len(args) {
				lexerName = args[i+1]
				i++
			}
		case "--theme", "-t":
			if i+1 < len(args) {
				themeName = args[i+1]
				i++
			}
		case "--no-line-numbers":
			showLineNumbers = false
		case "--no-grid":
			showGrid = false
		case "--no-pager", "-np":
			usePager = false
		}
	}

	filePath, err := resolveFilePath(filename)
	if err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	if fileInfo.IsDir() {
		return fmt.Errorf("cannot show directory, file required")
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	status, _ := compareFileWithBackup(filePath)

	var output bytes.Buffer

	// Print header
	relPath, _ := filepath.Rel(".", filePath)
	statusColor := status.Color()
	statusSymbol := "●"

	width := getTerminalWidth()

	// if showGrid {
	// 	output.WriteString(fmt.Sprintf("%s───────┬────────────────────────────────────────────────────────────────%s\n", ColorGray, ColorReset))
	// }

	if showGrid {
	    line := "───────┬" + strings.Repeat("─", width-10)
	    output.WriteString(fmt.Sprintf("%s%s%s\n", ColorGray, line, ColorReset))
	}

	output.WriteString(fmt.Sprintf("%s       │%s %sFile:%s %s ", ColorGray, ColorReset, ColorBold, ColorReset, relPath))
	if status != FileStatusUnchanged {
		output.WriteString(fmt.Sprintf("%s%s %s%s", statusColor, statusSymbol, status.String(), ColorReset))
	}
	output.WriteString("\n")

	modTime := fileInfo.ModTime().Format("2006-01-02 15:04:05")
	output.WriteString(fmt.Sprintf("%s       │%s %sSize:%s %s  %sModified:%s %s\n",
		ColorGray, ColorReset,
		ColorCyan, ColorReset, formatSize(fileInfo.Size()),
		ColorCyan, ColorReset, modTime))

	if lexerName != "" {
		output.WriteString(fmt.Sprintf("%s       │%s %sLexer:%s %s  %sTheme:%s %s\n",
			ColorGray, ColorReset,
			ColorCyan, ColorReset, lexerName,
			ColorCyan, ColorReset, themeName))
	}

	// if showGrid {
	// 	output.WriteString(fmt.Sprintf("%s───────┼────────────────────────────────────────────────────────────────%s\n", ColorGray, ColorReset))
	// }

	if showGrid {
	    line := "───────┼" + strings.Repeat("─", width-10)
	    output.WriteString(fmt.Sprintf("%s%s%s\n", ColorGray, line, ColorReset))
	}

	// Apply syntax highlighting
	var lexer chroma.Lexer
	if lexerName != "" {
		lexer = lexers.Get(lexerName)
	} else {
		lexer = lexers.Match(filePath)
	}

	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	style := styles.Get(themeName)
	if style == nil {
		style = styles.Monokai
	}

	formatter := formatters.TTY16m

	iterator, err := lexer.Tokenise(nil, string(content))
	if err != nil {
		return fmt.Errorf("failed to tokenize: %w", err)
	}

	var contentBuf bytes.Buffer
	err = formatter.Format(&contentBuf, style, iterator)
	if err != nil {
		return fmt.Errorf("failed to format: %w", err)
	}

	// Add line numbers
	if showLineNumbers {
		lines := strings.Split(contentBuf.String(), "\n")
		maxLineNum := len(lines)
		lineNumWidth := len(fmt.Sprintf("%d", maxLineNum))

		for i, line := range lines {
			lineNum := i + 1
			if showGrid {
				output.WriteString(fmt.Sprintf("%s%*d │%s %s\n", ColorGray, lineNumWidth, lineNum, ColorReset, line))
			} else {
				output.WriteString(fmt.Sprintf("%s%*d %s %s\n", ColorGray, lineNumWidth, lineNum, ColorReset, line))
			}
		}
	} else {
		output.WriteString(contentBuf.String())
	}

	// Footer
	// if showGrid {
	// 	output.WriteString(fmt.Sprintf("%s───────┴────────────────────────────────────────────────────────────────%s\n", ColorGray, ColorReset))
	// }

	if showGrid {
	    line := strings.Repeat("─", width)
	    output.WriteString(fmt.Sprintf("%s%s%s\n", ColorGray, line, ColorReset))
	}
	output.WriteString("\n")

	if usePager {
		return displayWithPager(output.String())
	} else {
		fmt.Print(output.String())
	}

	return nil
}

// ============================================================================
// TEMP COMMAND (-z) - Display clipboard content with syntax highlighting
// ============================================================================

// handleTempCommand writes clipboard content to a temp file and displays it with less
// func handleTempCommand(args []string) error {
// 	text, err := getClipboardText()
// 	if err != nil {
// 		return fmt.Errorf("failed to read clipboard: %w", err)
// 	}

// 	if text == "" {
// 		return fmt.Errorf("clipboard is empty")
// 	}

// 	// Determine lexer for syntax highlighting
// 	lexerName := ""
// 	for i := 0; i < len(args); i++ {
// 		if args[i] == "--lexer" && i+1 < len(args) {
// 			lexerName = args[i+1]
// 			break
// 		}
// 	}

// 	// Create a temporary file
// 	tmpFile, err := os.CreateTemp("", "pt_temp_*.txt")
// 	if err != nil {
// 		return fmt.Errorf("failed to create temporary file: %w", err)
// 	}
// 	defer os.Remove(tmpFile.Name()) // Clean up the temp file after the function exits
// 	defer tmpFile.Close()

// 	// Write clipboard content to the temp file
// 	_, err = tmpFile.WriteString(text)
// 	if err != nil {
// 		return fmt.Errorf("failed to write to temporary file: %w", err)
// 	}

// 	// Flush the file to ensure content is written
// 	tmpFile.Sync()

// 	// Check if a lexer is specified
// 	if lexerName != "" {
// 		// Use chroma quick.Highlight to format the content and write directly to stdout
// 		// This avoids issues with less stdin
// 		logger.Printf("Highlighting with lexer: '%s'", lexerName)
// 		// Use "terminal16m" or "terminal" style for ANSI colors
// 		err = quick.Highlight(os.Stdout, text, lexerName, "terminal16m", "terminal16m")
// 		if err != nil {
// 			// If highlighting fails, log a warning and proceed with plain output
// 			logger.Printf("Warning: failed to highlight with lexer '%s': %v", lexerName, err)
// 			// Write plain text to stdout
// 			_, err = os.Stdout.WriteString(text)
// 			if err != nil {
// 				return fmt.Errorf("failed to write plain text to stdout: %w", err)
// 			}
// 		}
// 	} else {
// 		// If no lexer, write plain text to stdout
// 		logger.Printf("Displaying plain text")
// 		_, err = os.Stdout.WriteString(text)
// 		if err != nil {
// 			return fmt.Errorf("failed to write plain text to stdout: %w", err)
// 		}
// 	}

// 	// Optionally, add a footer to indicate end of output
// 	fmt.Println("\n--- End of clipboard content ---")
// 	fmt.Printf("Temp file location: %s (will be deleted)\n", tmpFile.Name())
// 	fmt.Printf("Size: %d bytes\n", len(text))
// 	fmt.Printf("Time: %s\n", time.Now().Format("2006-01-02 15:04:05"))

// 	return nil
// }

// ============================================================================
// TEMP COMMAND (-z) - Display clipboard content with syntax highlighting
// ============================================================================

func handleTempCommand(args []string) error {
	text, err := getClipboardText()
	if err != nil {
		return fmt.Errorf("failed to read clipboard: %w", err)
	}

	if text == "" {
		return fmt.Errorf("clipboard is empty")
	}

	lexerName := ""
	themeName := "monokai"
	usePager := false
	showLineNumbers := true
	showGrid := true

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--lexer", "-l":
			if i+1 < len(args) {
				lexerName = args[i+1]
				i++
			}
		case "--theme", "-t":
			if i+1 < len(args) {
				themeName = args[i+1]
				i++
			}
		case "--pager", "-p":
			usePager = true
		case "--no-line-numbers":
			showLineNumbers = false
		case "--no-grid":
			showGrid = false
		}
	}

	var output bytes.Buffer

	// Header
	output.WriteString(fmt.Sprintf("%s───────┬────────────────────────────────────────────────────────────────%s\n", ColorGray, ColorReset))
	output.WriteString(fmt.Sprintf("%s       │%s %sClipboard Content%s\n", ColorGray, ColorReset, ColorBold, ColorReset))
	output.WriteString(fmt.Sprintf("%s       │%s %sSize:%s %s  %sTime:%s %s\n",
		ColorGray, ColorReset,
		ColorCyan, ColorReset, formatSize(int64(len(text))),
		ColorCyan, ColorReset, time.Now().Format("2006-01-02 15:04:05")))

	if lexerName != "" {
		output.WriteString(fmt.Sprintf("%s       │%s %sLexer:%s %s  %sTheme:%s %s\n",
			ColorGray, ColorReset,
			ColorCyan, ColorReset, lexerName,
			ColorCyan, ColorReset, themeName))
	}

	output.WriteString(fmt.Sprintf("%s───────┼────────────────────────────────────────────────────────────────%s\n", ColorGray, ColorReset))

	// Apply syntax highlighting
	var contentBuf bytes.Buffer
	if lexerName != "" {
		lexer := lexers.Get(lexerName)
		if lexer == nil {
			lexer = lexers.Fallback
		}
		lexer = chroma.Coalesce(lexer)

		style := styles.Get(themeName)
		if style == nil {
			style = styles.Monokai
		}

		formatter := formatters.TTY16m

		iterator, err := lexer.Tokenise(nil, text)
		if err != nil {
			logger.Printf("Warning: failed to tokenize: %v", err)
			contentBuf.WriteString(text)
		} else {
			err = formatter.Format(&contentBuf, style, iterator)
			if err != nil {
				logger.Printf("Warning: failed to format: %v", err)
				contentBuf.WriteString(text)
			}
		}
	} else {
		contentBuf.WriteString(text)
	}

	// Add line numbers
	if showLineNumbers {
		lines := strings.Split(contentBuf.String(), "\n")
		maxLineNum := len(lines)
		lineNumWidth := len(fmt.Sprintf("%d", maxLineNum))

		for i, line := range lines {
			lineNum := i + 1
			if showGrid {
				output.WriteString(fmt.Sprintf("%s%*d │%s %s\n", ColorGray, lineNumWidth, lineNum, ColorReset, line))
			} else {
				output.WriteString(fmt.Sprintf("%s%*d %s %s\n", ColorGray, lineNumWidth, lineNum, ColorReset, line))
			}
		}
	} else {
		output.WriteString(contentBuf.String())
	}

	// Footer
	output.WriteString(fmt.Sprintf("%s───────┴────────────────────────────────────────────────────────────────%s\n", ColorGray, ColorReset))

	if usePager {
		return displayWithPager(output.String())
	} else {
		fmt.Print(output.String())
	}

	return nil
}

// displayWithPager displays content using less or more pager
// func displayWithPager(content string) error {
// 	pagers := []string{"less", "more", "cat"}
// 	var pagerCmd string

// 	for _, p := range pagers {
// 		if _, err := exec.LookPath(p); err == nil {
// 			pagerCmd = p
// 			break
// 		}
// 	}

// 	if pagerCmd == "" {
// 		fmt.Print(content)
// 		return nil
// 	}

// 	var cmd *exec.Cmd
// 	if pagerCmd == "less" {
// 		cmd = exec.Command("less", "-R", "-F", "-X")
// 	} else {
// 		cmd = exec.Command(pagerCmd)
// 	}

// 	pipe, err := cmd.StdinPipe()
// 	if err != nil {
// 		fmt.Print(content)
// 		return nil
// 	}

// 	cmd.Stdout = os.Stdout
// 	cmd.Stderr = os.Stderr

// 	if err := cmd.Start(); err != nil {
// 		fmt.Print(content)
// 		return nil
// 	}

// 	_, err = pipe.Write([]byte(content))
// 	if err != nil {
// 		pipe.Close()
// 		cmd.Wait()
// 		fmt.Print(content)
// 		return nil
// 	}

// 	pipe.Close()
// 	return cmd.Wait()
// }

// displayWithPager displays content using less/more in streaming mode.
func displayWithPager(content string) error {
    pagers := []string{"less", "more"}
    var pagerCmd string

    for _, p := range pagers {
        if _, err := exec.LookPath(p); err == nil {
            pagerCmd = p
            break
        }
    }

    if pagerCmd == "" {
        fmt.Print(content)
        return nil
    }

    var cmd *exec.Cmd
    if pagerCmd == "less" {
        cmd = exec.Command("less", "-R", "-F", "-X")
    } else {
        cmd = exec.Command(pagerCmd)
    }

    stdin, err := cmd.StdinPipe()
    if err != nil {
        fmt.Print(content)
        return nil
    }

    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    if err := cmd.Start(); err != nil {
        fmt.Print(content)
        return nil
    }

    // STREAM content
    go func() {
        defer stdin.Close()

        buf := []byte(content)
        chunkSize := 4096

        for len(buf) > 0 {
            n := chunkSize
            if len(buf) < chunkSize {
                n = len(buf)
            }

            _, err := stdin.Write(buf[:n])
            if err != nil {
                // User likely pressed q → less closed stdin (EPIPE)
                return
            }
            buf = buf[n:]
        }
    }()

    return cmd.Wait()
}

// displayWithPager is a drop-in replacement pager WITHOUT 'less' binary and WITH ANSI support.
// func displayWithPager(content string) error {
// 	// Strip ANSI escape sequences
// 	clean := stripansi.Strip(content)

// 	app := tview.NewApplication()

// 	tv := tview.NewTextView().
// 		SetDynamicColors(true).
// 		SetRegions(true).
// 		SetWrap(true).
// 		SetScrollable(true)

// 	tv.SetText(clean)
// 	tv.ScrollToBeginning()

// 	lines := strings.Split(clean, "\n")
// 	var searchText string

// 	// -------- Scroll Helpers --------
// 	scrollUp := func() {
// 		row, col := tv.GetScrollOffset()
// 		if row > 0 {
// 			tv.ScrollTo(row-1, col)
// 		}
// 	}

// 	scrollDown := func() {
// 		row, col := tv.GetScrollOffset()
// 		tv.ScrollTo(row+1, col)
// 	}

// 	pageUp := func() {
// 		row, col := tv.GetScrollOffset()
// 		newRow := row - 20
// 		if newRow < 0 {
// 			newRow = 0
// 		}
// 		tv.ScrollTo(newRow, col)
// 	}

// 	pageDown := func() {
// 		row, col := tv.GetScrollOffset()
// 		tv.ScrollTo(row+20, col)
// 	}

// 	// -------- Search Functions --------
// 	highlightMatches := func(q string) {
// 		if q == "" {
// 			tv.SetText(clean)
// 			return
// 		}

// 		var out strings.Builder
// 		lq := strings.ToLower(q)

// 		for _, line := range lines {
// 			ll := strings.ToLower(line)
// 			pos := strings.Index(ll, lq)

// 			if pos >= 0 {
// 				out.WriteString(line[:pos])
// 				out.WriteString("[yellow]")
// 				out.WriteString(line[pos : pos+len(q)])
// 				out.WriteString("[-]")
// 				out.WriteString(line[pos+len(q):])
// 			} else {
// 				out.WriteString(line)
// 			}
// 			out.WriteString("\n")
// 		}

// 		tv.SetText(out.String())
// 	}

// 	jumpToFirst := func() {
// 		if searchText == "" {
// 			return
// 		}
// 		lq := strings.ToLower(searchText)
// 		for i, line := range lines {
// 			if strings.Contains(strings.ToLower(line), lq) {
// 				tv.ScrollTo(i, 0)
// 				return
// 			}
// 		}
// 	}

// 	// -------- Key Handler --------
// 	tv.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {

// 		switch ev.Rune() {
// 		case 'q':
// 			app.Stop()
// 			return nil

// 		case 'j':
// 			scrollDown()
// 			return nil

// 		case 'k':
// 			scrollUp()
// 			return nil

// 		case 'g':
// 			tv.ScrollToBeginning()
// 			return nil

// 		case 'G':
// 			tv.ScrollToEnd()
// 			return nil

// 		case '/':
// 			input := tview.NewInputField().
// 				SetLabel("Search: ")

// 			input.SetDoneFunc(func(key tcell.Key) {
// 				searchText = input.GetText()
// 				highlightMatches(searchText)
// 				jumpToFirst()
// 				app.SetRoot(tv, true)
// 			})

// 			app.SetRoot(input, true)
// 			return nil
// 		}

// 		switch ev.Key() {
// 		case tcell.KeyPgUp:
// 			pageUp()
// 			return nil

// 		case tcell.KeyPgDn:
// 			pageDown()
// 			return nil

// 		case tcell.KeyUp:
// 			scrollUp()
// 			return nil

// 		case tcell.KeyDown:
// 			scrollDown()
// 			return nil
// 		}

// 		return ev
// 	})

// 	return app.SetRoot(tv, true).Run()
// }



// handleDiffClipboardToFile reads clipboard, saves to temp file, and diffs with the resolved target file

// ============================================================================
// DIFF COMMAND - Compare files or clipboard
// ============================================================================

func handleDiffClipboardToFile(fileName string) error {
	// 1. Resolve the target file path (including recursive search)
	filePath, err := resolveFilePath(fileName)
	if err != nil {
		return fmt.Errorf("failed to resolve file path: %w", err)
	}

	// 2. Read clipboard content
	clipboardText, err := getClipboardText()
	if err != nil {
		return fmt.Errorf("failed to read clipboard: %w", err)
	}

	if clipboardText == "" {
		return fmt.Errorf("clipboard is empty, nothing to diff")
	}

	// 3. Validate the resolved target file path
	if err := validatePath(filePath); err != nil {
		return fmt.Errorf("invalid resolved file path: %w", err)
	}

	// 4. Create a temporary file
	tempFile, err := os.CreateTemp("", "pt_clipboard_diff_*.txt") // Use a descriptive prefix
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tempFile.Name()) // Clean up the temp file after the function exits
	defer tempFile.Close()

	// 5. Write clipboard content to the temporary file
	_, err = tempFile.WriteString(clipboardText)
	if err != nil {
		return fmt.Errorf("failed to write clipboard content to temporary file: %w", err)
	}
	tempFile.Sync() // Ensure data is written to disk

	logger.Printf("Diffing clipboard content (temp: %s) with resolved file: %s", tempFile.Name(), filePath)

	// 6. Run the core diff logic (runDelta) between the temp file and the resolved target file
	// func runDiff(toolName, file1, file2 string) error {
	// err = runDelta(tempFile.Name(), filePath)
	err = runDiff(difftool, tempFile.Name(), filePath)
	if err != nil {
		// runDelta already handles delta not found error and specific exit codes
		return fmt.Errorf("failed to run diff tool (delta): %w", err)
	}

	return nil
}

// func handleDiffCommand(args []string) error {
// 	if len(args) < 1 {
// 		return fmt.Errorf("filename required for diff command")
// 	}

// 	filename := args[0]
// 	useLast := len(args) > 1 && args[1] == "--last"

// 	filePath, err := resolveFilePath(filename)
// 	if err != nil {
// 		return err
// 	}

// 	backups, err := listBackups(filePath)
// 	if err != nil {
// 		return err
// 	}

// 	if len(backups) == 0 {
// 		return fmt.Errorf("no backups found for: %s (check %s/ directory)", filePath, appConfig.BackupDirName)
// 	}

// 	var selectedBackup BackupInfo

// 	if useLast {
// 		selectedBackup = backups[0]
// 		fmt.Printf("%s📊 Comparing with last backup: %s%s\n\n", ColorCyan, selectedBackup.Name, ColorReset)
// 	} else {
// 		printBackupTable(filePath, backups)

// 		reader := bufio.NewReader(os.Stdin)
// 		fmt.Printf("Enter backup number to compare (1-%d) or 0 to cancel: ", len(backups))

// 		input, err := reader.ReadString('\n')
// 		if err != nil {
// 			return fmt.Errorf("failed to read input: %w", err)
// 		}

// 		input = strings.TrimSpace(input)
// 		choice, err := strconv.Atoi(input)
// 		if err != nil {
// 			return fmt.Errorf("invalid input: please enter a number")
// 		}

// 		if choice < 0 || choice > len(backups) {
// 			return fmt.Errorf("invalid selection: must be between 0 and %d", len(backups))
// 		}

// 		if choice == 0 {
// 			return fmt.Errorf("diff cancelled")
// 		}

// 		selectedBackup = backups[choice-1]
// 		fmt.Printf("\n%s📊 Comparing with: %s%s\n\n", ColorCyan, selectedBackup.Name, ColorReset)
// 	}

// 	switch appConfig.DiffTool {
// 		case "meld", "winmerge", "amerge":
// 			fmt.Printf("appConfig.DiffTool: %s", appConfig.DiffTool)	
// 	}
	

// 	if appConfig.DiffTool == "winmerge" {
// 		err = runWinMerge(selectedBackup.Path, filePath)
// 		if err != nil {
// 			return fmt.Errorf("winmerge execution failed: %w", err)
// 		}		
// 	} else if appConfig.DiffTool == "meld" {
// 		err = runMeld(selectedBackup.Path, filePath)
// 		if err != nil {
// 			return fmt.Errorf("meld execution failed: %w", err)
// 		}
// 	} else if appConfig.DiffTool == "amerge" {
// 		err = runAMerge(selectedBackup.Path, filePath)
// 		if err != nil {
// 			return fmt.Errorf("Araxis merge execution failed: %w", err)
// 		}
// 	} else {
// 		err = runDelta(selectedBackup.Path, filePath)
// 		if err != nil {
// 			return fmt.Errorf("delta execution failed: %w", err)
// 		}
// 	}

// 	return nil
// }

// ==================== DIFF TOOLS CONFIGURATION ====================
type DiffToolConfig struct {
    Name           string   // Tool name (for display)
    Platform       []string // Supported platforms: "linux", "darwin", "windows"
    Type           string   // "CLI", "GUI", "TUI"
    License        string   // "Open Source", "Commercial", "Freeware"
    HomeURL        string   // URL for home page
    InstallURL     string   // URL for install instructions
    BinaryNames    []string // Names of binary possibilities
    NormalExitCode int      // Exit code that is considered normal (0 or 1)
    Args           []string // Additional arguments if needed
}

var diffTools = map[string]DiffToolConfig{
    "delta": {
        Name:           "Delta (git diff)",
        Platform:       []string{"windows", "linux", "darwin"},
        Type:           "CLI",
        License:        "Open Source",
        HomeURL:        "https://dandavison.github.io/delta/",
        InstallURL:     "https://github.com/dandavison/delta#installation",
        BinaryNames:    []string{"delta"},
        NormalExitCode: 1,
    },
    "diff": {
        Name:           "GNU diff",
        Platform:       []string{"linux", "darwin"},
        Type:           "CLI",
        License:        "Open Source",
        HomeURL:        "https://www.gnu.org/software/diffutils/",
        InstallURL:     "https://www.gnu.org/software/diffutils/#downloading",
        BinaryNames:    []string{"diff"},
        NormalExitCode: 1,
        Args:           []string{"-u"},
    },
    "sdiff": {
        Name:           "GNU sdiff",
        Platform:       []string{"linux", "darwin"},
        Type:           "CLI",
        License:        "Open Source",
        HomeURL:        "https://www.gnu.org/software/diffutils/",
        InstallURL:     "https://www.gnu.org/software/diffutils/#downloading",
        BinaryNames:    []string{"sdiff"},
        NormalExitCode: 1,
    },
    "vimdiff": {
        Name:           "vimdiff",
        Platform:       []string{"linux", "darwin"},
        Type:           "CLI (TUI)",
        License:        "Open Source",
        HomeURL:        "https://www.vim.org/",
        InstallURL:     "https://www.vim.org/download.php",
        BinaryNames:    []string{"vimdiff", "nvim", "vim"},
        NormalExitCode: 0,
        Args:           []string{"-d"},
    },
    "meld": {
        Name:           "Meld",
        Platform:       []string{"linux", "darwin", "windows"},
        Type:           "GUI",
        License:        "Open Source",
        HomeURL:        "https://meldmerge.org/",
        InstallURL:     "https://meldmerge.org/#download",
        BinaryNames:    []string{"meld"},
        NormalExitCode: 1,
    },
    "kdiff3": {
        Name:           "KDiff3",
        Platform:       []string{"linux", "darwin", "windows"},
        Type:           "GUI",
        License:        "Open Source",
        HomeURL:        "https://invent.kde.org/sdk/kdiff3",
        InstallURL:     "https://download.kde.org/stable/kdiff3/",
        BinaryNames:    []string{"kdiff3"},
        NormalExitCode: 1,
    },
    "diffmerge": {
        Name:           "DiffMerge",
        Platform:       []string{"linux", "darwin", "windows"},
        Type:           "GUI",
        License:        "Freeware",
        HomeURL:        "https://sourcegear.com/diffmerge/",
        InstallURL:     "https://sourcegear.com/diffmerge/downloads.php",
        BinaryNames:    []string{"diffmerge", "sgdm"},
        NormalExitCode: 1,
    },
    "kompare": {
        Name:           "Kompare",
        Platform:       []string{"linux"},
        Type:           "GUI",
        License:        "Open Source",
        HomeURL:        "https://apps.kde.org/kompare/",
        InstallURL:     "https://apps.kde.org/kompare/",
        BinaryNames:    []string{"kompare"},
        NormalExitCode: 1,
    },
    "tkdiff": {
        Name:           "TkDiff",
        Platform:       []string{"linux", "darwin", "windows"},
        Type:           "GUI",
        License:        "Open Source",
        HomeURL:        "https://sourceforge.net/projects/tkdiff/",
        InstallURL:     "https://sourceforge.net/projects/tkdiff/files/",
        BinaryNames:    []string{"tkdiff"},
        NormalExitCode: 1,
    },
    "bcompare": {
        Name:           "Beyond Compare",
        Platform:       []string{"linux", "darwin", "windows"},
        Type:           "GUI + CLI",
        License:        "Commercial",
        HomeURL:        "https://www.scootersoftware.com/",
        InstallURL:     "https://www.scootersoftware.com/download.php",
        BinaryNames:    []string{"bcompare", "bcomp"},
        NormalExitCode: 1,
    },
    "filemerge": {
        Name:           "FileMerge (Xcode)",
        Platform:       []string{"darwin"},
        Type:           "GUI",
        License:        "Free (Xcode)",
        HomeURL:        "https://developer.apple.com/xcode/",
        InstallURL:     "https://developer.apple.com/download/all/?q=xcode",
        BinaryNames:    []string{"opendiff"},
        NormalExitCode: 0,
    },
    "kaleidoscope": {
        Name:           "Kaleidoscope",
        Platform:       []string{"darwin"},
        Type:           "GUI",
        License:        "Commercial",
        HomeURL:        "https://kaleidoscope.app/",
        InstallURL:     "https://kaleidoscope.app/download",
        BinaryNames:    []string{"ksdiff", "kaleidoscope"},
        NormalExitCode: 1,
    },
}

// ==================== HELPER FUNCTIONS ====================
func findBinary(names []string) (string, bool) {
    for _, name := range names {
        if path, err := exec.LookPath(name); err == nil {
            return path, true
        }
    }
    return "", false
}

func isPlatformCompatible(toolPlatforms []string) bool {
    currentOS := runtime.GOOS
    for _, platform := range toolPlatforms {
        if (platform == "darwin" && currentOS == "darwin") ||
           (platform == "windows" && currentOS == "windows") ||
           (platform == "linux" && currentOS == "linux") {
            return true
        }
    }
    return false
}

// ==================== MAIN DIFF FUNCTION ====================
func runDiff(toolName, file1, file2 string) error {
    // Validate the tool
    config, exists := diffTools[toolName]
    if !exists {
        return fmt.Errorf("diff tool '%s' not supported", toolName)
    }
    
    // Cek platform compatibility
    if !isPlatformCompatible(config.Platform) {
        return fmt.Errorf("%s is not available on %s", config.Name, runtime.GOOS)
    }
    
    // Find binary
    binaryPath, found := findBinary(config.BinaryNames)
    if !found {
        return fmt.Errorf("%s is not installed. Install from: %s", config.Name, config.InstallURL)
    }
    
    // Set up arguments
    args := []string{}
    
    // Handle khusus vim/nvim
    if toolName == "vimdiff" && (filepath.Base(binaryPath) == "vim" || 
                                 filepath.Base(binaryPath) == "nvim") {
        args = append(args, "-d")
    } else if len(config.Args) > 0 {
        args = append(args, config.Args...)
    }
    
    args = append(args, file1, file2)
    
    // Execute command
    cmd := exec.Command(binaryPath, args...)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    cmd.Stdin = os.Stdin
    
    // Handle execution
    err := cmd.Run()
    
    if err != nil {
        if exitErr, ok := err.(*exec.ExitError); ok {
            if exitErr.ExitCode() == config.NormalExitCode {
                return nil
            }
        }
        return fmt.Errorf("failed to run %s: %v", config.Name, err)
    }
    
    return nil
}

// ==================== UPDATED HANDLE DIFF COMMAND ====================
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
        return fmt.Errorf("no backups found for: %s (check %s/ directory)", 
            filePath, appConfig.BackupDirName)
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

    // Use tools from config or default to delta
    toolName := appConfig.DiffTool
    if toolName == "" {
    	if difftool != "" {
    		toolName = difftool
    	} else {
    		toolName = "delta"	
    	}
        
    }
    
    fmt.Printf("%sDiffing use%s %s%s`%s`%s\n", ColorMagenta, ColorReset, ColorWhite, ColorBlue, toolName, ColorReset)

    // Validate the tool before execution
    if _, exists := diffTools[toolName]; !exists {
        fmt.Printf("%sWarning: diff tool '%s' not found, using default 'delta'%s\n", 
            ColorYellow, toolName, ColorReset)
        toolName = "delta"
    }
    
    // Check platform compatibility
    config := diffTools[toolName]
    if !isPlatformCompatible(config.Platform) {
        fmt.Printf("%sWarning: %s not available on %s, using default 'delta'%s\n", 
            ColorYellow, config.Name, runtime.GOOS, ColorReset)
        toolName = "delta"
    }
    
    // Check installation
    if _, found := findBinary(config.BinaryNames); !found {
        return fmt.Errorf("%s is not installed. Install from: %s\n"+
            "You can change diff tool in config file or use: pt config diff_tool <toolname>", 
            config.Name, config.InstallURL)
    }
    
    // Run diff
    err = runDiff(toolName, selectedBackup.Path, filePath)
    if err != nil {
        // Try fallback to delta if the main tool fails
        if toolName != "delta" {
            fmt.Printf("%sTrying fallback to delta...%s\n", ColorYellow, ColorReset)
            err = runDiff("delta", selectedBackup.Path, filePath)
        }
        
        if err != nil {
            return fmt.Errorf("diff execution failed: %w", err)
        }
    }

    return nil
}

// ==================== UTILITY FUNCTIONS ====================
func getAvailableTools() []string {
    available := []string{}
    for name, config := range diffTools {
        if isPlatformCompatible(config.Platform) {
            if _, found := findBinary(config.BinaryNames); found {
                available = append(available, name)
            }
        }
    }
    return available
}

func getSupportedTools() []string {
    supported := []string{}
    for name, config := range diffTools {
        if isPlatformCompatible(config.Platform) {
            supported = append(supported, name)
        }
    }
    return supported
}

func checkToolInstalled(toolName string) bool {
    config, exists := diffTools[toolName]
    if !exists {
        return false
    }
    if !isPlatformCompatible(config.Platform) {
        return false
    }
    _, found := findBinary(config.BinaryNames)
    return found
}

func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}

func listAvailableTools() {
    fmt.Printf("\n%s=== Available Diff Tools (installed) ===%s\n", ColorGreen, ColorReset)
    available := getAvailableTools()
    if len(available) > 0 {
        for _, tool := range available {
            config := diffTools[tool]
            fmt.Printf("  %s• %s%s - %s (%s)\n", 
                ColorCyan, tool, ColorReset, config.Name, config.Type)
        }
    } else {
        fmt.Println("  No diff tools found. Install delta: https://github.com/dandavison/delta")
    }
    
    fmt.Printf("\n%s=== Supported Tools (can be installed) ===%s\n", ColorGreen, ColorReset)
    supported := getSupportedTools()
    for _, tool := range supported {
        if !contains(available, tool) {
            config := diffTools[tool]
            fmt.Printf("  • %s - %s (%s) - %s\n", 
                tool, config.Name, config.Type, config.InstallURL)
        }
    }
}

func checkDeltaInstalled() bool {
	_, err := exec.LookPath("delta")
	return err == nil
}

func checkMeldInstalled() bool {
	_, err := exec.LookPath("meld")
	return err == nil
}

func checkWinMergeInstalled() string {
	if _, err := exec.LookPath("winmerge"); err == nil {
		return "winmerge"
	}

	if _, err := exec.LookPath("WinMergeU"); err == nil {
		return "winmergeu"
	}
	
	// return err == nil
	return ""
}

func checkAMergeInstalled() bool {
	_, err := exec.LookPath("amerge")
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
	} //else {
	// 	fmt.Printf("Error status [DELTA]: %v\n", err)
	// }

	return nil
}

func runMeld(file1, file2 string) error {
	if !checkMeldInstalled() {
		return fmt.Errorf("meld is not installed. Install it from: https://meldmerge.org")
	}

	cmd := exec.Command("meld", file1, file2)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	
	// meld exit code 1 is NORMAL when files are different
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return nil
			}
		}
		return err
	} //else {
	// 	fmt.Printf("Error status [MELD]: %v\n", err)
	// }

	return nil
}

func runWinMerge(file1, file2 string) error {
	exe := checkWinMergeInstalled()
	if exe != "" {
		return fmt.Errorf("winmerge is not installed. Install it from: https://winmerge.org")
	}

	cmd := exec.Command(exe, file1, file2)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	
	// wimerge exit code 1 is NORMAL when files are different
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return nil
			}
		}
		return err
	} //else {
	// 	fmt.Printf("Error status [WINMERGE]: %v\n", err)
	// }

	return nil
}

func runAMerge(file1, file2 string) error {
	exe := checkWinMergeInstalled()
	if exe != "" {
		return fmt.Errorf("winmerge is not installed. Install it from: https://www.araxis.com/merge")
	}

	cmd := exec.Command(exe, file1, file2)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	
	// wimerge exit code 1 is NORMAL when files are different
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return nil
			}
		}
		return err
	} //else {
	// 	fmt.Printf("Error status [AMERGE]: %v\n", err)
	// }

	return nil
}


// ============================================================================
// CHECK/STATUS COMMAND - Show file status (git-like)
// ============================================================================

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
	fmt.Printf("\n%s📊 PT Status%s\n\n", ColorBold+ColorCyan, ColorReset)

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Try to find project root (where .git or .pt is)
	projectRoot := cwd
	ptRoot, err := findPTRoot(cwd)
	if err == nil && ptRoot != "" {
		// If .pt found, use its parent as project root
		if filepath.Base(ptRoot) == appConfig.BackupDirName {
			projectRoot = filepath.Dir(ptRoot)
		} else {
			projectRoot = ptRoot
		}
		logger.Printf("Using project root: %s", projectRoot)
	} else {
		// Try to find .git
		gitRoot := findGitRoot(cwd)
		if gitRoot != "" {
			projectRoot = gitRoot
			logger.Printf("Using git root: %s", projectRoot)
		}
	}

	// Show which directory we're scanning
	relRoot, _ := filepath.Rel(cwd, projectRoot)
	if relRoot != "" && relRoot != "." {
		fmt.Printf("%sScanning from project root:%s %s\n\n", ColorGray, ColorReset, projectRoot)
	}

	// Load gitignore
	gitignore, err := loadGitIgnoreAndPtIgnore(projectRoot)
	if err != nil {
		logger.Printf("Warning: failed to load .gitignore: %v", err)
	}

	exceptions := make(map[string]bool)
	exceptions[appConfig.BackupDirName] = true

	// Build status tree
	tree, err := buildStatusTree(projectRoot, gitignore, exceptions, 0, appConfig.MaxSearchDepth)
	if err != nil {
		return fmt.Errorf("failed to build status tree: %w", err)
	}

	if tree == nil {
		return fmt.Errorf("no files to display")
	}

	// Print tree with status
	fmt.Printf("%s%s%s\n", ColorBold, filepath.Base(projectRoot), ColorReset)
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

// ============================================================================
// COMMIT COMMAND - Backup all changed files
// ============================================================================

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

	// Try to find project root (where .git or .pt is)
	projectRoot := cwd
	ptRoot, err := findPTRoot(cwd)
	if err == nil && ptRoot != "" {
		// If .pt found, use its parent as project root
		if filepath.Base(ptRoot) == appConfig.BackupDirName {
			projectRoot = filepath.Dir(ptRoot)
		} else {
			projectRoot = ptRoot
		}
		logger.Printf("Using project root: %s", projectRoot)
	} else {
		// Try to find .git
		gitRoot := findGitRoot(cwd)
		if gitRoot != "" {
			projectRoot = gitRoot
			logger.Printf("Using git root: %s", projectRoot)
		}
	}

	// Show which directory we're scanning
	relRoot, _ := filepath.Rel(cwd, projectRoot)
	if relRoot != "" && relRoot != "." {
		fmt.Printf("%sCommitting from project root:%s %s\n\n", ColorGray, ColorReset, projectRoot)
	}

	// Load gitignore
	gitignore, err := loadGitIgnoreAndPtIgnore(projectRoot)
	if err != nil {
		logger.Printf("Warning: failed to load .gitignore: %v", err)
	}

	exceptions := make(map[string]bool)
	exceptions[appConfig.BackupDirName] = true

	// Build status tree to find changed files
	tree, err := buildStatusTree(projectRoot, gitignore, exceptions, 0, appConfig.MaxSearchDepth)
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
		relPath, _ := filepath.Rel(projectRoot, file)
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
		relPath, _ := filepath.Rel(projectRoot, file)

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

// ============================================================================
// TREE COMMAND - Display directory tree
// ============================================================================

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

// ============================================================================
// REMOVE COMMAND - Safe file deletion with backup
// ============================================================================

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

	// emptyFile, err := os.Create(filePath)
	// if err != nil {
	// 	return fmt.Errorf("failed to create empty placeholder: %w", err)
	// }
	// emptyFile.Close()

	// logger.Printf("Created empty placeholder: %s", filePath)
	// fmt.Printf("📄 Created empty placeholder: %s\n", filePath)

	// Don't create placeholder - allow restore to recreate the file
	fmt.Printf("💡 Use 'pt -r %s' to restore from backup\n", filepath.Base(filePath))

	fmt.Printf("ℹ️  Original content (%d bytes) backed up to %s/\n", len(content), appConfig.BackupDirName)

	return nil
}

// ============================================================================
// FIX COMMAND - Detect and fix manually moved files
// ============================================================================

func handleFixCommand(args []string) error {
	fmt.Printf("\n🔍 Scanning for orphaned backups...\n\n")
	
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	
	// Find PT root
	ptRoot, err := findPTRoot(cwd)
	if err != nil || ptRoot == "" {
		return fmt.Errorf("no .pt directory found")
	}
	
	fmt.Printf("📂 Using .pt directory: %s\n\n", ptRoot)
	
	// Get parent of .pt
	ptParent := filepath.Dir(ptRoot)
	
	orphaned := make([]OrphanedBackup, 0)
	
	// Walk through all backup directories
	err = filepath.Walk(ptRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		
		if !info.IsDir() {
			return nil
		}
		
		// Skip the root .pt directory itself
		if path == ptRoot {
			return nil
		}
		
		// This is a backup subdirectory
		relPath, _ := filepath.Rel(ptRoot, path)
		
		// Convert backup dir name back to expected file path
		// e.g., "subdir_file.py" -> "subdir/file.py"
		expectedPath := strings.ReplaceAll(relPath, "_", string(os.PathSeparator))
		expectedFullPath := filepath.Join(ptParent, expectedPath)
		
		// Check if the expected file exists
		if _, err := os.Stat(expectedFullPath); os.IsNotExist(err) {
			// File doesn't exist at expected location
			// Try to find it elsewhere
			baseName := filepath.Base(expectedPath)
			matches, _ := findFilesRecursive(baseName, ptParent)
			
			orphaned = append(orphaned, OrphanedBackup{
				BackupDir:    path,
				ExpectedPath: expectedFullPath,
				ActualFiles:  matches,
			})
		}
		
		return nil
	})
	
	if err != nil {
		return err
	}
	
	if len(orphaned) == 0 {
		fmt.Printf("%s✅ No orphaned backups found. All files are in their expected locations.%s\n", 
			ColorGreen, ColorReset)
		return nil
	}
	
	fmt.Printf("%s⚠️  Found %d orphaned backup(s):%s\n\n", ColorYellow, len(orphaned), ColorReset)
	
	for idx, orphan := range orphaned {
		fmt.Printf("[%d] %sOrphaned backup:%s %s\n", 
			idx+1, ColorRed, ColorReset, filepath.Base(orphan.BackupDir))
		fmt.Printf("    Expected: %s (NOT FOUND)\n", orphan.ExpectedPath)
		
		if len(orphan.ActualFiles) > 0 {
			fmt.Printf("    %sPossible matches found:%s\n", ColorGreen, ColorReset)
			for i, match := range orphan.ActualFiles {
				relMatch, _ := filepath.Rel(ptParent, match)
				fmt.Printf("      %d) %s\n", i+1, relMatch)
			}
		} else {
			fmt.Printf("    %sNo matches found (file may be deleted)%s\n", ColorYellow, ColorReset)
		}
		fmt.Println()
	}
	
	// Ask user what to do
	fmt.Println("Options:")
	fmt.Println("  1. Auto-fix: Update backup references for files with single match")
	fmt.Println("  2. Manual: Select correct file for each orphaned backup")
	fmt.Println("  3. Clean: Remove orphaned backups (files deleted)")
	fmt.Println("  0. Cancel")
	
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\nChoice: ")
	input, _ := reader.ReadString('\n')
	choice := strings.TrimSpace(input)
	
	switch choice {
	case "1":
		return autoFixOrphanedBackups(orphaned, ptRoot, ptParent)
	case "2":
		return manualFixOrphanedBackups(orphaned, ptRoot, ptParent)
	case "3":
		return cleanOrphanedBackups(orphaned)
	case "0":
		fmt.Println("❌ Cancelled")
		return nil
	default:
		return fmt.Errorf("invalid choice")
	}
}

func findFilesRecursive(filename string, rootDir string) ([]string, error) {
	matches := make([]string, 0)
	
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		
		// Skip .pt directory
		if info.IsDir() && info.Name() == appConfig.BackupDirName {
			return filepath.SkipDir
		}
		
		if !info.IsDir() && info.Name() == filename {
			matches = append(matches, path)
		}
		
		return nil
	})
	
	return matches, err
}

func autoFixOrphanedBackups(orphaned []OrphanedBackup, ptRoot, ptParent string) error {
	fixed := 0
	skipped := 0
	
	for _, orphan := range orphaned {
		if len(orphan.ActualFiles) == 1 {
			// Only one match, auto-fix
			newPath := orphan.ActualFiles[0]
			newBackupDir, err := getBackupDir(ptRoot, newPath)
			if err != nil {
				skipped++
				continue
			}
			
			// Move backup directory
			if err := os.Rename(orphan.BackupDir, newBackupDir); err != nil {
				skipped++
				continue
			}
			
			// Update metadata
			entries, _ := os.ReadDir(newBackupDir)
			for _, entry := range entries {
				if strings.HasSuffix(entry.Name(), ".meta.json") {
					metaPath := filepath.Join(newBackupDir, entry.Name())
					data, _ := os.ReadFile(metaPath)
					var metadata BackupMetadata
					if json.Unmarshal(data, &metadata) == nil {
						metadata.Original = newPath
						newData, _ := json.MarshalIndent(metadata, "", "  ")
						os.WriteFile(metaPath, newData, 0644)
					}
				}
			}
			
			fmt.Printf("✅ Fixed: %s -> %s\n", 
				filepath.Base(orphan.ExpectedPath), 
				filepath.Base(newPath))
			fixed++
		} else {
			skipped++
		}
	}
	
	fmt.Printf("\n📊 Result: %d fixed, %d skipped\n", fixed, skipped)
	return nil
}

func manualFixOrphanedBackups(orphaned []OrphanedBackup, ptRoot, ptParent string) error {
	// Implementation for manual selection
	fmt.Println("Manual fix not yet implemented. Use auto-fix or clean.")
	return nil
}

func cleanOrphanedBackups(orphaned []OrphanedBackup) error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("\n⚠️  This will DELETE %d backup directories. Continue? (yes/no): ", len(orphaned))
	input, _ := reader.ReadString('\n')
	
	if strings.TrimSpace(strings.ToLower(input)) != "yes" {
		fmt.Println("❌ Cancelled")
		return nil
	}
	
	cleaned := 0
	for _, orphan := range orphaned {
		if err := os.RemoveAll(orphan.BackupDir); err == nil {
			fmt.Printf("🗑️  Removed: %s\n", filepath.Base(orphan.BackupDir))
			cleaned++
		}
	}
	
	fmt.Printf("\n✅ Cleaned %d orphaned backup(s)\n", cleaned)
	return nil
}

// ============================================================================
// MOVE COMMAND - Move file and adjust all backups
// ============================================================================

// ============================================================================
// MOVE COMMAND - Move file(s) and adjust all backups
// ============================================================================

func handleMoveCommand(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("move requires at least source and destination: pt move <source...> <destination>")
	}

	comment := ""
	patterns := []string{}
	recursive := false
	
	// Parse arguments - last non-flag arg is destination
	i := 0
	for i < len(args) {
		if args[i] == "-m" || args[i] == "--message" {
			if i+1 >= len(args) {
				return fmt.Errorf("-m/--message requires a value")
			}
			i++
			comment = args[i]
			i++
			continue
		}
		if args[i] == "-r" || args[i] == "--recursive" {
			recursive = true
			i++
			continue
		}
		patterns = append(patterns, args[i])
		i++
	}

	if len(patterns) < 2 {
		return fmt.Errorf("need at least source and destination")
	}

	// Last pattern is destination
	destPath := patterns[len(patterns)-1]
	sourcePatterns := patterns[:len(patterns)-1]
	
	// Check if we're moving a directory (single source, no wildcards)
	if len(sourcePatterns) == 1 && !strings.Contains(sourcePatterns[0], "*") && !strings.HasPrefix(sourcePatterns[0], "regex:") && !strings.HasPrefix(sourcePatterns[0], "r:") {
		if info, err := os.Stat(sourcePatterns[0]); err == nil && info.IsDir() {
			if recursive {
				return moveDirectoryWithBackups(sourcePatterns[0], destPath, comment)
			} else {
				return fmt.Errorf("use -r flag to move directories: pt move -r %s %s", sourcePatterns[0], destPath)
			}
		}
	}
	
	// Expand wildcards and regex patterns
	logger.Printf("Source patterns before expansion: %v", sourcePatterns)
	sourceFiles, err := expandGlobs(sourcePatterns)
	logger.Printf("Source files after expansion: %v", sourceFiles)
	
	if err != nil {
		return fmt.Errorf("pattern expansion failed: %w", err)
	}
	
	if len(sourceFiles) == 0 {
		return fmt.Errorf("no files matched the patterns: %v", sourcePatterns)
	}
	
	// Additional check: if we got back the exact same patterns (no expansion happened),
	// and they contain wildcards, it means no files matched
	if len(sourceFiles) == len(sourcePatterns) {
		allUnexpanded := true
		for i, f := range sourceFiles {
			if f != sourcePatterns[i] {
				allUnexpanded = false
				break
			}
		}
		if allUnexpanded {
			// Check if any pattern contains wildcards
			for _, pattern := range sourcePatterns {
				if strings.Contains(pattern, "*") || strings.Contains(pattern, "?") {
					return fmt.Errorf("no files matched pattern: %s", pattern)
				}
			}
		}
	}
	
	if len(sourceFiles) > 1 {
		fmt.Printf("🎯 Matched %d file(s) from patterns\n", len(sourceFiles))
	}

	// Resolve destination
	destResolved, err := filepath.Abs(destPath)
	if err != nil {
		return fmt.Errorf("invalid destination path: %w", err)
	}

	// Check if destination exists and is a directory
	destIsDir := false
	if destInfo, err := os.Stat(destResolved); err == nil {
		if !destInfo.IsDir() {
			// Destination exists but is not a directory
			if len(sourceFiles) > 1 {
				return fmt.Errorf("destination must be a directory when moving multiple files")
			}
			// Single file to existing file - not allowed
			return fmt.Errorf("destination already exists: %s", destResolved)
		}
		destIsDir = true
	} else {
		// Destination doesn't exist
		if len(sourceFiles) > 1 {
			// Multiple files - destination must be a directory, create it
			if err := os.MkdirAll(destResolved, 0755); err != nil {
				return fmt.Errorf("failed to create destination directory: %w", err)
			}
			destIsDir = true
			fmt.Printf("📁 Created destination directory: %s\n", destResolved)
		}
		// Single file - destination will be the new filename
	}

	fmt.Printf("\n🚚 Moving %d file(s) with backup adjustment...\n", len(sourceFiles))
	fmt.Printf("  Destination: %s\n", destResolved)
	if destIsDir {
		fmt.Printf("  Type: Directory\n")
	}
	fmt.Println()

	// Track results
	successCount := 0
	failCount := 0
	movedBackups := 0

	// Process each source file
	for idx, sourcePath := range sourceFiles {
		fileNum := idx + 1
		fmt.Printf("[%d/%d] Processing: %s\n", fileNum, len(sourceFiles), sourcePath)

		// Resolve source file
		sourceResolved, err := resolveFilePath(sourcePath)
		if err != nil {
			fmt.Printf("  %s❌ Source not found: %v%s\n", ColorRed, err, ColorReset)
			failCount++
			continue
		}

		// Check if source exists and is a file
		sourceInfo, err := os.Stat(sourceResolved)
		if err != nil {
			fmt.Printf("  %s❌ Cannot stat: %v%s\n", ColorRed, err, ColorReset)
			failCount++
			continue
		}

		if sourceInfo.IsDir() {
			fmt.Printf("  %s❌ Cannot move directories%s\n", ColorRed, ColorReset)
			failCount++
			continue
		}

		// Determine final destination path
		var finalDestPath string
		if destIsDir {
			finalDestPath = filepath.Join(destResolved, filepath.Base(sourceResolved))
		} else {
			finalDestPath = destResolved
		}

		// Check if destination already exists
		if _, err := os.Stat(finalDestPath); err == nil {
			fmt.Printf("  %s❌ Destination exists: %s%s\n", ColorRed, finalDestPath, ColorReset)
			failCount++
			continue
		}

		// Validate destination path
		if err := validatePath(finalDestPath); err != nil {
			fmt.Printf("  %s❌ Invalid destination: %v%s\n", ColorRed, err, ColorReset)
			failCount++
			continue
		}

		// Find PT root for source
		sourcePTRoot, err := findPTRoot(filepath.Dir(sourceResolved))
		if err != nil {
			fmt.Printf("  %s⚠️  No PT root for source%s\n", ColorYellow, ColorReset)
		}

		// Get source backup directory
		var sourceBackupDir string
		hasBackups := false
		if sourcePTRoot != "" {
			sourceBackupDir, err = getBackupDir(sourcePTRoot, sourceResolved)
			if err == nil {
				if info, err := os.Stat(sourceBackupDir); err == nil && info.IsDir() {
					entries, _ := os.ReadDir(sourceBackupDir)
					if len(entries) > 0 {
						hasBackups = true
						fmt.Printf("  📦 Found %d backup(s)\n", len(entries)/2)
					}
				}
			}
		}

		// Ensure destination parent directory exists
		destDir := filepath.Dir(finalDestPath)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			fmt.Printf("  %s❌ Cannot create dest dir: %v%s\n", ColorRed, err, ColorReset)
			failCount++
			continue
		}

		// Find or create PT root for destination
		destPTRoot, err := ensurePTDir(finalDestPath)
		if err != nil {
			fmt.Printf("  %s❌ Cannot ensure PT dir: %v%s\n", ColorRed, err, ColorReset)
			failCount++
			continue
		}

		// Get destination backup directory
		destBackupDir, err := getBackupDir(destPTRoot, finalDestPath)
		if err != nil {
			fmt.Printf("  %s❌ Cannot get dest backup dir: %v%s\n", ColorRed, err, ColorReset)
			failCount++
			continue
		}

		// Move backups first (if they exist)
		if hasBackups {
			// Ensure destination backup parent directory exists
			if err := os.MkdirAll(filepath.Dir(destBackupDir), 0755); err != nil {
				fmt.Printf("  %s⚠️  Cannot create backup parent: %v%s\n", ColorYellow, err, ColorReset)
			} else {
				// Move the entire backup directory
				err = os.Rename(sourceBackupDir, destBackupDir)
				if err != nil {
					fmt.Printf("  %s⚠️  Failed to move backups: %v%s\n", ColorYellow, err, ColorReset)
				} else {
					// Update metadata in all backup files
					entries, err := os.ReadDir(destBackupDir)
					if err == nil {
						updatedCount := 0
						for _, entry := range entries {
							if strings.HasSuffix(entry.Name(), ".meta.json") {
								metaPath := filepath.Join(destBackupDir, entry.Name())
								data, err := os.ReadFile(metaPath)
								if err != nil {
									continue
								}

								var metadata BackupMetadata
								if err := json.Unmarshal(data, &metadata); err != nil {
									continue
								}

								// Update original file path
								metadata.Original = finalDestPath

								newData, err := json.MarshalIndent(metadata, "", "  ")
								if err != nil {
									continue
								}

								if err := os.WriteFile(metaPath, newData, 0644); err == nil {
									updatedCount++
								}
							}
						}
						fmt.Printf("  ✅ Moved backups (%d metadata updated)\n", updatedCount)
						movedBackups += len(entries) / 2
					}
				}
			}
		}

		// Move the actual file
		err = os.Rename(sourceResolved, finalDestPath)
		if err != nil {
			// If move fails, try to restore backups
			if hasBackups {
				os.Rename(destBackupDir, sourceBackupDir)
			}
			fmt.Printf("  %s❌ Failed to move file: %v%s\n", ColorRed, err, ColorReset)
			failCount++
			continue
		}

		// Create backup of the move operation if comment provided
		if comment != "" {
			_, err = autoRenameIfExists(finalDestPath, "move: "+comment)
			if err != nil {
				logger.Printf("Warning: failed to create move backup for %s: %v", finalDestPath, err)
			}
		}

		// Show both source and destination names
		srcName := filepath.Base(sourceResolved)
		destName := filepath.Base(finalDestPath)
		
		// Show relative path or just filename if in same dir
		var displayPath string
		if rel, err := filepath.Rel(".", finalDestPath); err == nil && rel != "" {
			displayPath = rel
		} else {
			displayPath = finalDestPath
		}
		
		if srcName == destName {
			// Same filename, different directory
			fmt.Printf("  %s✅ Moved to: %s%s\n", ColorGreen, displayPath, ColorReset)
		} else {
			// Renamed
			fmt.Printf("  %s✅ Renamed and moved to: %s%s\n", ColorGreen, displayPath, ColorReset)
		}
		successCount++
	}

	// Summary
	fmt.Println()
	fmt.Printf("%s📊 Move Summary:%s\n", ColorBold, ColorReset)
	fmt.Printf("  %s✅ %d file(s) moved successfully%s\n", ColorGreen, successCount, ColorReset)
	if failCount > 0 {
		fmt.Printf("  %s❌ %d file(s) failed%s\n", ColorRed, failCount, ColorReset)
	}
	if movedBackups > 0 {
		fmt.Printf("  📦 %d backup(s) adjusted\n", movedBackups)
	}
	if comment != "" {
		fmt.Printf("  💬 Comment: \"%s\"\n", comment)
	}

	if failCount > 0 {
		return fmt.Errorf("%d file(s) failed to move", failCount)
	}

	return nil
}


// moveDirectoryWithBackups moves entire directory and adjusts all backups
func moveDirectoryWithBackups(sourceDir, destDir string, comment string) error {
	// Resolve source directory
	sourceResolved, err := filepath.Abs(sourceDir)
	if err != nil {
		return fmt.Errorf("invalid source path: %w", err)
	}
	
	sourceInfo, err := os.Stat(sourceResolved)
	if err != nil {
		return fmt.Errorf("source not found: %w", err)
	}
	
	if !sourceInfo.IsDir() {
		return fmt.Errorf("source is not a directory: %s", sourceResolved)
	}
	
	// Resolve destination
	destResolved, err := filepath.Abs(destDir)
	if err != nil {
		return fmt.Errorf("invalid destination path: %w", err)
	}
	
	// Check if destination exists
	if _, err := os.Stat(destResolved); err == nil {
		return fmt.Errorf("destination already exists: %s", destResolved)
	}
	
	fmt.Printf("\n🚚 Moving directory with backup adjustment...\n")
	fmt.Printf("  Source: %s\n", sourceResolved)
	fmt.Printf("  Destination: %s\n", destResolved)
	fmt.Println()
	
	// Find all files in source directory recursively
	var filesToMove []string
	err = filepath.Walk(sourceResolved, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			filesToMove = append(filesToMove, path)
		}
		return nil
	})
	
	if err != nil {
		return fmt.Errorf("failed to walk source directory: %w", err)
	}
	
	if len(filesToMove) == 0 {
		return fmt.Errorf("no files found in source directory")
	}
	
	fmt.Printf("📊 Found %d file(s) to move\n\n", len(filesToMove))
	
	// Find PT root for source
	sourcePTRoot, err := findPTRoot(sourceResolved)
	if err != nil {
		logger.Printf("Warning: failed to find PT root for source: %v", err)
	}
	
	// Create destination directory structure first
	if err := os.MkdirAll(destResolved, 0755); err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}
	
	// Track results
	successCount := 0
	failCount := 0
	movedBackups := 0
	
	// Process each file
	for idx, sourcePath := range filesToMove {
		fileNum := idx + 1
		relPath, _ := filepath.Rel(sourceResolved, sourcePath)
		fmt.Printf("[%d/%d] %s\n", fileNum, len(filesToMove), relPath)
		
		// Calculate destination path (preserve directory structure)
		destPath := filepath.Join(destResolved, relPath)
		
		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			fmt.Printf("  %s❌ Cannot create parent dir: %v%s\n", ColorRed, err, ColorReset)
			failCount++
			continue
		}
		
		// Check if file has backups
		var sourceBackupDir string
		hasBackups := false
		if sourcePTRoot != "" {
			sourceBackupDir, err = getBackupDir(sourcePTRoot, sourcePath)
			if err == nil {
				if info, err := os.Stat(sourceBackupDir); err == nil && info.IsDir() {
					entries, _ := os.ReadDir(sourceBackupDir)
					if len(entries) > 0 {
						hasBackups = true
						fmt.Printf("  📦 %d backup(s)\n", len(entries)/2)
					}
				}
			}
		}
		
		// Get destination PT root and backup dir
		destPTRoot, err := ensurePTDir(destPath)
		if err != nil {
			fmt.Printf("  %s❌ Cannot ensure PT dir: %v%s\n", ColorRed, err, ColorReset)
			failCount++
			continue
		}
		
		destBackupDir, err := getBackupDir(destPTRoot, destPath)
		if err != nil {
			fmt.Printf("  %s❌ Cannot get backup dir: %v%s\n", ColorRed, err, ColorReset)
			failCount++
			continue
		}
		
		// Move backups if they exist
		if hasBackups {
			if err := os.MkdirAll(filepath.Dir(destBackupDir), 0755); err == nil {
				if err := os.Rename(sourceBackupDir, destBackupDir); err == nil {
					// Update metadata
					entries, _ := os.ReadDir(destBackupDir)
					for _, entry := range entries {
						if strings.HasSuffix(entry.Name(), ".meta.json") {
							metaPath := filepath.Join(destBackupDir, entry.Name())
							data, _ := os.ReadFile(metaPath)
							var metadata BackupMetadata
							if json.Unmarshal(data, &metadata) == nil {
								metadata.Original = destPath
								newData, _ := json.MarshalIndent(metadata, "", "  ")
								os.WriteFile(metaPath, newData, 0644)
							}
						}
					}
					fmt.Printf("  ✅ Backups moved\n")
					movedBackups += len(entries) / 2
				}
			}
		}
		
		// Move the file
		if err := os.Rename(sourcePath, destPath); err != nil {
			fmt.Printf("  %s❌ Move failed: %v%s\n", ColorRed, err, ColorReset)
			failCount++
			continue
		}
		
		fmt.Printf("  %s✅ Moved%s\n", ColorGreen, ColorReset)
		successCount++
	}
	
	// Remove empty source directory
	os.RemoveAll(sourceResolved)
	
	fmt.Println()
	fmt.Printf("%s📊 Directory Move Summary:%s\n", ColorBold, ColorReset)
	fmt.Printf("  %s✅ %d file(s) moved%s\n", ColorGreen, successCount, ColorReset)
	if failCount > 0 {
		fmt.Printf("  %s❌ %d file(s) failed%s\n", ColorRed, failCount, ColorReset)
	}
	if movedBackups > 0 {
		fmt.Printf("  📦 %d backup(s) adjusted\n", movedBackups)
	}
	if comment != "" {
		fmt.Printf("  💬 Comment: \"%s\"\n", comment)
	}
	
	return nil
}

// ============================================================================
// BACKUP & RESTORE OPERATIONS
// ============================================================================

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

	// Check if original file exists
	fileExists := false
	if _, err := os.Stat(originalPath); err == nil {
		fileExists = true
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

	// if _, err := os.Stat(originalPath); err == nil {
	// 	if comment == "" {
	// 		comment = "Backup before restore"
	// 	}
	// 	_, err = autoRenameIfExists(originalPath, comment)
	// 	if err != nil {
	// 		return fmt.Errorf("failed to backup current file: %w", err)
	// 	}
	// }

	if fileExists {
		if comment == "" {
			comment = "Backup before restore"
		}
		_, err = autoRenameIfExists(originalPath, comment)
		if err != nil {
			return fmt.Errorf("failed to backup current file: %w", err)
		}
		fmt.Printf("📦 Current file backed up before restore\n")
	} else {
		fmt.Printf("📄 File was deleted, recreating from backup\n")
		// Ensure parent directory exists
		dir := filepath.Dir(originalPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
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

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

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

// func findConfigFile() string {
// 	configNames := []string{"pt.yml", "pt.yaml", ".pt.yml", ".pt.yaml"}

// 	searchPaths := []string{
// 		".",
// 		filepath.Join(os.Getenv("HOME"), ".config", "pt"),
// 		os.Getenv("HOME"),
// 	}

// 	if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
// 		searchPaths = append(searchPaths, userProfile, filepath.Join(userProfile, ".pt"))
// 	}

// 	for _, basePath := range searchPaths {
// 		for _, configName := range configNames {
// 			configPath := filepath.Join(basePath, configName)
// 			if _, err := os.Stat(configPath); err == nil {
// 				return configPath
// 			}
// 		}
// 	}

// 	return ""
// }

func findConfigFile() string {
    configNames := []string{"pt.yml", "pt.yaml", ".pt.yml", ".pt.yaml"}
    
    var searchPaths []string
    
    runtimeOS := runtime.GOOS
    exeDir, _ :=	 os.Executable()
    exeDir = filepath.Dir(exeDir)
    currentDir, _ := os.Getwd()
    
    switch runtimeOS {
    case "windows":
        // Windows search paths
        if appData := os.Getenv("APPDATA"); appData != "" {
            searchPaths = append(searchPaths,
                filepath.Join(appData, ".pt"),  // %APPDATA%/.pt/
                appData,                        // %APPDATA%/
            )
        }
        
        if programData := os.Getenv("PROGRAMDATA"); programData != "" {
            searchPaths = append(searchPaths,
                filepath.Join(programData, ".pt"),  // %PROGRAMDATA%/.pt/
                programData,                        // %PROGRAMDATA%/
            )
        }
        
        if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
            searchPaths = append(searchPaths,
                filepath.Join(userProfile, ".pt"),  // %USERPROFILE%/.pt/
            )
        }
        
        if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
            searchPaths = append(searchPaths,
                filepath.Join(localAppData, ".pt"),  // %LOCALAPPDATA%/.pt/
                localAppData,                         // %LOCALAPPDATA%/
            )
        }
        
        // Executable directory
        searchPaths = append(searchPaths,
            filepath.Join(exeDir, ".pt"),  // exedir/.pt/
            exeDir,                        // exedir/
        )
        
        // Current directory
        searchPaths = append(searchPaths,
            filepath.Join(currentDir, ".pt"),  // currentdir/.pt/
            currentDir,                        // currentdir/
        )
        
    case "darwin":  // macOS
        home := os.Getenv("HOME")
        
        // macOS specific paths
        if home != "" {
            // User-level configs
            searchPaths = append(searchPaths,
                filepath.Join(home, ".config", ".pt"),  // ~/.config/.pt/
                filepath.Join(home, ".config"),         // ~/.config/
                filepath.Join(home, ".pt"),             // ~/.pt/
                home,                                   // ~/
                filepath.Join(home, "Library", "Application Support", ".pt"), // ~/Library/Application Support/.pt/
                filepath.Join(home, "Library", "Application Support"),        // ~/Library/Application Support/
            )
        }
        
        // System-level configs
        searchPaths = append(searchPaths,
            filepath.Join("/etc", ".pt"),           // /etc/.pt/
            "/etc",                                 // /etc/
            filepath.Join("/usr", "etc", ".pt"),    // /usr/etc/.pt/
            filepath.Join("/usr", "etc"),           // /usr/etc/
            filepath.Join("/usr", "local", "etc", ".pt"),  // /usr/local/etc/.pt/
            filepath.Join("/usr", "local", "etc"),         // /usr/local/etc/
        )
        
        // Executable directory
        searchPaths = append(searchPaths,
            filepath.Join(exeDir, ".pt"),  // exedir/.pt/
            exeDir,                        // exedir/
        )
        
        // Current directory
        searchPaths = append(searchPaths,
            filepath.Join(currentDir, ".pt"),  // currentdir/.pt/
            currentDir,                        // currentdir/
        )
        
    default:  // Linux and other Unix-like
        home := os.Getenv("HOME")
        
        if home != "" {
            // XDG Base Directory Specification + legacy
            if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
                searchPaths = append(searchPaths,
                    filepath.Join(xdgConfigHome, ".pt"),  // $XDG_CONFIG_HOME/.pt/
                    xdgConfigHome,                        // $XDG_CONFIG_HOME/
                )
            } else {
                searchPaths = append(searchPaths,
                    filepath.Join(home, ".config", ".pt"),  // $HOME/.config/.pt/
                    filepath.Join(home, ".config"),         // $HOME/.config/
                )
            }
            
            searchPaths = append(searchPaths,
                filepath.Join(home, ".pt"),  // $HOME/.pt/
                home,                        // $HOME/
            )
        }
        
        // System-level configs
        searchPaths = append(searchPaths,
            filepath.Join("/etc", ".pt"),           // /etc/.pt/
            "/etc",                                 // /etc/
            filepath.Join("/usr", "etc", ".pt"),    // /usr/etc/.pt/
            filepath.Join("/usr", "etc"),           // /usr/etc/
            filepath.Join("/usr", "local", "etc", ".pt"),  // /usr/local/etc/.pt/
            filepath.Join("/usr", "local", "etc"),         // /usr/local/etc/
        )
        
        // Executable directory
        searchPaths = append(searchPaths,
            filepath.Join(exeDir, ".pt"),  // exedir/.pt/
            exeDir,                        // exedir/
        )
        
        // Current directory
        searchPaths = append(searchPaths,
            filepath.Join(currentDir, ".pt"),  // currentdir/.pt/
            currentDir,                        // currentdir/
        )
    }
    
    // Remove duplicates while preserving order
    // fmt.Printf("searchPaths: %s", searchPaths)
    uniquePaths := make([]string, 0, len(searchPaths))
    seen := make(map[string]bool)
    for _, path := range searchPaths {
        if !seen[path] {
            seen[path] = true
            uniquePaths = append(uniquePaths, path)
        }
    }

    // fmt.Printf("uniquePaths: %s", uniquePaths)
    
    // Search for config file
    for _, basePath := range uniquePaths {
        for _, configName := range configNames {
            configPath := filepath.Join(basePath, configName)
            if _, err := os.Stat(configPath); err == nil {
            	// fmt.Printf("configPath: %s", configPath)
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

// findPTRoot searches for .pt or .git directory in current and parent directories (like .git)
// It starts from the given path and walks up the directory tree until it finds .pt or .git or reaches root.
// If .pt is found, returns its path.
// If .git is found (and no .pt was found above it), returns the parent directory of .git (where .pt should be).
// If neither is found, returns "".
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
	// Search up the directory tree until we find .pt or .git or reach filesystem root
	for {
		// Check the .pt first
		ptDir := filepath.Join(current, appConfig.BackupDirName)
		if info, err := os.Stat(ptDir); err == nil && info.IsDir() {
			logger.Printf("Found %s directory at: %s", appConfig.BackupDirName, ptDir)
			return ptDir, nil // Return the FULL PATH to the found .pt
		}

		// Cek .git
		gitDir := filepath.Join(current, ".git")
		if info, err := os.Stat(gitDir); err == nil && (info.IsDir() || info.Mode().IsRegular()) {
			// logger.Printf("Found .git directory/file at: %s", gitDir)
			// Return the directory WHERE .git IS located (not the path to .git itself)
			// logger.Printf("Will use parent of .git for %s: %s", appConfig.BackupDirName, current)
			return current, nil // <-- Main change: return 'current' instead of 'gitDir'
		}

		parent := filepath.Dir(current)
		// Reached filesystem root (parent == current means we can't go up anymore)
		if parent == current {
			break
		}
		current = parent
	}
	// No .pt or .git directory found in any parent
	// logger.Printf("No %s or .git directory found in tree from: %s", appConfig.BackupDirName, absPath)
	logger.Printf("No %s directory found in tree from: %s", appConfig.BackupDirName, absPath)
	return "", nil
}

func findGitRoot(startPath string) string {
	current := startPath
	absPath, err := filepath.Abs(current)
	if err != nil {
		return ""
	}
	current = absPath

	for {
		gitDir := filepath.Join(current, ".git")
		if info, err := os.Stat(gitDir); err == nil && (info.IsDir() || info.Mode().IsRegular()) {
			logger.Printf("Found .git at: %s", gitDir)
			return current
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return ""
}

// ensurePTDir creates .pt directory if it doesn't exist
// Returns the absolute path to the .pt directory (could be in parent dir)
// This function mimics git behavior - searches upward for existing .pt or .git
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

	// Try to find existing .pt directory or the parent directory indicated by .git by walking up the tree
	ptRootResult, err := findPTRoot(dir)
	if err != nil {
		return "", err
	}

	// If findPTRoot found an existing .pt directory (not just the parent for a new one)
	// ptRootResult will be the path to the .pt directory itself.
	// If findPTRoot found .git or reached root without finding either,
	// ptRootResult will be the directory *where .pt should be created*.
	// We need to differentiate.

	if ptRootResult != "" {
		// Check if ptRootResult is actually the path to an existing .pt directory
		ptBaseName := filepath.Base(ptRootResult)
		if ptBaseName == appConfig.BackupDirName {
			// Yes, ptRootResult is the existing .pt directory path
			logger.Printf("Using existing %s from parent tree: %s", appConfig.BackupDirName, ptRootResult)
			// Print relative path from current working directory for user clarity
			cwd, _ := os.Getwd()
			relPath, _ := filepath.Rel(cwd, ptRootResult)
			if relPath != "" && relPath != "." {
				fmt.Printf("📁 Using existing %s from: %s%s/%s", appConfig.BackupDirName, ColorCyan, relPath, ColorReset)
			}
			return ptRootResult, nil
		} else {
			// ptRootResult is the directory where .pt should be created (e.g., where .git was found)
			// logger.Printf("Found parent context (.git or root) at: %s. Will create %s here.", ptRootResult, appConfig.BackupDirName)
			// Proceed to create .pt in ptRootResult
			absDir := ptRootResult // Use the path returned by findPTRoot as the base directory
			ptDir := filepath.Join(absDir, appConfig.BackupDirName)

			// Check if .pt directory exists at this level (this handles the case where findPTRoot returned a parent, and .pt was created there between calls)
			info, err = os.Stat(ptDir)
			if os.IsNotExist(err) {
				// Create .pt directory with appropriate permissions (0755)
				// On Unix-like systems, the leading dot makes it conventionally hidden.
				// On Windows, we need to explicitly set the hidden attribute after creation.
				err = os.Mkdir(ptDir, 0755) // Use Mkdir instead of MkdirAll for the single directory
				if err != nil {
					return "", fmt.Errorf("failed to create %s directory: %w", appConfig.BackupDirName, err)
				}
				logger.Printf("Created %s directory: %s", appConfig.BackupDirName, ptDir)
				fmt.Printf("📁 Created %s directory: %s", appConfig.BackupDirName, ptDir)

				// Set hidden attribute on Windows
				if runtime.GOOS == "windows" {
					err = setWindowsHiddenAttribute(ptDir)
					if err != nil {
						// Log the error but don't fail the operation, as the directory was created.
						logger.Printf("Warning: failed to set hidden attribute on Windows: %v", err)
					} else {
						logger.Printf("Set hidden attribute on Windows for: %s", ptDir)
					}
				}

				// Create .gitignore to ignore .pt directory in the *same parent directory* (absDir)
				createPTGitignore(absDir)
			} else if err != nil {
				return "", fmt.Errorf("failed to check %s directory: %w", appConfig.BackupDirName, err)
			} else if !info.IsDir() {
				return "", fmt.Errorf("%s exists but is not a directory: %s", appConfig.BackupDirName, ptDir)
			}
			// Return the path to the .pt directory we found or created
			return ptDir, nil
		}
	} else {
		// No .pt or .git found anywhere in the parent tree, create .pt in the immediate directory of the file
		// logger.Printf("No .pt or .git found in tree. Creating %s in file's directory: %s", appConfig.BackupDirName, dir)
		logger.Printf("No .pt found in tree. Creating %s in file's directory: %s", appConfig.BackupDirName, dir)
		// Get the absolute path of the directory where we'll create .pt
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return "", err
		}
		ptDir := filepath.Join(absDir, appConfig.BackupDirName)

		// Check if .pt directory exists at this level
		info, err = os.Stat(ptDir)
		if os.IsNotExist(err) {
			// Create .pt directory with appropriate permissions (0755)
			err = os.Mkdir(ptDir, 0755) // Use Mkdir instead of MkdirAll for the single directory
			if err != nil {
				return "", fmt.Errorf("failed to create %s directory: %w", appConfig.BackupDirName, err)
			}
			logger.Printf("Created %s directory: %s", appConfig.BackupDirName, ptDir)
			fmt.Printf("📁 Created %s directory: %s", appConfig.BackupDirName, ptDir)

			// Set hidden attribute on Windows
			if runtime.GOOS == "windows" {
				err = setWindowsHiddenAttribute(ptDir)
				if err != nil {
					// Log the error but don't fail the operation, as the directory was created.
					logger.Printf("Warning: failed to set hidden attribute on Windows: %v", err)
				} //else {
				// 	logger.Printf("Set hidden attribute on Windows for: %s", ptDir)
				// }
			}

			// Create .gitignore to ignore .pt directory in the *same parent directory* (absDir)
			createPTGitignore(absDir)
		} else if err != nil {
			return "", fmt.Errorf("failed to check %s directory: %w", appConfig.BackupDirName, err)
		} else if !info.IsDir() {
			return "", fmt.Errorf("%s exists but is not a directory: %s", appConfig.BackupDirName, ptDir)
		}
		// Return the path to the .pt directory we created
		return ptDir, nil
	}
}

// expandGlobs expands wildcard patterns and returns list of matching files
func expandGlobs(patterns []string) ([]string, error) {
	files := make([]string, 0)
	seen := make(map[string]bool)
	
	for _, pattern := range patterns {
		logger.Printf("Processing pattern: '%s'", pattern)
		
		// Check if it's a regex pattern (starts with regex: or r:)
		if strings.HasPrefix(pattern, "regex:") || strings.HasPrefix(pattern, "r:") {
			regexPattern := strings.TrimPrefix(pattern, "regex:")
			regexPattern = strings.TrimPrefix(regexPattern, "r:")
			
			// Search current directory recursively for regex matches
			matches, err := findFilesWithRegex(regexPattern)
			if err != nil {
				return nil, fmt.Errorf("regex error in '%s': %w", pattern, err)
			}
			logger.Printf("Regex '%s' matched %d files", pattern, len(matches))
			for _, match := range matches {
				absMatch, _ := filepath.Abs(match)
				if !seen[absMatch] {
					files = append(files, match)
					seen[absMatch] = true
				}
			}
		} else if strings.Contains(pattern, "*") || strings.Contains(pattern, "?") || strings.Contains(pattern, "[") {
			// It's a glob pattern
			logger.Printf("Treating as glob pattern: '%s'", pattern)
			
			// Try filepath.Glob first
			matches, err := filepath.Glob(pattern)
			if err != nil {
				return nil, fmt.Errorf("invalid glob pattern '%s': %w", pattern, err)
			}
			
			logger.Printf("Glob matched %d files", len(matches))
			
			// Filter out directories
			for _, match := range matches {
				if info, err := os.Stat(match); err == nil {
					if info.IsDir() {
						logger.Printf("Skipping directory: %s", match)
						continue
					}
					absMatch, _ := filepath.Abs(match)
					if !seen[absMatch] {
						files = append(files, match)
						seen[absMatch] = true
						logger.Printf("Added file: %s", match)
					}
				}
			}
		} else {
			// Not a glob or regex, treat as literal file path
			logger.Printf("Treating as literal path: '%s'", pattern)
			
			// Check if file exists
			if info, err := os.Stat(pattern); err == nil {
				if info.IsDir() {
					logger.Printf("Skipping directory: %s", pattern)
					continue
				}
				absPattern, _ := filepath.Abs(pattern)
				if !seen[absPattern] {
					files = append(files, pattern)
					seen[absPattern] = true
					logger.Printf("Added file: %s", pattern)
				}
			} else {
				// File doesn't exist, but don't error yet
				// It might be handled by resolveFilePath later
				logger.Printf("File not found (will try resolve later): %s", pattern)
				absPattern, _ := filepath.Abs(pattern)
				if !seen[absPattern] {
					files = append(files, pattern)
					seen[absPattern] = true
				}
			}
		}
	}
	
	logger.Printf("expandGlobs result: %d files", len(files))
	return files, nil
}

// findFilesWithRegex recursively searches for files matching regex pattern
func findFilesWithRegex(pattern string) ([]string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	
	matches := make([]string, 0)
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	
	gitignore, _ := loadGitIgnoreAndPtIgnore(cwd)
	
	err = filepath.Walk(cwd, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		
		// Skip ignored paths
		if gitignore != nil && gitignore.shouldIgnore(path, info.IsDir()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		
		if !info.IsDir() {
			relPath, _ := filepath.Rel(cwd, path)
			if re.MatchString(relPath) || re.MatchString(info.Name()) {
				matches = append(matches, path)
			}
		}
		
		return nil
	})
	
	return matches, err
}

// setWindowsHiddenAttribute sets the hidden attribute on a file or directory on Windows.
// It uses Windows-specific system calls.
// func setWindowsHiddenAttribute(path string) error {
// 	if runtime.GOOS != "windows" {
// 		// This function should only be called on Windows.
// 		return nil
// 	}

// 	// Convert the Go string path to a Windows UTF-16 string pointer (LPCWSTR)
// 	// This is required by the Windows API function.
// 	ptr, err := syscall.UTF16PtrFromString(path)
// 	if err != nil {
// 		return err
// 	}

// 	// Get current attributes
// 	attributes, err := windows.GetFileAttributes(ptr)
// 	if err != nil {
// 		return err
// 	}

// 	// Add the hidden attribute flag
// 	newAttributes := attributes | windows.FILE_ATTRIBUTE_HIDDEN

// 	// Set the new attributes
// 	return windows.SetFileAttributes(ptr, newAttributes)
// }

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

func searchFileRecursive(filename string, maxDepth int) ([]FileSearchResult, error) {
	results := make([]FileSearchResult, 0)
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	gitignore, err := loadGitIgnoreAndPtIgnore(cwd)
	if err != nil {
		logger.Printf("Warning: failed to load ignore patterns: %v", err)
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

        if gitignore != nil && gitignore.shouldIgnore(path, info.IsDir()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Check ignore patterns
		// if shouldIgnore(path, ignorePatterns) {
		// 	if info.IsDir() {
		// 		return filepath.SkipDir
		// 	}
		// 	return nil
		// }

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

// printShowHeader prints bat-like header
func printShowHeader(filePath string, info os.FileInfo, status FileStatus, showGrid bool) {
	relPath, _ := filepath.Rel(".", filePath)
	statusColor := status.Color()
	statusSymbol := "●"

	if showGrid {
		fmt.Printf("%s───────┬────────────────────────────────────────────────────────────────%s\n", ColorGray, ColorReset)
	}

	// File line with status indicator
	fmt.Printf("%s       │%s %sFile:%s %s ", ColorGray, ColorReset, ColorBold, ColorReset, relPath)
	if status != FileStatusUnchanged {
		fmt.Printf("%s%s %s%s", statusColor, statusSymbol, status.String(), ColorReset)
	}
	fmt.Println()

	// Size and modified time
	modTime := info.ModTime().Format("2006-01-02 15:04:05")
	fmt.Printf("%s       │%s %sSize:%s %s  %sModified:%s %s\n",
		ColorGray, ColorReset,
		ColorCyan, ColorReset, formatSize(info.Size()),
		ColorCyan, ColorReset, modTime)

	if showGrid {
		fmt.Printf("%s───────┼────────────────────────────────────────────────────────────────%s\n", ColorGray, ColorReset)
	}
}

// printShowFooter prints bat-like footer
func printShowFooter(filePath string, info os.FileInfo, contentLen int, showGrid bool) {
	if showGrid {
		fmt.Printf("%s───────┴────────────────────────────────────────────────────────────────%s\n", ColorGray, ColorReset)
	}
	fmt.Println()
}

// printWithLineNumbers prints content with line numbers
func printWithLineNumbers(content string, showGrid bool) {
	lines := strings.Split(content, "\n")
	maxLineNum := len(lines)
	lineNumWidth := len(fmt.Sprintf("%d", maxLineNum))

	for i, line := range lines {
		lineNum := i + 1
		if showGrid {
			fmt.Printf("%s%*d │%s %s\n", ColorGray, lineNumWidth, lineNum, ColorReset, line)
		} else {
			fmt.Printf("%s%*d %s %s\n", ColorGray, lineNumWidth, lineNum, ColorReset, line)
		}
	}
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

// ============================================================================
// HELP & VERSION
// ============================================================================

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

	fmt.Printf("\n%s👁️  VIEW & DISPLAY:%s\n", ColorBold+ColorYellow, ColorReset)
	fmt.Printf("  %spt show <filename>%s          Display file with syntax highlighting (like bat)\n", ColorGreen, ColorReset)
	fmt.Printf("  %spt show <file> -l <lexer>%s  Specify lexer (e.g., go, python, javascript)\n", ColorGreen, ColorReset)
	fmt.Printf("  %spt show <file> -t <theme>%s  Specify theme (default: monokai)\n", ColorGreen, ColorReset)
	fmt.Printf("  %spt show <file> --pager%s      Use pager (less) for navigation\n", ColorGreen, ColorReset)
	fmt.Printf("  %spt -z [options]%s             Show clipboard content\n", ColorGreen, ColorReset)
	fmt.Printf("    %s-l, --lexer <type>%s        Syntax highlighting (e.g., go, python)\n", ColorGreen, ColorReset)
	fmt.Printf("    %s-t, --theme <theme>%s       Color theme (default: monokai)\n", ColorGreen, ColorReset)
	fmt.Printf("    %s-np, --no-pager%s               Use pager mode (less)\n", ColorGreen, ColorReset)
	fmt.Printf("    %s--no-line-numbers%s         Disable line numbers\n", ColorGreen, ColorReset)
	fmt.Printf("    %s--no-grid%s                 Disable grid separators\n", ColorGreen, ColorReset)

	fmt.Printf("\n%s🎯 GIT-LIKE WORKFLOW:%s\n", ColorBold+ColorYellow, ColorReset)
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
	fmt.Printf("  %spt -d <filename> -z%s         Diff clipboard with file\n", ColorGreen, ColorReset)

	fmt.Printf("\n%s🌳 TREE & UTILITIES:%s\n", ColorBold+ColorYellow, ColorReset)
	fmt.Printf("  %spt -t [path]%s                Show directory tree\n", ColorGreen, ColorReset)
	fmt.Printf("  %spt -t [path] -e items,items%s       Tree with exceptions\n", ColorGreen, ColorReset)
	fmt.Printf("  %spt -rm <filename>%s           Safe delete (backup first)\n", ColorGreen, ColorReset)
	fmt.Printf("  %spt move <src> <dst>%s         Move file and adjust backups\n", ColorGreen, ColorReset)
	fmt.Printf("  %spt move <src...> <dst>%s      Move multiple files to directory\n", ColorGreen, ColorReset)
	fmt.Printf("  %spt mv <src...> <dst> -m%s     Move with comment\n", ColorGreen, ColorReset)
	fmt.Printf("  %spt move -r <dir> <dest>%s     Move directory recursively\n", ColorGreen, ColorReset)
	fmt.Printf("  %spt move \"*.py\" dest/%s        Move with wildcard\n", ColorGreen, ColorReset)
	fmt.Printf("  %spt move \"regex:test.*\" dest/%s Move with regex\n", ColorGreen, ColorReset)
	fmt.Printf("  %spt fix%s                      Detect & fix manual moves\n", ColorGreen, ColorReset)

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
	fmt.Printf("  %s$%s pt -d notes.txt -z          %s# Diff with temp file from clipboard (no backup)%s\n", ColorGray, ColorReset, ColorGray, ColorReset)
	fmt.Printf("  %s$%s pt show main.go              %s# View with syntax highlighting%s\n", ColorGray, ColorReset, ColorGray, ColorReset)
	fmt.Printf("  %s$%s pt show main.go -t dracula  %s# Use dracula theme%s\n", ColorGray, ColorReset, ColorGray, ColorReset)
	fmt.Printf("  %s$%s pt show main.go --pager      %s# Use less pager%s\n", ColorGray, ColorReset, ColorGray, ColorReset)
	fmt.Printf("  %s$%s pt -z -l python -p           %s# Show clipboard with pager%s\n", ColorGray, ColorReset, ColorGray, ColorReset)
	fmt.Printf("  %s$%s pt move file.txt docs/      %s# Move single file%s\n", ColorGray, ColorReset, ColorGray, ColorReset)
	fmt.Printf("  %s$%s pt move *.py src/           %s# Move multiple files%s\n", ColorGray, ColorReset, ColorGray, ColorReset)
	fmt.Printf("  %s$%s pt mv f1.go f2.rs backup/   %s# Move with backups%s\n", ColorGray, ColorReset, ColorGray, ColorReset)
	fmt.Printf("  %s$%s pt move -r subdir/ newdir/  %s# Move entire directory%s\n", ColorGray, ColorReset, ColorGray, ColorReset)
	fmt.Printf("  %s$%s pt move \"*.go\" backup/     %s# Wildcard move%s\n", ColorGray, ColorReset, ColorGray, ColorReset)
	fmt.Printf("  %s$%s pt move \"r:test_.*\" tmp/   %s# Regex move%s\n", ColorGray, ColorReset, ColorGray, ColorReset)
	fmt.Printf("  %s$%s pt fix                     %s# Fix manual moves%s\n", ColorGray, ColorReset, ColorGray, ColorReset)
	
	fmt.Printf("\n%s🎯 GIT-LIKE WORKFLOW:%s\n", ColorBold+ColorCyan, ColorReset)
	fmt.Printf("  1. %spt check%s                  - See what files changed (like git status)\n", ColorYellow, ColorReset)
	fmt.Printf("  2. %spt commit -m \"msg\"%s        - Backup all changes (like git commit)\n", ColorYellow, ColorReset)
	fmt.Printf("  3. %spt -l <file>%s              - View commit history\n", ColorYellow, ColorReset)
	fmt.Printf("  4. %spt -d <file> --last%s       - See what changed\n", ColorYellow, ColorReset)
	fmt.Printf("  5. %spt -r <file> --last%s       - Rollback if needed\n", ColorYellow, ColorReset)

	fmt.Printf("\n%s🎨 THEMES & LEXERS:%s\n", ColorBold+ColorCyan, ColorReset)
	fmt.Printf("  %sPopular Themes:%s monokai (default), dracula, solarized-dark, solarized-light,\n", ColorBold, ColorReset)
	fmt.Printf("                 github, vim, xcode, nord, gruvbox, one-dark\n")
	fmt.Printf("  %sPopular Lexers:%s go, python, javascript, typescript, rust, java, c, cpp,\n", ColorBold, ColorReset)
	fmt.Printf("                 bash, shell, json, yaml, xml, html, css, sql, markdown\n")
	fmt.Printf("  %sPager Controls:%s\n", ColorBold, ColorReset)
	fmt.Printf("    • Space      - Next page\n")
	fmt.Printf("    • b          - Previous page\n")
	fmt.Printf("    • /pattern   - Search forward\n")
	fmt.Printf("    • q          - Quit\n")
	fmt.Printf("    • h          - Help (in less)\n")
	
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

// ============================================================================
// MAIN
// ============================================================================

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

    for i := 1; i < len(os.Args); i++ {
	    if os.Args[i] == "--tool" && i+1 < len(os.Args) {
	        difftool = os.Args[i+1]
	        break
	    }
	}

	for i := 1; i < len(os.Args); i++ {
	    if os.Args[i] == "-z" && i+1 < len(os.Args) {
	        foundZ = true
	        break
	    }
	}

    // Setup logger based on the parsed debug flag
    setupLogger()

	switch os.Args[1] {
		case "show":
			if len(os.Args) < 3 {
				fmt.Printf("%s❌ Error: Filename required for show command%s\n", ColorRed, ColorReset)
				fmt.Println("\nUsage:")
				fmt.Println("  pt show <filename>")
				fmt.Println("  pt show <filename> --lexer <type> --theme <theme>")
				fmt.Println("  pt show <filename> --pager")
				fmt.Println("\nExamples:")
				fmt.Println("  pt show main.go")
				fmt.Println("  pt show main.go --lexer go --theme dracula")
				fmt.Println("  pt show script.py --theme monokai --pager")
				os.Exit(1)
			}

			err := handleShowCommand(os.Args[2:])
			if err != nil {
				fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
				os.Exit(1)
			}

		case "move", "mv", "-mv":
			if len(os.Args) < 4 {
				fmt.Printf("%s❌ Error: At least source and destination required%s\n", ColorRed, ColorReset)
				fmt.Println("\nUsage:")
				fmt.Println("  pt move <source> <destination>")
				fmt.Println("  pt move <source1> <source2> ... <destination>")
				fmt.Println("  pt mv <source...> <destination> -m \"comment\"")
				fmt.Println("\nExamples:")
				fmt.Println("  pt move file.txt newdir/")
				fmt.Println("  pt move file1.py file2.go file3.rs dest/")
				fmt.Println("  pt mv old.py new/location/renamed.py -m \"reorganize\"")
				fmt.Println("  pt mv *.txt backup/ -m \"archive text files\"")
				os.Exit(1)
			}

			err := handleMoveCommand(os.Args[2:])
			if err != nil {
				fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
				os.Exit(1)
			}
		
		case "fix":
			err := handleFixCommand(os.Args[2:])
			if err != nil {
				fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
				os.Exit(1)
			}

		case "-z": 
			err := handleTempCommand(os.Args[2:]) // Pass remaining args (like --lexer)
			if err != nil {
				fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
				os.Exit(1)
			}

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
			if len(os.Args) < 3 { // Minimal arg: pt -d <file_name>
				fmt.Printf("%s❌ Error: Filename required%s\n", ColorRed, ColorReset)
				os.Exit(1)
			}

			// Check for the specific combination: pt -d <file_name> -z
			// We look for -z in os.Args[3] or later, after the file name at os.Args[2]
			// for _, arg := range os.Args[3:] { // Start checking from the 4th argument (index 3)
			// 	if arg == "-z" {
			// 		foundZ = true
			// 		break
			// 	}
			// }


			if foundZ {
				// If -z is found, treat os.Args[2] as the file name and use new logic
				fileName := os.Args[2] // Get the file name argument
				// Call the new function
				err := handleDiffClipboardToFile(fileName)
				if err != nil {
					fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
					os.Exit(1)
				}
				return // Exit after handling the -d <file_name> -z case
			} else {
				// If -z is not found, proceed with the original handleDiffCommand logic
				// Pass all arguments starting from the file name (os.Args[2:])
				err := handleDiffCommand(os.Args[2:]) // This expects [filename, optional --last]
				if err != nil {
					fmt.Printf("%s❌ Error: %v%s\n", ColorRed, err, ColorReset)
					os.Exit(1)
				}
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


	}
}
