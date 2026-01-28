// app.go - Main Application Logic
//
// This file contains the core application logic:
// - System tray icon and menu
// - Clipboard monitoring loop
// - Auto-batching of multiple screenshots (20 second window)
// - Image validation (size, aspect ratio, duplicates)
// - Coordination of upload and notifications
//
// The clipboard watcher runs in a background goroutine, polling the
// clipboard sequence number every 500ms. When it changes and contains
// an image, we capture it and add it to the current batch.
//
// Auto-batching allows users to take multiple screenshots of a scrollable
// kill list. After 20 seconds of no new screenshots, all images in the
// batch are uploaded together.
package main

import (
	"fmt"
	"image"
	"os"
	"os/exec"
	"sync"
	"time"

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
)

func RunApp() {
	var err error

	ShowSplash()

	// Check for updates on startup and every 30 minutes
	go StartUpdateChecker()

	config, err = LoadConfig()
	if err != nil {
		os.Exit(1)
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

	notifyIcon.SetToolTip(fmt.Sprintf("Tarkov Screenshoter %s - Auto-capture active", CurrentVersion))
	notifyIcon.SetVisible(true)

	buildTrayMenu()

	watching = true
	go watchClipboardAuto()

	showBalloon("Tarkov Screenshoter "+CurrentVersion, "Auto-capture active! Screenshots are auto-batched (20s window).")

	mainWindow.Run()
}

func watchClipboardAuto() {
	lastSeq := GetClipboardSequenceNumber()
	fmt.Println("[AUTO] Watching clipboard... seq:", lastSeq)

	ticker := 0
	for watching {
		time.Sleep(500 * time.Millisecond)
		ticker++

		// Heartbeat every 30 seconds
		if ticker%60 == 0 {
			fmt.Printf("[AUTO] Heartbeat - still watching, seq: %d\n", lastSeq)
		}

		currentSeq := GetClipboardSequenceNumber()
		if currentSeq != lastSeq {
			lastSeq = currentSeq
			fmt.Println("[AUTO] Clipboard changed, seq:", currentSeq)

			time.Sleep(300 * time.Millisecond)

			// Try to capture directly - more reliable than checking format first
			go captureAndBatch()
		}
	}
	fmt.Println("[AUTO] Watcher stopped!")
}

func captureAndBatch() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[AUTO] PANIC recovered: %v\n", r)
		}
	}()

	// Check if Tarkov is running
	tarkovRunning := IsTarkovRunning()
	fmt.Printf("[AUTO] Tarkov running: %v\n", tarkovRunning)
	if !tarkovRunning {
		fmt.Println("[AUTO] Tarkov not running, ignoring screenshot")
		return
	}

	batchMutex.Lock()

	isUploading := batchUploading
	batchMutex.Unlock() // Unlock early to get image without holding lock

	img, err := GetClipboardImage()
	if err != nil || img == nil {
		fmt.Println("[AUTO] No image in clipboard (text/other)")
		return
	}

	fmt.Printf("[AUTO] Got image %dx%d\n", img.Bounds().Dx(), img.Bounds().Dy())

	width := img.Bounds().Dx()
	height := img.Bounds().Dy()

	// Check minimum size (server requires 800x400)
	if width < 800 || height < 400 {
		fmt.Printf("[AUTO] Image too small (%dx%d), minimum 800x400, skipping...\n", width, height)
		return
	}

	// Check for valid aspect ratios
	aspectRatio := float64(width) / float64(height)
	validRatio := isValidAspectRatio(aspectRatio)
	if !validRatio {
		fmt.Printf("[AUTO] Invalid aspect ratio (%.2f), skipping...\n", aspectRatio)
		showWarning("Invalid Screenshot", "Aspect ratio not supported. Use 16:9, 16:10, 21:9, or 4:3")
		return
	}

	// Add to pending queue if uploading, otherwise add to batch
	batchMutex.Lock()
	if isUploading {
		pendingImages = append(pendingImages, img)
		count := len(pendingImages)
		fmt.Printf("[PENDING] Image %d added to pending queue (upload in progress)\n", count)
		showBalloon("Screenshot queued", fmt.Sprintf("%d screenshot(s) waiting for next batch", count))
		batchMutex.Unlock()
		return
	}

	// Add image to batch
	batchImages = append(batchImages, img)
	count := len(batchImages)
	fmt.Printf("[BATCH] Image %d added to batch\n", count)

	// Show notification
	if count == 1 {
		showBalloon("Screenshot captured", "Waiting 20s for more screenshots...")
	} else {
		showBalloon("Screenshot captured", fmt.Sprintf("%d screenshots in batch. Waiting 20s...", count))
	}

	// Reset timer
	if batchTimer != nil {
		batchTimer.Stop()
	}
	batchTimer = time.AfterFunc(batchWaitTime, processBatch)

	batchMutex.Unlock()
}

