// Tarkov Kill Screen Analyzer
//
// A Windows system tray application that captures the kill screen on a
// user-pressed hotkey and uploads it to an OCR API for kill tracking.
//
// This application does NOT interact with Escape from Tarkov in any way.
// A capture happens only when the user presses the hotkey while Tarkov is the
// active foreground window (see hotkey.go). Nothing is captured automatically,
// in the background, or while any other program is in focus. The clipboard is
// never read — an earlier version used it, the current one does not.
//
// What leaves the machine is documented in TRANSPARENCY.md.
//
// Author: miwidot
// License: MIT
package main

import (
	"fmt"
	"os"

	"github.com/lxn/walk"
)

func main() {
	// Clean up old exe from previous self-update
	CleanupOldExe()

	// Check for --selfupdate-test flag
	for _, arg := range os.Args[1:] {
		if len(arg) > len("--selfupdate-test=") && arg[:len("--selfupdate-test=")] == "--selfupdate-test=" {
			selfUpdateTestPath = arg[len("--selfupdate-test="):]
			fmt.Println("[UPDATE] Test mode: update source =", selfUpdateTestPath)
		}
	}

	// Prevent multiple instances
	if IsAlreadyRunning() {
		walk.MsgBox(nil, T("already.running.title"),
			T("already.running.msg"),
			walk.MsgBoxIconWarning)
		fmt.Println("Another instance is already running. Exiting.")
		os.Exit(0)
	}

	RunApp()
}
