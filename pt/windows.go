//go:build windows
// +build windows

package main

import (
    "syscall"
    "golang.org/x/sys/windows"
)

// setWindowsHiddenAttribute sets the hidden attribute on Windows.
// This function makes the .pt directory hidden in Windows Explorer.
func setWindowsHiddenAttribute(path string) error {
    // Convert Go string to Windows UTF-16 string pointer
    ptr, err := syscall.UTF16PtrFromString(path)
    if err != nil {
        return err
    }

    // Get current file attributes
    attributes, err := windows.GetFileAttributes(ptr)
    if err != nil {
        return err
    }

    // Add the FILE_ATTRIBUTE_HIDDEN flag
    newAttributes := attributes | windows.FILE_ATTRIBUTE_HIDDEN

    // Set the new attributes
    return windows.SetFileAttributes(ptr, newAttributes)
}