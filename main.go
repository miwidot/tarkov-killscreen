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
	// Prevent multiple instances
	if IsAlreadyRunning() {
		walk.MsgBox(nil, "Tarkov Kill Screen Analyzer",
			"Die Anwendung läuft bereits.\n\nThe application is already running.",
			walk.MsgBoxIconWarning)
		fmt.Println("Another instance is already running. Exiting.")
		os.Exit(0)
	}

	RunApp()
}
