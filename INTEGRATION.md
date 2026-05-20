# Integration instructions for commit_log.go
# ─────────────────────────────────────────────────────────────────────────────
# 1. DELETE handleCommitCommand from main.go  (commit_log.go replaces it)
#    Search for:   func handleCommitCommand(args []string) error {
#    Delete everything from that line to its matching closing brace.
#
# 2. In parseArguments() – add to the commands map:
#    (find the block:  commands := map[string]bool{ ... })

        "log":   true,
        "reset": true,

#
# 3. In the main() switch – add two new cases:
#    (find:  case "commit":)
#    Add after it:

        case "log":
            err = handleLogCommand(info.Files)

        case "reset":
            dryRun := info.BoolFlags["--dry-run"]
            err = handleResetCommand(info.Files, dryRun)

#
# 4. In parseArguments() – add "--dry-run" to the boolFlags map:
#    (find the block:  boolFlags := map[string]bool{ ... })

        "--dry-run": true,

#
# 5. In printHelp() – add to the GIT-LIKE WORKFLOW section:

	fmt.Printf("  %spt log%s                                Show commit history\n", ColorGreen, ColorReset)
	fmt.Printf("  %spt log <id>%s                           Show files in a specific commit\n", ColorGreen, ColorReset)
	fmt.Printf("  %spt reset --last%s                       Restore most recent commit (all files)\n", ColorGreen, ColorReset)
	fmt.Printf("  %spt reset --last -n 2%s                  Restore 2nd most recent commit\n", ColorGreen, ColorReset)
	fmt.Printf("  %spt reset <id>%s                         Restore a specific commit by ID\n", ColorGreen, ColorReset)
	fmt.Printf("  %spt reset <id> --dry-run%s               Preview restore without touching disk\n", ColorGreen, ColorReset)
