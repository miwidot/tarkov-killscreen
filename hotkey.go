// hotkey.go - Global Hotkey Registration and Screenshot Capture
//
// This file provides direct screenshot capture using Windows hotkeys.
// Instead of monitoring the clipboard (which has many issues), we:
// - Register a global hotkey (Print Screen or custom)
// - Capture the screen directly using Windows API
// - Process the image immediately
//
// This approach avoids all clipboard-related issues:
// - No clipboard lock contention
// - No sequence number bugs
// - No interference from other apps
// - Full control over image quality
package main

import (
	"fmt"
	"image"
	"time"

	"github.com/kbinani/screenshot"
)

var (
	procGetAsyncKeyState = user32.NewProc("GetAsyncKeyState")
)

const (
	VK_SNAPSHOT = 0x2C // Print Screen key
)

// RegisterScreenshotHotkey is a no-op for polling approach
func RegisterScreenshotHotkey() error {
	return nil
}

// UnregisterScreenshotHotkey is a no-op for polling approach
func UnregisterScreenshotHotkey() {
}

// WatchHotkey polls for Print Screen key press
func WatchHotkey() {
	fmt.Println("[HOTKEY] Watching for Print Screen key (polling)...")

	var lastState int16 = 0

	for watching {
		time.Sleep(50 * time.Millisecond) // Poll every 50ms

		// GetAsyncKeyState returns key state
		// High bit (0x8000) = key is currently down
		// Low bit (0x0001) = key was pressed since last call
		ret, _, _ := procGetAsyncKeyState.Call(VK_SNAPSHOT)
		state := int16(ret)

		// Detect key press (transition from not pressed to pressed)
		// High bit set = negative in int16, so state < 0 means key is down
		if state < 0 && lastState >= 0 {
			fmt.Println("[HOTKEY] Print Screen pressed!")
			go captureScreen()
		}

		lastState = state
	}

	fmt.Println("[HOTKEY] Watcher stopped!")
}

// captureScreen takes a screenshot directly (no clipboard involved)
func captureScreen() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[CAPTURE] PANIC recovered: %v\n", r)
		}
	}()

	// Check if Tarkov is running
	if !IsTarkovRunning() {
		fmt.Println("[CAPTURE] Tarkov not running, ignoring")
		return
	}

	// Get the primary display bounds
	n := screenshot.NumActiveDisplays()
	if n == 0 {
		fmt.Println("[CAPTURE] No displays found")
		return
	}

	// Capture primary display (index 0)
	bounds := screenshot.GetDisplayBounds(0)
	fmt.Printf("[CAPTURE] Capturing display 0: %dx%d\n", bounds.Dx(), bounds.Dy())

	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		fmt.Printf("[CAPTURE] Failed to capture: %v\n", err)
		return
	}

	fmt.Printf("[CAPTURE] Got image %dx%d\n", img.Bounds().Dx(), img.Bounds().Dy())

	// Add to batch for processing
	addToBatch(img)
}

// addToBatch adds the captured image to the batch
func addToBatch(img image.Image) {
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()

	// Check minimum size
	if width < 800 || height < 400 {
		fmt.Printf("[CAPTURE] Image too small (%dx%d)\n", width, height)
		return
	}

	// Check aspect ratio
	aspectRatio := float64(width) / float64(height)
	if !isValidAspectRatio(aspectRatio) {
		fmt.Printf("[CAPTURE] Invalid aspect ratio (%.2f)\n", aspectRatio)
		return
	}

	// Add to batch
	batchMutex.Lock()

	if batchUploading {
		pendingImages = append(pendingImages, img)
		count := len(pendingImages)
		fmt.Printf("[PENDING] Image %d added to pending queue\n", count)
		showBalloon("Screenshot queued", fmt.Sprintf("%d screenshot(s) waiting", count))
		batchMutex.Unlock()
		return
	}

	batchImages = append(batchImages, img)
	count := len(batchImages)
	fmt.Printf("[BATCH] Image %d added to batch\n", count)

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
