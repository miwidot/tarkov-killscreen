// app.go - Main Application Logic
//
// This file contains the core application logic:
// - System tray icon and menu
// - Hotkey-based screenshot capture
// - Auto-batching of multiple screenshots (20 second window)
// - Image validation (size, aspect ratio)
// - Coordination of upload and notifications
//
// Auto-batching allows users to take multiple screenshots of a scrollable
// kill list. After 20 seconds of no new screenshots, all images in the
// batch are uploaded together.
package main

import (
	"fmt"
	"image"
	"os"
	"runtime/debug"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/lxn/walk"
)

var (
	mainWindow *walk.MainWindow
	notifyIcon *walk.NotifyIcon
	config     *Config
	watching   bool
	processing bool

	// Auto-batching for multi-screenshot raids
	batchImages    []image.Image
	batchTimer     *time.Timer
	batchMutex     sync.Mutex
	batchWaitTime  = 20 * time.Second // Wait 20 seconds for more screenshots
	batchUploading bool               // Prevent new captures during upload

	// Queue for screenshots taken during upload
	pendingImages []image.Image

	// Session statistics
	totalUploaded int
	totalKills    int
	totalFailed   int
	statsAction   *walk.Action // Tray menu stats line
)

// getTerminalWidth returns the current console window width in columns.
// Falls back to 80 columns if the console info cannot be queried.
func getTerminalWidth() int {
	handle, _ := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
	type consoleInfo struct {
		Size       [2]uint16
		CursorPos  [2]uint16
		Attributes uint16
		Window     [4]uint16
		MaxSize    [2]uint16
	}
	var info consoleInfo
	proc := kernel32.NewProc("GetConsoleScreenBufferInfo")
	ret, _, _ := proc.Call(uintptr(handle), uintptr(unsafe.Pointer(&info)))
	if ret == 0 {
		return 80
	}
	width := int(info.Window[2]-info.Window[0]) + 1
	if width < 40 {
		return 80
	}
	return width
}

// centerLine pads text with leading spaces so it appears centered in a
// terminal of the given width. ANSI escape codes are stripped before
// measuring visible length.
func centerLine(text string, width int) string {
	// Strip ANSI codes to measure visible length
	visible := text
	for {
		start := -1
		for i := 0; i < len(visible); i++ {
			if visible[i] == '\033' {
				start = i
				break
			}
		}
		if start == -1 {
			break
		}
		end := start
		for end < len(visible) && visible[end] != 'm' {
			end++
		}
		if end < len(visible) {
			visible = visible[:start] + visible[end+1:]
		} else {
			break
		}
	}

	pad := (width - len(visible)) / 2
	if pad < 0 {
		pad = 0
	}
	spaces := ""
	for i := 0; i < pad; i++ {
		spaces += " "
	}
	return spaces + text
}

// printBanner prints the colored, centered startup banner to the console.
func printBanner() {
	enableAnsiColors()

	green := "\033[32m"
	yellow := "\033[1;33m"
	dim := "\033[90m"
	reset := "\033[0m"
	w := getTerminalWidth()

	fmt.Println()
	fmt.Println(centerLine(green+"=========================================="+reset, w))
	fmt.Println(centerLine(green+"T  A  R  K  O  V"+reset, w))
	fmt.Println(centerLine(yellow+"-- STAMMTISCH --"+reset, w))
	fmt.Println(centerLine(green+"K I L L C O U N T E R"+reset, w))
	fmt.Println(centerLine(green+"=========================================="+reset, w))
	fmt.Println(centerLine(dim+"Version: "+CurrentVersion+reset, w))
	fmt.Println()
}

// RunApp is the main entry point for the application. It shows the splash
// screen, loads config, creates the system tray icon, and starts the hotkey
// watcher loop.
func RunApp() {
	var err error

	printBanner()
	ShowSplash()

	config, err = LoadConfig()
	if err != nil {
		os.Exit(1)
	}

	// Apply configured language
	SetLanguage(config.Language)

	// Check for updates on startup and every 30 minutes (after language is set)
	go StartUpdateChecker()

	// First run: prompt for API token if not set
	if !HasToken() {
		promptForToken(config)
	}

	// Apply configured hotkey
	if config.Hotkeys.CaptureKey != "" {
		SetHotkey(config.Hotkeys.CaptureKey)
	}

	mainWindow, err = walk.NewMainWindow()
	if err != nil {
		os.Exit(1)
	}

	notifyIcon, err = walk.NewNotifyIcon(mainWindow)
	if err != nil {
		os.Exit(1)
	}
	defer notifyIcon.Dispose()

	icon, err := walk.NewIconFromImageForDPI(createIconImage(), 96)
	if err == nil {
		notifyIcon.SetIcon(icon)
	}

	notifyIcon.SetToolTip(fmt.Sprintf(T("tray.tooltip"), CurrentVersion))
	notifyIcon.SetVisible(true)

	buildTrayMenu()

	watching = true

	// Use hotkey-based capture (registers and watches on same thread)
	go WatchHotkey()
	hotkeyName := GetHotkeyName(currentHotkey)
	showBalloon("Tarkov Screenshoter "+CurrentVersion, fmt.Sprintf(T("ready.capture"), hotkeyName))

	// Warn if Tarkov is running elevated but we are not
	if !isElevated() && IsTarkovRunning() {
		showWarning("Tarkov Screenshoter", T("admin.hint"))
	}

	mainWindow.Run()
}