func processBatch() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[BATCH] PANIC recovered: %v\n", r)
		}
		batchMutex.Lock()
		batchUploading = false

		// Check if there are pending images from during upload
		if len(pendingImages) > 0 {
			fmt.Printf("[BATCH] Moving %d pending images to new batch\n", len(pendingImages))
			batchImages = pendingImages
			pendingImages = nil
			// Start timer for the new batch
			batchTimer = time.AfterFunc(batchWaitTime, processBatch)
			showBalloon("New batch started", fmt.Sprintf("%d screenshot(s) from queue. Waiting 20s...", len(batchImages)))
		}

		batchMutex.Unlock()
		fmt.Println("[BATCH] Ready for new screenshots")
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

	fmt.Printf("[BATCH] Processing %d images...\n", len(images))
	showBalloon("Processing", fmt.Sprintf("Uploading %d screenshot(s)...", len(images)))

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

	if err != nil {
		fmt.Println("[BATCH] Error:", err)
		showBalloon("Error", err.Error())
		return
	}

	if resp != nil && resp.Success {
		if !IsValidTarkovScreenshot(resp) {
			reason := FormatKillSummary(resp)
			fmt.Println("[BATCH] Invalid images:", reason)
			showWarning("Not Tarkov Screenshots", reason)
			return
		}

		summary := FormatKillSummary(resp)
		fmt.Println("[BATCH] Success:", summary)

		if resp.Data.TotalKills > 0 {
			saveResp, err := SaveKills(resp, config)
			if err != nil {
				fmt.Println("[BATCH] Save error:", err)
				showBalloon("Kill Analysis", summary+" (not saved: "+err.Error()+")")
			} else {
				fmt.Println("[BATCH] Saved! RaidID:", saveResp.RaidID)
				showBalloon("Kills Saved!", summary)
			}
		} else {
			showBalloon("Analysis Complete", summary)
		}
	} else {
		showBalloon("Done", "No kills detected")
	}
}

func buildTrayMenu() {
	statusAction := walk.NewAction()
	statusAction.SetText("Auto-capture: ON (20s batch window)")
	statusAction.SetEnabled(false)
	notifyIcon.ContextMenu().Actions().Add(statusAction)

	tokenAction := walk.NewAction()
	updateTokenAction(tokenAction)
	tokenAction.SetEnabled(false)
	notifyIcon.ContextMenu().Actions().Add(tokenAction)

	notifyIcon.ContextMenu().Actions().Add(walk.NewSeparatorAction())

	// Process now (skip wait)
	processNowAction := walk.NewAction()
	processNowAction.SetText("Process Now (skip wait)")
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

	openFolderAction := walk.NewAction()
	openFolderAction.SetText("Open Screenshots Folder")
	openFolderAction.Triggered().Attach(func() {
		openExplorer(config.ScreenshotPath)
	})
	notifyIcon.ContextMenu().Actions().Add(openFolderAction)

	settingsAction := walk.NewAction()
	settingsAction.SetText("Settings...")
	settingsAction.Triggered().Attach(func() {
		showSettings()
	})
	notifyIcon.ContextMenu().Actions().Add(settingsAction)

	notifyIcon.ContextMenu().Actions().Add(walk.NewSeparatorAction())

	exitAction := walk.NewAction()
	exitAction.SetText("Exit")
	exitAction.Triggered().Attach(func() {
		watching = false
		walk.App().Exit(0)
	})
	notifyIcon.ContextMenu().Actions().Add(exitAction)
}

func updateTokenAction(action *walk.Action) {
	if HasToken() {
		token, _ := LoadToken()
		if len(token) > 8 {
			action.SetText("Token: " + token[:4] + "..." + token[len(token)-4:])
		} else {
			action.SetText("Token: OK")
		}
	} else {
		action.SetText("Token: NOT SET")
	}
}

func showSettings() {
	saved, _ := ShowSettingsDialog(nil, config)
	if saved {
		config, _ = LoadConfig()
		showBalloon("Settings", "Configuration updated")
	}
}

func showBalloon(title, message string) {
	if notifyIcon != nil {
		notifyIcon.ShowInfo(title, message)
	}
}

func showWarning(title, message string) {
	if notifyIcon != nil {
		notifyIcon.ShowWarning(title, message)
	}
}

func openExplorer(path string) {
	exec.Command("explorer", path).Start()
}

// isValidAspectRatio checks if the aspect ratio is reasonable for a game screenshot
func isValidAspectRatio(ratio float64) bool {
	// Allow anything between 4:3 (1.33) and 32:9 (3.56)
	// This covers all standard gaming resolutions plus cropped regions
	// Only block: portrait mode (<1.2) and multi-monitor (>3.8)
	return ratio >= 1.2 && ratio <= 3.8
}

