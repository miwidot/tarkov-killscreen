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
	"image/jpeg"
	"os"
	"path/filepath"
	"time"

	"github.com/kbinani/screenshot"
	"golang.org/x/sys/windows/registry"
)

// DebugSaveScreenshots controls whether captured images are saved to a debug/
// folder next to the executable. Enabled automatically in debug builds.
var DebugSaveScreenshots = debugMode

func init() {
	// Set up display helper functions to avoid import cycle
	SetDisplayFuncs(
		func() int { return screenshot.NumActiveDisplays() },
		func(i int) RECT {
			b := screenshot.GetDisplayBounds(i)
			return RECT{
				Left:   int32(b.Min.X),
				Top:    int32(b.Min.Y),
				Right:  int32(b.Max.X),
				Bottom: int32(b.Max.Y),
			}
		},
	)
}

// debugSaveScreenshot writes img as JPEG to the debug/ folder with a timestamped
// filename. No-op when DebugSaveScreenshots is false.
func debugSaveScreenshot(img image.Image, suffix string) {
	if !DebugSaveScreenshots {
		return
	}

	// Save to exe directory / debug folder
	exe, err := os.Executable()
	if err != nil {
		debugLog("[DEBUG] Failed to get executable path: %v\n", err)
		return
	}
	exe, _ = filepath.EvalSymlinks(exe)
	debugDir := filepath.Join(filepath.Dir(exe), "debug")
	debugLog("[DEBUG] Saving to: %s\n", debugDir)
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		debugLog("[DEBUG] Failed to create debug dir: %v\n", err)
		return
	}

	filename := fmt.Sprintf("capture_%s_%s.jpg", time.Now().Format("2006-01-02_15-04-05"), suffix)
	filepath := filepath.Join(debugDir, filename)

	file, err := os.Create(filepath)
	if err != nil {
		debugLog("[DEBUG] Failed to create file: %v\n", err)
		return
	}
	defer file.Close()

	if err := jpeg.Encode(file, img, &jpeg.Options{Quality: 95}); err != nil {
		debugLog("[DEBUG] Failed to encode JPEG: %v\n", err)
		return
	}

	debugLog("[DEBUG] Saved: %s\n", filepath)
}

var (
	procGetAsyncKeyState  = user32.NewProc("GetAsyncKeyState")
	procRegisterHotKey    = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey  = user32.NewProc("UnregisterHotKey")
	currentHotkey         uintptr = VK_SNAPSHOT // Default to Print Screen
	hotkeyRegistered      bool
)

const hotkeyID = 1      // ID for RegisterHotKey
const maxBatchSize = 10 // Maximum screenshots per batch/upload

// registerGlobalHotkey uses RegisterHotKey to claim Print Screen globally.
// This prevents Windows 11 Snipping Tool from hijacking the key.
func registerGlobalHotkey(vk uintptr) {
	unregisterGlobalHotkey()
	if vk == VK_SNAPSHOT {
		ret, _, err := procRegisterHotKey.Call(0, hotkeyID, 0, vk)
		if ret != 0 {
			hotkeyRegistered = true
			debugLn("[HOTKEY] Registered PrintScreen globally (Snipping Tool blocked)")
		} else {
			debugLog("[HOTKEY] Failed to register PrintScreen: %v\n", err)
		}
	}
}

func unregisterGlobalHotkey() {
	if hotkeyRegistered {
		procUnregisterHotKey.Call(0, hotkeyID)
		hotkeyRegistered = false
		debugLn("[HOTKEY] Unregistered global hotkey")
	}
}