// watchClipboardAuto polls the clipboard sequence number to detect new
// screenshots. This is the legacy capture method, kept as fallback.
func watchClipboardAuto() {
	lastSeq := GetClipboardSequenceNumber()
	debugLn("[AUTO] Watching clipboard... seq:", lastSeq)

	ticker := 0
	for watching {
		time.Sleep(500 * time.Millisecond)
		ticker++

		// Heartbeat every 30 seconds
		if ticker%60 == 0 {
			debugLog("[AUTO] Heartbeat - still watching, seq: %d\n", lastSeq)
		}

		currentSeq := GetClipboardSequenceNumber()
		if currentSeq != lastSeq {
			lastSeq = currentSeq
			debugLn("[AUTO] Clipboard changed, seq:", currentSeq)

			time.Sleep(300 * time.Millisecond)

			// Try to capture directly - more reliable than checking format first
			go captureAndBatch()
		}
	}
	debugLn("[AUTO] Watcher stopped!")
}

// captureAndBatch reads a screenshot from the clipboard, validates it, and
// adds it to the current batch. Used by the clipboard watcher path.
func captureAndBatch() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[AUTO] PANIC recovered: %v\n", r) // Always log panics
		}
	}()

	// Check if Tarkov is running (outside of lock)
	tarkovRunning := IsTarkovRunning()
	debugLog("[AUTO] Tarkov running: %v\n", tarkovRunning)
	if !tarkovRunning {
		debugLn("[AUTO] Tarkov not running, ignoring screenshot")
		return
	}

	// Get image from clipboard (outside of lock - clipboard has its own lock)
	img, err := GetClipboardImage()
	if err != nil || img == nil {
		debugLn("[AUTO] No image in clipboard (text/other)")
		return
	}

	debugLog("[AUTO] Got image %dx%d\n", img.Bounds().Dx(), img.Bounds().Dy())

	width := img.Bounds().Dx()
	height := img.Bounds().Dy()

	// Check minimum size (server requires 800x400)
	if width < 800 || height < 400 {
		debugLog("[AUTO] Image too small (%dx%d), minimum 800x400, skipping...\n", width, height)
		return
	}

	// Check for valid aspect ratios
	aspectRatio := float64(width) / float64(height)
	validRatio := isValidAspectRatio(aspectRatio)
	if !validRatio {
		debugLog("[AUTO] Invalid aspect ratio (%.2f), skipping...\n", aspectRatio)
		showWarning(T("invalid.screenshot"), T("invalid.aspect"))
		return
	}

	// Now lock to add to batch or pending queue
	batchMutex.Lock()

	// Add to pending queue if uploading, otherwise add to batch
	if batchUploading {
		pendingImages = append(pendingImages, img)
		count := len(pendingImages)
		debugLog("[PENDING] Image %d added to pending queue (upload in progress)\n", count)
		showBalloon(T("screenshot.queued"), fmt.Sprintf(T("screenshot.queued.count"), count))
		batchMutex.Unlock()
		return
	}

	// Add image to batch
	batchImages = append(batchImages, img)
	count := len(batchImages)
	debugLog("[BATCH] Image %d added to batch\n", count)

	// Show notification
	if count == 1 {
		showBalloon(T("screenshot.captured"), T("screenshot.waiting"))
	} else {
		showBalloon(T("screenshot.captured"), fmt.Sprintf(T("screenshot.batch"), count))
	}

	// Reset timer
	if batchTimer != nil {
		batchTimer.Stop()
	}
	batchTimer = time.AfterFunc(batchWaitTime, processBatch)

	batchMutex.Unlock()
}

