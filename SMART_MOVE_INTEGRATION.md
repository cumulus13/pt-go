# Smart Move – Integration Guide

## What changed

`smart_move.go` is a **drop-in replacement** for the `handleMoveCommand` function
in `main.go`.  It also adds three new helper functions.

---

## Step 1 – Remove the old function from `main.go`

Delete (or comment out) the existing `handleMoveCommand` function in `main.go`.
Search for the signature:

```go
func handleMoveCommand(args []string, overwrite bool) error {
```

Everything from that line to its matching closing `}` must be removed.

> The helper functions `moveDirectoryWithBackups`, `expandGlobs`,
> `findFilesWithRegex`, and `findFilesRecursive` are **unchanged** – keep them.

---

## Step 2 – Add `smart_move.go` to the package

Copy `smart_move.go` into the `pt/` directory (same folder as `main.go`).
The file declares `package main` so Go picks it up automatically.

---

## Step 3 – Verify no duplicate declarations

`smart_move.go` introduces these new symbols:

| Symbol | Purpose |
|---|---|
| `smartExpandGlobs` | replaces the `expandGlobs` call in the move path |
| `smartDeepGlob` | walks project tree for wildcard matches |
| `resolveSmartMatches` | groups results by base name, resolves ambiguity |
| `promptSmartSelection` | numbered selection table (reuses `printFileSearchResults`) |
| `handleMoveCommand` | replaces the original; must be deleted from `main.go` |

Run `go build ./...` – if you see a "declared and not used" or "redeclared" error
you missed a duplicate.

---

## New behaviour matrix

```
Pattern             Files found in project   -o    Action
──────────────────  ───────────────────────  ────  ──────────────────────────────────
pt move *.go dst/   0 (nowhere)              any   "no files matched *.go" – skipped
pt move *.go dst/   1 (unique)               no    confirm prompt → backup + move
pt move *.go dst/   1 (unique)               yes   backup existing dest → move
pt move *.go dst/   >1 same base name        any   numbered table → user picks one
pt move file.go .   0                        any   pass-through as literal path
pt move file.go .   1                        no    confirm prompt → backup + move
pt move file.go .   >1                       any   numbered table → user picks one
```

### Destination "." handling

`pt move *.go .` resolves destination to the absolute current directory, so
files in subdirectories are moved **into the cwd** (not renamed in place).

### Overwrite without `-o`

When a file already exists at the destination and `-o` was **not** given,
the tool now shows a per-file `y/N` prompt instead of a hard exit-1 error.
Answering `N` skips that file and continues with the rest.

When the answer is `y` (or `-o` was given), the existing destination is
**backed up first** via `autoRenameIfExists` before being replaced.

---

## Example sessions

```
$ pt move *.go archive/
🔍 '*.go' not matched in cwd – searching project tree…
✅ Found 'main.go': /home/user/pt/main.go
✅ Found 'monitor.go': /home/user/pt/monitor.go
🎯 2 file(s) selected

🚚 Moving 2 file(s)…
  Destination: /home/user/archive
  Type: directory

[1/2] /home/user/pt/main.go
  📦 2 backup file(s) to relocate
  ✅ Backups relocated (2 metadata updated)
  ✅ Moved → archive/main.go

[2/2] /home/user/pt/monitor.go
  ✅ Moved → archive/monitor.go

📊 Move summary:
  ✅ 2 moved
  📦 2 backup(s) relocated
```

```
$ pt move *.py .
⚠️  Multiple files named 'utils.py' found:
┌──────────────────────────────────────────────────────────────────────┬────────────────────┬──────────────┐
│ Path                                                                  │ Modified           │         Size │
├──────────────────────────────────────────────────────────────────────┼────────────────────┼──────────────┤
│   1. src/utils.py                                                     │ 2026-05-14 09:12   │       3.2 kB │
│   2. tests/utils.py                                                   │ 2026-05-13 17:44   │       1.1 kB │
└──────────────────────────────────────────────────────────────────────┴────────────────────┴──────────────┘
Select file for 'utils.py' (1-2, 0 = skip): 1
✅ Found 'utils.py': /home/user/project/src/utils.py
…
```

```
$ pt move config.json /etc/app/
[1/1] /home/user/project/config.json
  ⚠️  Destination exists: /etc/app/config.json
  Overwrite? (y/N): y
  📦 Backing up existing destination…
  📦 Backup created: config_json.20260515_103045123456.87654321
  ✅ Moved → /etc/app/config.json
```

---

## No other files need changing

`handleMoveWithInfo` in `main.go` already calls `handleMoveCommand(args, overwrite)`
and parses `-o` / `--overwrite` from `CommandInfo.BoolFlags` – so the routing
layer requires zero edits.
