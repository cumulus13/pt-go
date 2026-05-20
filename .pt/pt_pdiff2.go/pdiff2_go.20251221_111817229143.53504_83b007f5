package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// ANSI color codes
const (
	Reset      = "\033[0m"
	Bold       = "\033[1m"
	Italic     = "\033[3m"
	Red        = "\033[31m"
	Green      = "\033[32m"
	Yellow     = "\033[33m"
	Blue       = "\033[34m"
	Grey       = "\033[38;5;249m"
	BoldRed    = "\033[1;31m"
	BoldGreen  = "\033[1;32m"
	BoldYellow = "\033[1;33m"
	BrightGreen = "\033[1;38;2;0;203;0m"
	WhiteOnBlue = "\033[37;44m"
)

type Hunk struct {
	SourceStart int
	SourceLen   int
	TargetStart int
	TargetLen   int
	Section     string
	Lines       []string
}

type FileDiff struct {
	Old   string
	New   string
	Hunks []Hunk
}

type PDiff2 struct{}

func (p *PDiff2) DiffFiles(file1, file2 any) (string, error) {
	// Helper function to get content from file or data
	getContent := func(input any) (string, error) {
		switch v := input.(type) {
		case string:
			// Check if it's a file path
			if _, err := os.Stat(v); err == nil {
				data, err := os.ReadFile(v)
				if err != nil {
					return "", fmt.Errorf("failed to read file %s: %v", v, err)
				}
				return string(data), nil
			}
			// Assume it's raw content/data
			return v, nil
		case []byte:
			return string(v), nil
		default:
			return "", fmt.Errorf("unsupported type: %T", v)
		}
	}
	
	content1, err := getContent(file1)
	if err != nil {
		return "", err
	}
	
	content2, err := getContent(file2)
	if err != nil {
		return "", err
	}
	
	// Create temporary files for diff comparison
	tmpFile1, err := os.CreateTemp("", "pdiff1-*.tmp")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile1.Name())
	defer tmpFile1.Close()
	
	tmpFile2, err := os.CreateTemp("", "pdiff2-*.tmp")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile2.Name())
	defer tmpFile2.Close()
	
	// Write contents to temp files
	if _, err := tmpFile1.WriteString(content1); err != nil {
		return "", fmt.Errorf("failed to write to temp file: %v", err)
	}
	if _, err := tmpFile2.WriteString(content2); err != nil {
		return "", fmt.Errorf("failed to write to temp file: %v", err)
	}
	
	tmpFile1.Close()
	tmpFile2.Close()
	
	// Run git diff on the temp files
	cmd := exec.Command("git", "diff", "--no-index", "-U0", "-p", tmpFile1.Name(), tmpFile2.Name())
	output, _ := cmd.CombinedOutput() // git diff returns exit code 1 when there are differences
	
	return string(output), nil
}

// func (p *PDiff2) GetGitDiff(cached bool) (string, error) {
// 	args := []string{"diff", "-U0", "-p"}
// 	if cached {
// 		args = append(args, "--cached")
// 	}
	
// 	cmd := exec.Command("git", args...)
// 	output, err := cmd.CombinedOutput()
// 	if err != nil {
// 		return "", fmt.Errorf("error running git diff: %v", err)
// 	}
	
// 	return string(output), nil
// }

func (p *PDiff2) GetGitDiff(cached bool, filePath ...string) (string, error) {
	args := []string{"diff", "-U0", "-p"}
	if cached {
		args = append(args, "--cached")
	}
	
	// Tambahkan file path jika ada
	if len(filePath) > 0 {
		args = append(args, "--")
		args = append(args, filePath...)
	}
	
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error running git diff: %v", err)
	}
	
	return string(output), nil
}

