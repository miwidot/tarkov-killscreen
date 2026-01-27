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

func main() {
	RunApp()
}
