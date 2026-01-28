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
	currentHotkey        uintptr = VK_SNAPSHOT // Default to Print Screen
)

// Virtual key codes
const (
	VK_SNAPSHOT   = 0x2C // Print Screen
	VK_F11        = 0x7A // F11
	VK_F12        = 0x7B // F12
	VK_SCROLL     = 0x91 // Scroll Lock
	VK_PAUSE      = 0x13 // Pause/Break
)

// HotkeyOptions defines available hotkey choices for the dropdown
var HotkeyOptions = []string{
	"PrintScreen",
	"F12",
	"F11",
	"ScrollLock",
	"Pause",
}

// HotkeyLabels maps key names to display labels
var HotkeyLabels = map[string]string{
	"PrintScreen": "Print Screen",
	"F12":         "F12",
	"F11":         "F11",
	"ScrollLock":  "Scroll Lock",
	"Pause":       "Pause/Break",
}

// hotkeyToVK maps key names to virtual key codes
var hotkeyToVK = map[string]uintptr{
	"PrintScreen": VK_SNAPSHOT,
	"F12":         VK_F12,
	"F11":         VK_F11,
	"ScrollLock":  VK_SCROLL,
	"Pause":       VK_PAUSE,
}

// SetHotkey updates the capture hotkey
func SetHotkey(keyName string) {
	if vk, ok := hotkeyToVK[keyName]; ok {
		currentHotkey = vk
		fmt.Printf("[HOTKEY] Set capture key to: %s (0x%02X)\n", keyName, vk)
	}
}

// WatchHotkey polls for configured hotkey press
func WatchHotkey() {
	keyName := GetHotkeyName(currentHotkey)
	fmt.Printf("[HOTKEY] Watching for %s key (polling)...\n", keyName)

	var lastState int16 = 0

	for watching {
		time.Sleep(50 * time.Millisecond) // Poll every 50ms

		// GetAsyncKeyState returns key state
		// High bit (0x8000) = key is currently down
		// Low bit (0x0001) = key was pressed since last call
		ret, _, _ := procGetAsyncKeyState.Call(currentHotkey)
		state := int16(ret)

		// Detect key press (transition from not pressed to pressed)
		// High bit set = negative in int16, so state < 0 means key is down
		if state < 0 && lastState >= 0 {
			fmt.Printf("[HOTKEY] %s pressed!\n", keyName)
			go captureScreen()
		}

		lastState = state
	}

	fmt.Println("[HOTKEY] Watcher stopped!")
}

// GetHotkeyName returns the display name for a virtual key code
func GetHotkeyName(vk uintptr) string {
	for name, code := range hotkeyToVK {
		if code == vk {
			if label, ok := HotkeyLabels[name]; ok {
				return label
			}
			return name
		}
	}
	return fmt.Sprintf("Unknown (0x%02X)", vk)
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
