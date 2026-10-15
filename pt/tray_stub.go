//go:build !windows && !tray
// +build !windows,!tray

package main

// startTray is a no-op: system tray support is not compiled into this build.
// (Linux/macOS tray needs CGO; build with `CGO_ENABLED=1 go build -tags tray ./pt`.)
func startTray() {}