func (p *PDiff2) ParseDiff(diffText string) []FileDiff {
	files := []FileDiff{}
	var currentFile *FileDiff
	var hunk *Hunk
	
	scanner := bufio.NewScanner(strings.NewReader(diffText))
	hunkRegex := regexp.MustCompile(`@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@(.*)`)
	
	for scanner.Scan() {
		line := scanner.Text()
		
		if strings.HasPrefix(line, "--- ") {
			oldFile := strings.TrimSpace(line[4:])
			currentFile = &FileDiff{Old: oldFile, New: "", Hunks: []Hunk{}}
			files = append(files, *currentFile)
			currentFile = &files[len(files)-1]
		} else if strings.HasPrefix(line, "+++ ") {
			newFile := strings.TrimSpace(line[4:])
			currentFile.New = newFile
		} else if strings.HasPrefix(line, "@@") {
			matches := hunkRegex.FindStringSubmatch(line)
			if len(matches) > 0 {
				sourceStart, _ := strconv.Atoi(matches[1])
				sourceLen := 1
				if matches[2] != "" {
					sourceLen, _ = strconv.Atoi(matches[2])
				}
				targetStart, _ := strconv.Atoi(matches[3])
				targetLen := 1
				if matches[4] != "" {
					targetLen, _ = strconv.Atoi(matches[4])
				}
				section := strings.TrimSpace(matches[5])
				
				hunk = &Hunk{
					SourceStart: sourceStart,
					SourceLen:   sourceLen,
					TargetStart: targetStart,
					TargetLen:   targetLen,
					Section:     section,
					Lines:       []string{},
				}
				currentFile.Hunks = append(currentFile.Hunks, *hunk)
				hunk = &currentFile.Hunks[len(currentFile.Hunks)-1]
			}
		} else if hunk != nil {
			hunk.Lines = append(hunk.Lines, line)
		}
	}
	
	return files
}

func (p *PDiff2) PrintDiff(diffText string) {
	files := p.ParseDiff(diffText)
	
	if len(files) == 0 {
		fmt.Printf("%s%sNo changes found.%s\n", Bold, Yellow, Reset)
		return
	}
	
	for _, f := range files {
		oldFile := f.Old
		newFile := f.New
		
		if oldFile == "/dev/null" {
			fmt.Printf("     🆕 ++ %s%s%s%s\n", Bold, Green, newFile, Reset)
		} else if newFile == "/dev/null" {
			fmt.Printf("  🗑️  -- %s%s%s%s\n", Bold, Red, oldFile, Reset)
		} else {
			fmt.Printf("  📝 %s%s%s%s -> %s%s\n", Bold, Yellow, Italic, oldFile, newFile, Reset)
		}
		
		for _, h := range f.Hunks {
			fmt.Printf("     📌 %d,%d -> %d,%d %s%s%s %s %s\n",
				h.SourceStart, h.SourceLen, h.TargetStart, h.TargetLen,
				WhiteOnBlue, Italic, h.Section, Reset, Reset)
			
			added := 0
			removed := 0
			
			for _, line := range h.Lines {
				var icon, color, symbol string
				
				if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
					icon = "🟢"
					color = BrightGreen
					symbol = "+"
					added++
				} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
					icon = "🔴"
					color = BoldRed
					symbol = "-"
					removed++
				} else {
					icon = "⚪"
					color = Grey
					symbol = " "
				}
				
				fmt.Printf("     %s %s%s %s%s\n", icon, color, symbol, strings.TrimRight(line, "\n\r"), Reset)
			}
			
			fmt.Printf("     %s+%d%s %s-%d%s\n\n", BoldGreen, added, Reset, BoldRed, removed, Reset)
		}
	}
}

func (p *PDiff2) Main() {
	// Check if it's a git repository (skip check if comparing files directly)
	argsLen := len(os.Args)
	
	var diffText string
	var err error
	
	if argsLen == 3 {
		// Mode: compare two files directly
		// pdiff2 file1 file2
		diffText, err = p.DiffFiles(os.Args[1], os.Args[2])
		if err != nil {
			fmt.Printf("%sError comparing files: %s%s\n", Red, err, Reset)
			os.Exit(1)
		}
	} else if argsLen > 1 {
		// Mode: read multiple diff files and combine them
		// pdiff2 diff1.diff diff2.diff ...
		if _, err := os.Stat(".git"); os.IsNotExist(err) {
			fmt.Printf("%sNot a Git repository.%s\n", Red, Reset)
			os.Exit(1)
		}
		
		var allDiffs strings.Builder
		for _, diffPath := range os.Args[1:] {
			data, err := os.ReadFile(diffPath)
			if err != nil {
				fmt.Printf("%sFile not found: %s%s\n", Red, diffPath, Reset)
				continue
			}
			allDiffs.WriteString(string(data))
			allDiffs.WriteString("\n")
		}
		diffText = allDiffs.String()
	} else {
		// Mode: default git diff
		if _, err := os.Stat(".git"); os.IsNotExist(err) {
			fmt.Printf("%sNot a Git repository.%s\n", Red, Reset)
			os.Exit(1)
		}
		
		diffText, err = p.GetGitDiff(false)
		if err != nil {
			fmt.Printf("%s%s%s\n", Red, err, Reset)
			os.Exit(1)
		}
	}
	
	p.PrintDiff(diffText)
}

func run_main() {
	pdiff := &PDiff2{}
	pdiff.Main()
}