// Virtual key codes
const (
	VK_SNAPSHOT      = 0x2C // Print Screen
	VK_F2            = 0x71 // F2
	VK_F3            = 0x72 // F3
	VK_F4            = 0x73 // F4
	VK_F5            = 0x74 // F5
	VK_F6            = 0x75 // F6
	VK_F7            = 0x76 // F7
	VK_F8            = 0x77 // F8
	VK_F9            = 0x78 // F9
	VK_F10           = 0x79 // F10
	VK_F11           = 0x7A // F11
	VK_F12           = 0x7B // F12
	VK_SCROLL        = 0x91 // Scroll Lock
	VK_PAUSE         = 0x13 // Pause/Break
	VK_PRIOR         = 0x21 // Page Up
	VK_NEXT          = 0x22 // Page Down
	VK_HOME          = 0x24 // Home
	VK_END           = 0x23 // End
	VK_INSERT        = 0x2D // Insert
	VK_DELETE        = 0x2E // Delete
	VK_NUMPAD0       = 0x60 // Numpad 0
	VK_NUMPAD1       = 0x61 // Numpad 1
	VK_NUMPAD2       = 0x62 // Numpad 2
	VK_NUMPAD3       = 0x63 // Numpad 3
	VK_NUMPAD4       = 0x64 // Numpad 4
	VK_NUMPAD5       = 0x65 // Numpad 5
	VK_NUMPAD6       = 0x66 // Numpad 6
	VK_NUMPAD7       = 0x67 // Numpad 7
	VK_NUMPAD8       = 0x68 // Numpad 8
	VK_NUMPAD9       = 0x69 // Numpad 9
	VK_MULTIPLY      = 0x6A // Numpad *
	VK_ADD           = 0x6B // Numpad +
	VK_SUBTRACT      = 0x6D // Numpad -
	VK_DIVIDE        = 0x6F // Numpad /
)

// HotkeyOptions defines available hotkey choices for the dropdown
var HotkeyOptions = []string{
	"PrintScreen",
	"F2",
	"F3",
	"F4",
	"F5",
	"F6",
	"F7",
	"F8",
	"F9",
	"F10",
	"F11",
	"F12",
	"ScrollLock",
	"Pause",
	"PageUp",
	"PageDown",
	"Home",
	"End",
	"Insert",
	"Delete",
	"Numpad0",
	"Numpad1",
	"Numpad2",
	"Numpad3",
	"Numpad4",
	"Numpad5",
	"Numpad6",
	"Numpad7",
	"Numpad8",
	"Numpad9",
	"NumpadMultiply",
	"NumpadAdd",
	"NumpadSubtract",
	"NumpadDivide",
}

// GetHotkeyLabel returns the localized display label for a hotkey name.
func GetHotkeyLabel(name string) string {
	return T("hotkey." + name)
}

// hotkeyToVK maps key names to virtual key codes
var hotkeyToVK = map[string]uintptr{
	"PrintScreen":     VK_SNAPSHOT,
	"F2":              VK_F2,
	"F3":              VK_F3,
	"F4":              VK_F4,
	"F5":              VK_F5,
	"F6":              VK_F6,
	"F7":              VK_F7,
	"F8":              VK_F8,
	"F9":              VK_F9,
	"F10":             VK_F10,
	"F11":             VK_F11,
	"F12":             VK_F12,
	"ScrollLock":      VK_SCROLL,
	"Pause":           VK_PAUSE,
	"PageUp":          VK_PRIOR,
	"PageDown":        VK_NEXT,
	"Home":            VK_HOME,
	"End":             VK_END,
	"Insert":          VK_INSERT,
	"Delete":          VK_DELETE,
	"Numpad0":         VK_NUMPAD0,
	"Numpad1":         VK_NUMPAD1,
	"Numpad2":         VK_NUMPAD2,
	"Numpad3":         VK_NUMPAD3,
	"Numpad4":         VK_NUMPAD4,
	"Numpad5":         VK_NUMPAD5,
	"Numpad6":         VK_NUMPAD6,
	"Numpad7":         VK_NUMPAD7,
	"Numpad8":         VK_NUMPAD8,
	"Numpad9":         VK_NUMPAD9,
	"NumpadMultiply":  VK_MULTIPLY,
	"NumpadAdd":       VK_ADD,
	"NumpadSubtract":  VK_SUBTRACT,
	"NumpadDivide":    VK_DIVIDE,
}

// snippingToolRegistryKey is the registry path for the PrintScreen → Snipping Tool setting.
const snippingToolRegistryKey = `Control Panel\Keyboard`
const snippingToolValueName = "PrintScreenKeyForSnippingEnabled"

// snippingToolDisabled tracks whether we disabled the Snipping Tool so we can restore it.
var snippingToolDisabled bool

// disableSnippingToolPrintScreen sets the registry value to prevent Windows 11
// from hijacking PrintScreen for the Snipping Tool.
func disableSnippingToolPrintScreen() {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, snippingToolRegistryKey, registry.SET_VALUE)
	if err != nil {
		debugLog("[HOTKEY] Failed to open registry for Snipping Tool: %v\n", err)
		return
	}
	defer key.Close()

	if err := key.SetDWordValue(snippingToolValueName, 0); err != nil {
		debugLog("[HOTKEY] Failed to disable Snipping Tool PrintScreen: %v\n", err)
		return
	}
	snippingToolDisabled = true
	debugLn("[HOTKEY] Disabled Snipping Tool PrintScreen via registry")
}

