//go:build unix

package main

// macOS and Linux specifics. The Windows counterparts are in platform_windows.go.

import (
	"io/fs"
	"os"
	"syscall"
)

// enableTerminalColors reports whether stdout is an interactive terminal,
// which interprets ANSI color sequences on macOS and Linux.
func enableTerminalColors() bool {
	info, err := os.Stdout.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// allocatedSize returns the disk space a file occupies: its allocated blocks
// (always counted in 512-byte units), as "du" and Git itself measure it. It
// matters for Git's many small loose objects: a 200-byte object still uses a
// whole 4 KiB block.
func allocatedSize(info fs.FileInfo) int64 {
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return int64(st.Blocks) * 512
	}
	return info.Size()
}