// processBatch uploads all batched screenshots to the OCR API, saves any
// detected kills, and shows the result notification. Called after the 20s
// batch timer expires.
func processBatch() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[BATCH] PANIC recovered: %v\n", r) // Always log panics
		}
		batchMutex.Lock()
		batchUploading = false

		// Check if there are pending images from during upload
		if len(pendingImages) > 0 {
			debugLog("[BATCH] Moving %d pending images to new batch\n", len(pendingImages))
			batchImages = pendingImages
			pendingImages = nil
			// Start timer for the new batch
			batchTimer = time.AfterFunc(batchWaitTime, processBatch)
			showBalloon(T("batch.new"), fmt.Sprintf(T("batch.new.msg"), len(batchImages)))
		}

		batchMutex.Unlock()
		fmt.Println("[READY] Waiting for screenshots...")
	}()

	batchMutex.Lock()
	images := batchImages
	batchImages = nil
	batchTimer = nil
	batchUploading = true
	batchMutex.Unlock()

	if len(images) == 0 {
		return
	}

	// Verify all images have our signature (safety check)
	validImages := make([]image.Image, 0, len(images))
	for i, img := range images {
		if VerifySignature(img) {
			debugLog("[SIGNATURE] Image %d verified ✓\n", i+1)
			validImages = append(validImages, img)
		} else {
			debugLog("[SIGNATURE] Image %d FAILED verification, skipping\n", i+1)
		}
	}

	if len(validImages) == 0 {
		fmt.Println("[BATCH] No valid images to upload") // User-facing error
		totalFailed++
		updateStatsAction()
		showWarning(T("upload.failed"), T("batch.novalid"))
		return
	}

	images = validImages
	uploadCount := len(images)
	fmt.Printf("[BATCH] Processing %d images...\n", uploadCount) // User-facing status
	showBalloon(T("batch.processing"), fmt.Sprintf(T("batch.uploading"), uploadCount))

	var resp *OCRResponse
	var err error

	if len(images) == 1 {
		resp, err = UploadScreenshot(images[0], config)
	} else {
		resp, err = UploadMultipleScreenshots(images, config)
	}

	// Clear images to free memory (~8MB per image)
	for i := range images {
		images[i] = nil
	}
	images = nil
	debug.FreeOSMemory() // Force Go to return memory to OS

	if err != nil {
		fmt.Println("[BATCH] Error:", err)
		totalFailed++
		updateStatsAction()
		showBalloon(T("error"), err.Error())
		if config.Feedback.OverlayEnabled {
			ShowOverlayMessage(T("overlay.upload.failed"), err.Error())
		}
		return
	}

	if resp != nil && resp.Success {
		totalUploaded += uploadCount
		summary := FormatKillSummary(resp)

		// Check if some images were invalid (but we may still have valid kills)
		if !IsValidTarkovScreenshot(resp) {
			debugLn("[BATCH] Some images invalid, checking for valid kills...")
		}

		fmt.Println("[BATCH] Result:", summary)

		// Determine overlay line 1: prefer API message, fall back to default
		overlayMsg := resp.Message

		if resp.Data.TotalKills > 0 {
			totalKills += resp.Data.TotalKills
			// Save kills even if some images were invalid - server filters invalid ones
			saveResp, err := SaveKills(resp, config)
			if err != nil {
				fmt.Println("[BATCH] Save error:", err)
				showBalloon(T("kills.analysis"), summary+" (not saved: "+err.Error()+")")
			} else {
				fmt.Println("[BATCH] Saved! RaidID:", saveResp.RaidID)
				showBalloon(T("kills.saved"), summary)
			}
			if config.Feedback.OverlayEnabled {
				if overlayMsg == "" {
					overlayMsg = T("kills.saved")
				}
				ShowOverlayMessage(overlayMsg, summary)
			}
		} else if !IsValidTarkovScreenshot(resp) {
			// No kills and invalid images - warn user
			reason := FormatKillSummary(resp)
			showWarning(T("not.tarkov"), reason)
			if config.Feedback.OverlayEnabled {
				if overlayMsg == "" {
					overlayMsg = T("not.tarkov")
				}
				ShowOverlayMessage(overlayMsg, reason)
			}
		} else {
			showBalloon(T("analysis.complete"), summary)
			if config.Feedback.OverlayEnabled {
				if overlayMsg == "" {
					overlayMsg = T("analysis.complete")
				}
				ShowOverlayMessage(overlayMsg, summary)
			}
		}
	} else {
		showBalloon(T("batch.done"), T("batch.nokills"))
		if config.Feedback.OverlayEnabled {
			ShowOverlayMessage(T("batch.done"), T("batch.nokills"))
		}
	}

	updateStatsAction()
}