// restoreSnippingToolPrintScreen re-enables the Snipping Tool PrintScreen mapping.
func restoreSnippingToolPrintScreen() {
	if !snippingToolDisabled {
		return
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, snippingToolRegistryKey, registry.SET_VALUE)
	if err != nil {
		return
	}
	defer key.Close()

	key.SetDWordValue(snippingToolValueName, 1)
	snippingToolDisabled = false
	debugLn("[HOTKEY] Restored Snipping Tool PrintScreen via registry")
}

// SetHotkey updates the capture hotkey. When PrintScreen is selected,
// the Windows 11 Snipping Tool is automatically disabled via registry.
func SetHotkey(keyName string) {
	if vk, ok := hotkeyToVK[keyName]; ok {
		// Manage Snipping Tool registry based on hotkey
		if vk == VK_SNAPSHOT {
			disableSnippingToolPrintScreen()
		} else {
			restoreSnippingToolPrintScreen()
		}
		currentHotkey = vk
		registerGlobalHotkey(vk)
		debugLog("[HOTKEY] Set capture key to: %s\n", keyName)
	}
}

// WatchHotkey polls for configured hotkey press
func WatchHotkey() {
	keyName := GetHotkeyName(currentHotkey)
	fmt.Printf("[READY] Press %s to capture screenshots\n", keyName)

	var lastState int16 = 0

	for watching.Load() {
		time.Sleep(50 * time.Millisecond) // Poll every 50ms

		// GetAsyncKeyState returns key state
		// High bit (0x8000) = key is currently down
		// Low bit (0x0001) = key was pressed since last call
		ret, _, _ := procGetAsyncKeyState.Call(currentHotkey)
		state := int16(ret)

		// Detect key press (transition from not pressed to pressed)
		// High bit set = negative in int16, so state < 0 means key is down
		if state < 0 && lastState >= 0 {
			fmt.Println("[SCREENSHOT] Captured!")
			go captureScreen()
		}

		lastState = state
	}

	debugLn("[HOTKEY] Watcher stopped!")
}

// GetHotkeyName returns the display name for a virtual key code
func GetHotkeyName(vk uintptr) string {
	for name, code := range hotkeyToVK {
		if code == vk {
			return GetHotkeyLabel(name)
		}
	}
	return fmt.Sprintf("Unknown (0x%02X)", vk)
}

// captureScreen takes a screenshot of the display where Tarkov is running
// and adds it to the batch. Uses the screenshot library directly instead
// of the clipboard.
func captureScreen() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[CAPTURE] PANIC recovered: %v\n", r) // Always log panics
		}
	}()

	// Check if Tarkov is running
	if !IsTarkovRunning() {
		debugLn("[CAPTURE] Tarkov not running, ignoring")
		return
	}

	// Check if Tarkov is the foreground window — skip if user is alt-tabbed
	// to Twitch, YouTube, browser, etc. (otherwise we'd capture random screens)
	if !IsTarkovForeground() {
		debugLn("[CAPTURE] Tarkov is running but not foreground, ignoring")
		return
	}

	// Check if image viewer is running (re-capture prevention)
	// Skip in admin/debug builds so developer can test with image viewers open
	if !debugMode {
		if isViewer, viewerName := IsImageViewerRunning(); isViewer {
			fmt.Printf("[CAPTURE] Image viewer detected: %s - blocking to prevent re-capture\n", viewerName) // User-facing warning
			showWarning(T("capture.blocked"), fmt.Sprintf(T("capture.closeviewer"), viewerName))
			return
		}
	}

	// Get the display where Tarkov is running
	n := screenshot.NumActiveDisplays()
	if n == 0 {
		fmt.Println("[CAPTURE] No displays found") // User-facing error
		return
	}

	// Find which display Tarkov is on
	displayIndex := GetTarkovDisplayIndex()
	if displayIndex < 0 || displayIndex >= n {
		displayIndex = 0 // Fallback to primary
	}

	bounds := screenshot.GetDisplayBounds(displayIndex)
	debugLog("[CAPTURE] Capturing display %d: %dx%d\n", displayIndex, bounds.Dx(), bounds.Dy())

	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		fmt.Printf("[CAPTURE] Failed to capture: %v\n", err) // User-facing error
		return
	}

	debugLog("[CAPTURE] Got image %dx%d\n", img.Bounds().Dx(), img.Bounds().Dy())

	// Debug: Save raw capture for inspection
	debugSaveScreenshot(img, "raw")

	// Add to batch for processing
	addToBatch(img)
}

