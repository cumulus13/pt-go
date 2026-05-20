// commit_log.go  –  drop into pt/ alongside main.go
//
// Adds a commit index to pt so that "pt commit -m" becomes fully reversible
// as a GROUP, equivalent to "git reset --hard <commit>".
//
// ─────────────────────────────────────────────────────────────────────────────
// NEW COMMANDS
// ─────────────────────────────────────────────────────────────────────────────
//   pt log                      list all commits (newest first)
//   pt log <id>                 show every file in one commit
//   pt reset --last             restore the most recent commit (all its files)
//   pt reset --last -n 2        restore the Nth most recent commit
//   pt reset <id>               restore a specific commit by short/full ID
//   pt reset <id> --dry-run     show what would be restored without touching disk
//
// ─────────────────────────────────────────────────────────────────────────────
// STORAGE  (.pt/PT_COMMITS)
// ─────────────────────────────────────────────────────────────────────────────
// One JSON line per commit, newest appended last.
// Example line (pretty-printed for readability):
//
//   {
//     "id":        "a3f9c1d2",           // 8-char hex, unique per commit
//     "message":   "fix bugs",           // raw user message (no "commit:" prefix)
//     "timestamp": "2026-05-20T22:34:28.123456Z",
//     "files": [
//       {
//         "original": "/abs/path/main.go",
//         "backup":   "/abs/path/.pt/main.go/main_go.20260520_223428123456.1234_abcd",
//         "status":   "modified"          // "modified" | "new"
//       }
//     ]
//   }
//
// The file is append-only.  Each reset does NOT delete or alter past entries;
// it appends a new commit whose message is "reset: restored from <id>".
//
// ─────────────────────────────────────────────────────────────────────────────
// INTEGRATION – changes required in main.go
// ─────────────────────────────────────────────────────────────────────────────
//   1. DELETE handleCommitCommand from main.go  (replaced here)
//   2. In the main() switch add:
//        case "log":
//            err = handleLogCommand(info.Files)
//        case "reset":
//            err = handleResetCommand(info.Files, info.BoolFlags["--dry-run"])
//   3. In parseArguments(), add to the commands map:
//        "log": true, "reset": true,
//   4. In printHelp(), add the two new commands to the help text.

package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// Data structures
// ─────────────────────────────────────────────────────────────────────────────

const ptCommitsFile = "PT_COMMITS"

// CommitFileEntry records one file that was backed up as part of a commit.
type CommitFileEntry struct {
	Original string `json:"original"` // absolute path of the working file
	Backup   string `json:"backup"`   // absolute path of the backup file in .pt
	Status   string `json:"status"`   // "modified" | "new"
}

