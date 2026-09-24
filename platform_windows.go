//go:build windows

package main

// Windows specifics. The macOS and Linux counterparts are in platform_unix.go.

import (
	"io/fs"
	"os"
	"syscall"
)

// enableVirtualTerminalProcessing is the console mode flag that makes the
// Windows console interpret ANSI escape sequences (Windows 10 and later).
const enableVirtualTerminalProcessing = 0x0004

// SetConsoleMode is not exposed by the syscall package: load it from kernel32.
var procSetConsoleMode = syscall.NewLazyDLL("kernel32.dll").NewProc("SetConsoleMode")

// enableTerminalColors reports whether stdout is a console able to display
// ANSI colors, switching escape-sequence processing on if needed. It returns
// false when stdout is redirected to a file or a pipe (Git Bash/mintty
// included), or when the console is too old to support it.
func enableTerminalColors() bool {
	handle := syscall.Handle(os.Stdout.Fd())
	var mode uint32
	if err := syscall.GetConsoleMode(handle, &mode); err != nil {
		return false // not a console
	}
	if mode&enableVirtualTerminalProcessing != 0 {
		return true
	}
	ok, _, _ := procSetConsoleMode.Call(uintptr(handle), uintptr(mode|enableVirtualTerminalProcessing))
	return ok != 0
}

// allocatedSize returns the size of a file. os.Stat does not expose the
// allocated size on Windows (NTFS even stores tiny files inside its file
// table), so the logical size is the best cheap approximation.
func allocatedSize(info fs.FileInfo) int64 {
	return info.Size()
}
