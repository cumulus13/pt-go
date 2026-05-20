//go:build !windows
// +build !windows

package main

// setWindowsHiddenAttribute is a no-op on Unix-like systems (Linux, macOS, BSD).
// On Unix, hidden files/directories use a dot prefix (e.g., .pt),
// which is already handled by the directory name itself.
func setWindowsHiddenAttribute(path string) error {
    // No-op: Unix uses dot prefix for hidden files
    return nil
}