// addToBatch validates the captured image (size, aspect ratio, re-capture),
// compresses it to JPEG immediately, and stores only the compressed bytes
// in the batch. The raw image (~15 MB for 2K) is freed after compression.
func addToBatch(img *image.RGBA) {
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()

	// Check minimum size
	if width < 800 || height < 400 {
		debugLog("[CAPTURE] Image too small (%dx%d)\n", width, height)
		return
	}

	// Check for re-capture (screenshot of a screenshot with our signature)
	if HasSignature(img) {
		fmt.Println("[CAPTURE] Re-capture detected! Ignoring...") // User-facing warning
		showWarning(T("capture.recapture"), T("capture.recapture.msg"))
		return
	}

	// Embed our signature in-place (no copy needed)
	EmbedSignature(img)

	// Check aspect ratio
	aspectRatio := float64(width) / float64(height)
	if !isValidAspectRatio(aspectRatio) {
		debugLog("[CAPTURE] Invalid aspect ratio (%.2f)\n", aspectRatio)
		return
	}

	// Compress to JPEG immediately — raw image can be GC'd after this
	jpegData, err := compressImage(img, config)
	img = nil
	if err != nil {
		fmt.Printf("[CAPTURE] Failed to compress: %v\n", err)
		return
	}
	debugLog("[CAPTURE] Compressed %dx%d → %d KB\n", width, height, len(jpegData)/1024)

	// Add compressed bytes to batch
	batchMutex.Lock()

	if batchUploading {
		if len(pendingImages) >= maxBatchSize {
			fmt.Printf("[PENDING] Limit reached (%d), ignoring screenshot\n", maxBatchSize)
			batchMutex.Unlock()
			PlayCaptureSound()
			showWarning(T("screenshot.limit"), fmt.Sprintf(T("screenshot.limit.msg"), maxBatchSize))
			return
		}
		pendingImages = append(pendingImages, jpegData)
		count := len(pendingImages)
		debugLog("[PENDING] Image %d added to pending queue\n", count)
		batchMutex.Unlock()
		triggerCaptureFeedback(count)
		return
	}

	if len(batchImages) >= maxBatchSize {
		fmt.Printf("[BATCH] Limit reached (%d), ignoring screenshot\n", maxBatchSize)
		batchMutex.Unlock()
		PlayCaptureSound()
		showWarning(T("screenshot.limit"), fmt.Sprintf(T("screenshot.limit.msg"), maxBatchSize))
		return
	}

	batchImages = append(batchImages, jpegData)
	count := len(batchImages)
	fmt.Printf("[BATCH] Screenshot %d/%d added, waiting 20s...\n", count, maxBatchSize)

	// At max: start upload immediately instead of waiting
	if count >= maxBatchSize {
		fmt.Println("[BATCH] Max reached, processing immediately...")
		if batchTimer != nil {
			batchTimer.Stop()
		}
		batchTimer = time.AfterFunc(1*time.Second, processBatch)
		batchMutex.Unlock()
		triggerCaptureFeedback(count)
		return
	}

	// Reset timer
	if batchTimer != nil {
		batchTimer.Stop()
	}
	batchTimer = time.AfterFunc(batchWaitTime, processBatch)

	batchMutex.Unlock()
	triggerCaptureFeedback(count)
}

// triggerCaptureFeedback fires the configured feedback mechanisms (flash,
// sound, overlay) and falls back to a balloon notification if overlay is off.
func triggerCaptureFeedback(count int) {
	if config == nil {
		return
	}
	if config.Feedback.FlashEnabled {
		ShowFlash()
	}
	if config.Feedback.SoundEnabled {
		PlayCaptureSound()
	}
	if config.Feedback.OverlayEnabled {
		ShowOverlay(count)
	}
	if !config.Feedback.OverlayEnabled {
		// Fallback: balloon notification
		if count == 1 {
			showBalloon(T("screenshot.captured"), T("screenshot.waiting"))
		} else {
			showBalloon(T("screenshot.captured"), fmt.Sprintf(T("screenshot.batch"), count))
		}
	}
}
