// log.go - Logging and Console Utilities
//
// Provides debug-only logging functions gated by the debugMode variable
// (set via build tags in debug_release.go / debug_debug.go).
// Also handles enabling ANSI color codes on the Windows console.
package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

// debugLog prints a formatted message only in debug builds.
func debugLog(format string, args ...interface{}) {
	if debugMode {
		fmt.Printf(format, args...)
	}
}

// debugLn prints arguments followed by a newline, only in debug builds.
func debugLn(args ...interface{}) {
	if debugMode {
		fmt.Println(args...)
	}
}

// enableAnsiColors enables ANSI escape codes on Windows console
// by setting the ENABLE_VIRTUAL_TERMINAL_PROCESSING flag.
func enableAnsiColors() {
	handle, _ := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
	var mode uint32
	procGet := kernel32.NewProc("GetConsoleMode")
	procSet := kernel32.NewProc("SetConsoleMode")
	procGet.Call(uintptr(handle), uintptr(unsafe.Pointer(&mode)))
	procSet.Call(uintptr(handle), uintptr(mode|0x0004)) // ENABLE_VIRTUAL_TERMINAL_PROCESSING
}