// CommitRecord is one line in PT_COMMITS.
type CommitRecord struct {
	ID        string            `json:"id"`
	Message   string            `json:"message"`
	Timestamp time.Time         `json:"timestamp"`
	Files     []CommitFileEntry `json:"files"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Commit log path helpers
// ─────────────────────────────────────────────────────────────────────────────

// commitsFilePath returns the absolute path of PT_COMMITS for a given .pt root.
func commitsFilePath(ptRoot string) string {
	return filepath.Join(ptRoot, ptCommitsFile)
}

// generateCommitID returns an 8-character random hex string.
func generateCommitID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// appendCommitRecord appends one CommitRecord as a JSON line to PT_COMMITS.
// The file is created if it does not exist.
func appendCommitRecord(ptRoot string, rec CommitRecord) error {
	path := commitsFilePath(ptRoot)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open PT_COMMITS: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("marshal commit record: %w", err)
	}
	_, err = fmt.Fprintf(f, "%s\n", data)
	return err
}

// readAllCommits reads every CommitRecord from PT_COMMITS, newest last.
// Returns an empty slice (not an error) when the file does not exist yet.
func readAllCommits(ptRoot string) ([]CommitRecord, error) {
	path := commitsFilePath(ptRoot)
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open PT_COMMITS: %w", err)
	}
	defer f.Close()

	var records []CommitRecord
	scanner := bufio.NewScanner(f)
	// PT_COMMITS lines can be long (many files); increase buffer to 4 MB.
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var rec CommitRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			logger.Printf("PT_COMMITS line %d parse error (skipped): %v", lineNum, err)
			continue
		}
		records = append(records, rec)
	}
	return records, scanner.Err()
}

// findCommitByID returns the first CommitRecord whose ID starts with prefix.
// Search order is newest-first (we reverse the slice before searching).
func findCommitByID(records []CommitRecord, prefix string) (CommitRecord, bool) {
	prefix = strings.ToLower(prefix)
	for i := len(records) - 1; i >= 0; i-- {
		if strings.HasPrefix(strings.ToLower(records[i].ID), prefix) {
			return records[i], true
		}
	}
	return CommitRecord{}, false
}

// resolveProjectPTRoot finds the .pt root for the current project.
// Returns ("", error) if no .pt directory can be found or created.
func resolveProjectPTRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	raw, err := findPTRoot(cwd)
	if err != nil {
		return "", err
	}
	if raw == "" {
		return "", fmt.Errorf("no %s directory found; run 'pt commit' first to create one",
			appConfig.BackupDirName)
	}
	// normPTRoot is defined in smart_move.go.
	// If smart_move.go is not in the package, inline the logic:
	if filepath.Base(raw) == appConfig.BackupDirName {
		if info, e := os.Stat(raw); e == nil && info.IsDir() {
			return raw, nil
		}
	}
	candidate := filepath.Join(raw, appConfig.BackupDirName)
	if info, e := os.Stat(candidate); e == nil && info.IsDir() {
		return candidate, nil
	}
	return "", fmt.Errorf("could not locate %s directory", appConfig.BackupDirName)
}

// ─────────────────────────────────────────────────────────────────────────────
// handleCommitCommand  (replaces the version in main.go – DELETE that one)
// ─────────────────────────────────────────────────────────────────────────────
func handleCommitCommand(args []string) error {
	// Parse -m / --message
	commitMessage := ""
	for i := 0; i < len(args); i++ {
		if (args[i] == "-m" || args[i] == "--message") && i+1 < len(args) {
			commitMessage = args[i+1]
			break
		}
	}
	if commitMessage == "" {
		return fmt.Errorf("commit message required: pt commit -m \"your message\"")
	}

	fmt.Printf("\n%s📦 Committing changes…%s\n\n", ColorBold+ColorCyan, ColorReset)

	// Locate project root (same logic as original)
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}
	projectRoot := cwd
	ptRoot, err := findPTRoot(cwd)
	if err == nil && ptRoot != "" {
		if filepath.Base(ptRoot) == appConfig.BackupDirName {
			projectRoot = filepath.Dir(ptRoot)
		} else {
			projectRoot = ptRoot
		}
	} else {
		if gr := findGitRoot(cwd); gr != "" {
			projectRoot = gr
		}
	}

	relRoot, _ := filepath.Rel(cwd, projectRoot)
	if relRoot != "" && relRoot != "." {
		fmt.Printf("%sCommitting from project root:%s %s\n\n", ColorGray, ColorReset, projectRoot)
	}

	// Load ignores
	gitignore, _ := loadGitIgnoreAndPtIgnore(projectRoot)
	exceptions := map[string]bool{appConfig.BackupDirName: true}

	// Build status tree
	tree, err := buildStatusTree(projectRoot, gitignore, exceptions, 0, appConfig.MaxSearchDepth)
	if err != nil {
		return fmt.Errorf("build status tree: %w", err)
	}
	if tree == nil {
		return fmt.Errorf("no files found")
	}

	// Collect changed files
	var changedFiles []string
	collectChangedFiles(tree, &changedFiles)
	if len(changedFiles) == 0 {
		fmt.Printf("%s✓ Nothing to commit. All files match their last backups.%s\n",
			ColorGreen, ColorReset)
		return nil
	}

	// Show what will be committed
	fmt.Println("Files to commit:")
	for i, file := range changedFiles {
		relPath, _ := filepath.Rel(projectRoot, file)
		status, _ := compareFileWithBackup(file)
		statusColor := status.Color()
		fmt.Printf("  %d. %s%s%s %s[%s]%s\n",
			i+1, ColorGreen, relPath, ColorReset,
			statusColor, status.String(), ColorReset)
	}
	fmt.Println()

	// Confirm
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Commit %d file(s) with message %q? (y/N): ",
		len(changedFiles), commitMessage)
	input, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(input)) != "y" {
		fmt.Println("❌ Commit cancelled")
		return nil
	}

	// Ensure .pt exists so we can write PT_COMMITS
	activePTRoot, err := ensurePTDir(projectRoot + string(os.PathSeparator) + "dummy")
	if err != nil {
		return fmt.Errorf("ensurePTDir: %w", err)
	}

	// Build the commit record while backing up files
	backupComment := "commit: " + commitMessage
	commitID := generateCommitID()
	rec := CommitRecord{
		ID:        commitID,
		Message:   commitMessage,
		Timestamp: time.Now().UTC(),
	}

	successCount := 0
	failCount := 0

	for _, file := range changedFiles {
		relPath, _ := filepath.Rel(projectRoot, file)
		status, _ := compareFileWithBackup(file)

		// Back up the file (same call as original)
		_, err := autoRenameIfExists(file, backupComment, false)
		if err != nil {
			fmt.Printf("%s✗%s %s: %v\n", ColorRed, ColorReset, relPath, err)
			failCount++
			continue
		}

		// Find the backup that was just created (newest backup for this file)
		backupPath := ""
		if backups, berr := listBackups(file); berr == nil && len(backups) > 0 {
			backupPath = backups[0].Path
		}

		rec.Files = append(rec.Files, CommitFileEntry{
			Original: file,
			Backup:   backupPath,
			Status:   status.String(),
		})

		fmt.Printf("%s✓%s %s\n", ColorGreen, ColorReset, relPath)
		successCount++
	}

	// Write commit record to PT_COMMITS
	if successCount > 0 {
		if werr := appendCommitRecord(activePTRoot, rec); werr != nil {
			fmt.Printf("%s⚠️  Commit index write failed: %v%s\n", ColorYellow, werr, ColorReset)
			fmt.Printf("    Files are backed up but this commit won't appear in 'pt log'.\n")
		} else {
			fmt.Printf("\n%s💾 Commit recorded:%s %s[%s]%s\n",
				ColorBold, ColorReset, ColorCyan, commitID, ColorReset)
		}
	}

	// Summary
	fmt.Println()
	fmt.Printf("%s📦 Commit summary:%s\n", ColorBold, ColorReset)
	fmt.Printf("  %s✓ %d file(s) backed up%s\n", ColorGreen, successCount, ColorReset)
	if failCount > 0 {
		fmt.Printf("  %s✗ %d file(s) failed%s\n", ColorRed, failCount, ColorReset)
	}
	fmt.Printf("  💬 Message: %q\n", commitMessage)
	fmt.Printf("  🔑 ID: %s\n", commitID)
	fmt.Printf("\n%sUse 'pt log' to see history, 'pt reset %s' to undo.%s\n",
		ColorGray, commitID, ColorReset)

	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// pt log
// ─────────────────────────────────────────────────────────────────────────────

func handleLogCommand(args []string) error {
	// Optional: a commit ID to show detail
	detailID := ""
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			detailID = a
			break
		}
	}

	ptRoot, err := resolveProjectPTRoot()
	if err != nil {
		return err
	}

	records, err := readAllCommits(ptRoot)
	if err != nil {
		return fmt.Errorf("read commit log: %w", err)
	}
	if len(records) == 0 {
		fmt.Printf("%sNo commits found. Use 'pt commit -m \"message\"' to create one.%s\n",
			ColorGray, ColorReset)
		return nil
	}

	// Newest first
	reversed := make([]CommitRecord, len(records))
	for i, r := range records {
		reversed[len(records)-1-i] = r
	}

	if detailID != "" {
		rec, found := findCommitByID(records, detailID)
		if !found {
			return fmt.Errorf("commit %q not found", detailID)
		}
		printCommitDetail(rec, ptRoot)
		return nil
	}

	printCommitLog(reversed, ptRoot)
	return nil
}

func printCommitLog(records []CommitRecord, ptRoot string) {
	cwd, _ := os.Getwd()
	width := getTerminalWidth()
	sep := strings.Repeat("─", width)

	fmt.Printf("\n%s%s📋 PT Commit Log%s\n", ColorBold+ColorCyan, "", ColorReset)
	fmt.Printf("%s%s%s\n\n", ColorGray, sep, ColorReset)

	for i, rec := range records {
		// Mark resets differently
		icon := "●"
		msgColor := ColorYellow
		if strings.HasPrefix(rec.Message, "reset:") {
			icon = "↩"
			msgColor = ColorMagenta
		}

		age := formatAge(time.Since(rec.Timestamp))
		fmt.Printf("%s%s %s[%s]%s %s%s%s\n",
			msgColor, icon, ColorReset,
			ColorCyan+rec.ID+ColorReset,
			ColorReset,
			msgColor, rec.Message, ColorReset)
		fmt.Printf("    %s%s  %s files  %s%s\n",
			ColorGray, rec.Timestamp.Local().Format("2006-01-02 15:04:05"),
			pluralN(len(rec.Files), "file"),
			age, ColorReset)

		// Show files (collapsed)
		for _, fe := range rec.Files {
			rel, err := filepath.Rel(cwd, fe.Original)
			if err != nil {
				rel = fe.Original
			}
			statusColor := ColorGreen
			statusMark := "M"
			if fe.Status == "new" {
				statusColor = ColorCyan
				statusMark = "A"
			}
			backupOK := "✓"
			if fe.Backup == "" {
				backupOK = "?"
			} else if _, serr := os.Stat(fe.Backup); serr != nil {
				backupOK = "✗" // backup file missing
			}
			fmt.Printf("    %s%s%s %s%s  %s[backup %s]%s\n",
				statusColor, statusMark, ColorReset,
				rel,
				ColorReset,
				ColorGray, backupOK, ColorReset)
		}

		if i < len(records)-1 {
			fmt.Println()
		}
	}

	fmt.Printf("\n%s%s%s\n", ColorGray, sep, ColorReset)
	fmt.Printf("%sTotal: %d commit(s). Use 'pt log <id>' for detail, 'pt reset <id>' to restore.%s\n\n",
		ColorGray, len(records), ColorReset)
}

func printCommitDetail(rec CommitRecord, ptRoot string) {
	cwd, _ := os.Getwd()
	width := getTerminalWidth()
	sep := strings.Repeat("─", width)

	fmt.Printf("\n%s%s Commit [%s]%s\n", ColorBold+ColorCyan, "●", rec.ID, ColorReset)
	fmt.Printf("%s%s%s\n", ColorGray, sep, ColorReset)
	fmt.Printf("  %sMessage:%s   %s\n", ColorBold, ColorReset, rec.Message)
	fmt.Printf("  %sTimestamp:%s %s  (%s ago)\n",
		ColorBold, ColorReset,
		rec.Timestamp.Local().Format("2006-01-02 15:04:05"),
		formatAge(time.Since(rec.Timestamp)))
	fmt.Printf("  %sFiles:%s     %s\n\n",
		ColorBold, ColorReset, pluralN(len(rec.Files), "file"))

	for _, fe := range rec.Files {
		rel, err := filepath.Rel(cwd, fe.Original)
		if err != nil {
			rel = fe.Original
		}
		statusColor := ColorGreen
		if fe.Status == "new" {
			statusColor = ColorCyan
		}

		fmt.Printf("  %s%s%s  %s\n", statusColor, strings.ToUpper(fe.Status), ColorReset, rel)
		fmt.Printf("    %sOriginal:%s %s\n", ColorGray, ColorReset, fe.Original)

		if fe.Backup == "" {
			fmt.Printf("    %sBackup:  (not recorded)%s\n", ColorRed, ColorReset)
		} else {
			brel, _ := filepath.Rel(cwd, fe.Backup)
			_, serr := os.Stat(fe.Backup)
			backupStatus := ColorGreen + "exists" + ColorReset
			if serr != nil {
				backupStatus = ColorRed + "MISSING" + ColorReset
			}
			fmt.Printf("    %sBackup:%s  %s  [%s]\n",
				ColorGray, ColorReset, brel, backupStatus)
		}
		fmt.Println()
	}

	fmt.Printf("%s%s%s\n", ColorGray, sep, ColorReset)
	fmt.Printf("%sUse 'pt reset %s' to restore all files to this state.%s\n\n",
		ColorGray, rec.ID, ColorReset)
}

// ─────────────────────────────────────────────────────────────────────────────
// pt reset
// ─────────────────────────────────────────────────────────────────────────────

func handleResetCommand(args []string, dryRun bool) error {
	useLast := false
	nthLast := 1
	commitIDArg := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--last", "-lt":
			useLast = true
		case "-n":
			if i+1 < len(args) {
				i++
				n, err := strconv.Atoi(args[i])
				if err != nil || n < 1 {
					return fmt.Errorf("-n requires a positive integer")
				}
				nthLast = n
				useLast = true
			}
		case "--dry-run":
			dryRun = true
		default:
			if !strings.HasPrefix(args[i], "-") {
				commitIDArg = args[i]
			}
		}
	}

	if !useLast && commitIDArg == "" {
		return fmt.Errorf("usage:\n" +
			"  pt reset --last              restore most recent commit\n" +
			"  pt reset --last -n 2         restore 2nd most recent commit\n" +
			"  pt reset <id>                restore a specific commit\n" +
			"  pt reset <id> --dry-run      preview without touching disk")
	}

	ptRoot, err := resolveProjectPTRoot()
	if err != nil {
		return err
	}

	records, err := readAllCommits(ptRoot)
	if err != nil {
		return fmt.Errorf("read commit log: %w", err)
	}
	if len(records) == 0 {
		return fmt.Errorf("no commits found in %s", commitsFilePath(ptRoot))
	}

	// Filter out "reset:" entries when selecting by --last so we always pick
	// a real commit, not a previous reset operation.
	var realCommits []CommitRecord
	for _, r := range records {
		if !strings.HasPrefix(r.Message, "reset:") {
			realCommits = append(realCommits, r)
		}
	}

	var target CommitRecord
	var found bool

	if useLast {
		idx := len(realCommits) - nthLast
		if idx < 0 {
			return fmt.Errorf("only %d real commit(s) exist, cannot go back %d",
				len(realCommits), nthLast)
		}
		target = realCommits[idx]
		found = true
	} else {
		target, found = findCommitByID(records, commitIDArg)
		if !found {
			return fmt.Errorf("commit %q not found\nUse 'pt log' to see available commits",
				commitIDArg)
		}
	}

	return executeReset(target, ptRoot, dryRun)
}

func executeReset(rec CommitRecord, ptRoot string, dryRun bool) error {
	cwd, _ := os.Getwd()
	width := getTerminalWidth()
	sep := strings.Repeat("─", width)

	dryTag := ""
	if dryRun {
		dryTag = " " + ColorYellow + "[DRY RUN]" + ColorReset
	}

	fmt.Printf("\n%s↩ Resetting to commit [%s]%s%s\n",
		ColorBold+ColorMagenta, rec.ID, ColorReset, dryTag)
	fmt.Printf("  %s%q  %s  %s%s\n",
		ColorYellow, rec.Message, ColorReset,
		rec.Timestamp.Local().Format("2006-01-02 15:04:05"),
		ColorReset)
	fmt.Printf("  %d file(s) to restore\n", len(rec.Files))
	fmt.Printf("%s%s%s\n\n", ColorGray, sep, ColorReset)

	// Validate all backup files exist BEFORE touching anything
	missing := 0
	for _, fe := range rec.Files {
		if fe.Backup == "" {
			fmt.Printf("  %s⚠️  No backup recorded for %s%s\n",
				ColorYellow, fe.Original, ColorReset)
			missing++
			continue
		}
		if _, err := os.Stat(fe.Backup); err != nil {
			rel, _ := filepath.Rel(cwd, fe.Backup)
			fmt.Printf("  %s❌ Backup missing: %s%s\n", ColorRed, rel, ColorReset)
			missing++
		}
	}
	if missing > 0 {
		fmt.Printf("\n%s❌ %d backup file(s) are missing. Reset aborted – nothing was changed.%s\n",
			ColorRed, missing, ColorReset)
		return fmt.Errorf("%d backup file(s) missing, reset aborted", missing)
	}

	if dryRun {
		fmt.Printf("%s── Dry run – files that WOULD be restored: ──%s\n", ColorGray, ColorReset)
		for _, fe := range rec.Files {
			rel, _ := filepath.Rel(cwd, fe.Original)
			brel, _ := filepath.Rel(cwd, fe.Backup)
			fmt.Printf("  %s→%s %-45s  %sfrom: %s%s\n",
				ColorGreen, ColorReset, rel, ColorGray, brel, ColorReset)
		}
		fmt.Printf("\n%s(no files were changed)%s\n\n", ColorGray, ColorReset)
		return nil
	}

	// Confirm
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s⚠️  This will restore %d file(s) to their state at commit [%s].%s\n",
		ColorYellow, len(rec.Files), rec.ID, ColorReset)
	fmt.Printf("  Current working-tree versions will be backed up first.\n")
	fmt.Print("  Continue? (y/N): ")
	ans, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(ans)) != "y" {
		fmt.Println("❌ Reset cancelled")
		return nil
	}

	// Execute restore – one file at a time
	// restoreBackup (from main.go) backs up the current file before restoring.
	successCount := 0
	failCount := 0
	resetFiles := []CommitFileEntry{}

	for _, fe := range rec.Files {
		rel, _ := filepath.Rel(cwd, fe.Original)
		fmt.Printf("  %s→%s %s\n", ColorGreen, ColorReset, rel)

		restoreComment := fmt.Sprintf("pre-reset backup (restoring commit %s)", rec.ID)
		err := restoreBackup(fe.Backup, fe.Original, restoreComment)
		if err != nil {
			fmt.Printf("    %s❌ Failed: %v%s\n", ColorRed, err, ColorReset)
			failCount++
			continue
		}
		successCount++

		// Record what was actually restored so we can log it
		resetFiles = append(resetFiles, fe)
	}

	// Append a "reset" entry to PT_COMMITS so the log shows the operation
	if successCount > 0 {
		resetRec := CommitRecord{
			ID:        generateCommitID(),
			Message:   fmt.Sprintf("reset: restored from commit %s (%q)", rec.ID, rec.Message),
			Timestamp: time.Now().UTC(),
			Files:     resetFiles,
		}
		if werr := appendCommitRecord(ptRoot, resetRec); werr != nil {
			logger.Printf("warning: could not append reset record to PT_COMMITS: %v", werr)
		}
	}

	// Summary
	fmt.Println()
	fmt.Printf("%s📊 Reset summary:%s\n", ColorBold, ColorReset)
	fmt.Printf("  %s✓ %d file(s) restored%s\n", ColorGreen, successCount, ColorReset)
	if failCount > 0 {
		fmt.Printf("  %s✗ %d file(s) failed%s\n", ColorRed, failCount, ColorReset)
	}
	fmt.Printf("  🔑 Restored from commit: %s\n", rec.ID)
	fmt.Printf("  💬 %q\n", rec.Message)
	if failCount > 0 {
		return fmt.Errorf("%d file(s) failed to restore", failCount)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Formatting helpers
// ─────────────────────────────────────────────────────────────────────────────

func formatAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return fmt.Sprintf("%dw ago", int(d.Hours()/(24*7)))
	}
}

func pluralN(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}