// buildTrayMenu populates the system tray context menu with status info,
// a "Process Now" action, settings, and exit.
func buildTrayMenu() {
	statusAction := walk.NewAction()
	hotkeyLabel := GetHotkeyName(currentHotkey)
	statusAction.SetText(fmt.Sprintf(T("tray.hotkey"), hotkeyLabel))
	statusAction.SetEnabled(false)
	notifyIcon.ContextMenu().Actions().Add(statusAction)

	tokenAction := walk.NewAction()
	updateTokenAction(tokenAction)
	tokenAction.SetEnabled(false)
	notifyIcon.ContextMenu().Actions().Add(tokenAction)

	statsAction = walk.NewAction()
	updateStatsAction()
	statsAction.SetEnabled(false)
	notifyIcon.ContextMenu().Actions().Add(statsAction)

	notifyIcon.ContextMenu().Actions().Add(walk.NewSeparatorAction())

	// Process now (skip wait)
	processNowAction := walk.NewAction()
	processNowAction.SetText(T("tray.processnow"))
	processNowAction.Triggered().Attach(func() {
		batchMutex.Lock()
		if batchTimer != nil {
			batchTimer.Stop()
			batchTimer = nil
		}
		batchMutex.Unlock()
		go processBatch()
	})
	notifyIcon.ContextMenu().Actions().Add(processNowAction)

	settingsAction := walk.NewAction()
	settingsAction.SetText(T("tray.settings"))
	settingsAction.Triggered().Attach(func() {
		showSettings()
	})
	notifyIcon.ContextMenu().Actions().Add(settingsAction)

	// Show "Restart as Admin" when not elevated
	if !isElevated() {
		adminAction := walk.NewAction()
		adminAction.SetText(T("tray.admin"))
		adminAction.Triggered().Attach(func() {
			if err := restartAsAdmin(); err != nil {
				showWarning(T("admin.restart.failed"), err.Error())
			}
		})
		notifyIcon.ContextMenu().Actions().Add(adminAction)
	}

	notifyIcon.ContextMenu().Actions().Add(walk.NewSeparatorAction())

	exitAction := walk.NewAction()
	exitAction.SetText(T("tray.exit"))
	exitAction.Triggered().Attach(func() {
		watching = false
		unregisterGlobalHotkey()
		restoreSnippingToolPrintScreen()
		walk.App().Exit(0)
	})
	notifyIcon.ContextMenu().Actions().Add(exitAction)
}

// updateTokenAction sets the tray menu label to show a masked token preview.
func updateTokenAction(action *walk.Action) {
	if HasToken() {
		token, _ := LoadToken()
		if len(token) > 8 {
			action.SetText("Token: " + token[:4] + "..." + token[len(token)-4:])
		} else {
			action.SetText("Token: OK")
		}
	} else {
		action.SetText(T("tray.token.notset"))
	}
}

// updateStatsAction refreshes the session stats line in the tray menu.
func updateStatsAction() {
	if statsAction != nil {
		statsAction.SetText(fmt.Sprintf(T("tray.stats"), totalUploaded, totalKills, totalFailed))
	}
	// Also update tooltip with stats
	if notifyIcon != nil {
		notifyIcon.SetToolTip(fmt.Sprintf(T("tray.tooltip"), CurrentVersion))
	}
}

// showSettings opens the settings dialog and reloads config on save.
func showSettings() {
	saved, _ := ShowSettingsDialog(nil, config)
	if saved {
		config, _ = LoadConfig()
		SetLanguage(config.Language)
		showBalloon(T("settings.title"), T("settings.updated"))
	}
}

// showBalloon displays an info balloon notification from the system tray icon.
func showBalloon(title, message string) {
	if notifyIcon != nil {
		notifyIcon.ShowInfo(title, message)
	}
}

// showWarning displays a warning balloon notification from the system tray icon.
func showWarning(title, message string) {
	if notifyIcon != nil {
		notifyIcon.ShowWarning(title, message)
	}
}

// isValidAspectRatio checks if the aspect ratio is reasonable for a game screenshot
func isValidAspectRatio(ratio float64) bool {
	// Allow anything between 4:3 (1.33) and 32:9 (3.56)
	// This covers all standard gaming resolutions plus cropped regions
	// Only block: portrait mode (<1.2) and multi-monitor (>3.8)
	return ratio >= 1.2 && ratio <= 3.8
}

