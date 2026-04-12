// Tarkov Kill Screen Analyzer
//
// A Windows system tray application that monitors the clipboard for screenshots
// and uploads them to an OCR API for kill tracking.
//
// This application does NOT interact with Escape from Tarkov in any way.
// It only reads images from the Windows clipboard (user-initiated screenshots)
// and uploads them to a web service for text extraction.
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
