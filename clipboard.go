// clipboard.go - Windows Clipboard API wrapper
//
// This file provides functions to read images from the Windows clipboard.
// It uses standard Windows API calls (user32.dll, kernel32.dll) to:
// - Detect clipboard changes via sequence number
// - Check if clipboard contains image data
// - Extract image data in DIB (Device Independent Bitmap) format
//
// IMPORTANT: This code only READS from the clipboard. It does not:
// - Write to the clipboard
// - Interact with any application
// - Access any process memory
//
// The clipboard is a shared Windows resource that any application can read.
// When a user presses PrintScreen or Win+Shift+S, Windows places the
// screenshot in the clipboard, which we then read.
package main

import (
	"fmt"
	"image"
	"time"
	"unsafe"
)

var (
	procOpenClipboard              = user32.NewProc("OpenClipboard")
	procCloseClipboard             = user32.NewProc("CloseClipboard")
	procGetClipboardData           = user32.NewProc("GetClipboardData")
	procIsClipboardFormatAvailable = user32.NewProc("IsClipboardFormatAvailable")
	procGetClipboardSequenceNumber = user32.NewProc("GetClipboardSequenceNumber")

	procGlobalLock   = kernel32.NewProc("GlobalLock")
	procGlobalUnlock = kernel32.NewProc("GlobalUnlock")
	procGlobalSize   = kernel32.NewProc("GlobalSize")
)

// GetClipboardSequenceNumber returns a number that changes when clipboard changes
func GetClipboardSequenceNumber() uint32 {
	ret, _, _ := procGetClipboardSequenceNumber.Call()
	return uint32(ret)
}

const (
	CF_BITMAP = 2
	CF_DIB    = 8
	CF_DIBV5  = 17
)

type BITMAPINFOHEADER struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

// HasClipboardImage checks if there's an image in clipboard
func HasClipboardImage() bool {
	ret, _, _ := procIsClipboardFormatAvailable.Call(CF_DIB)
	return ret != 0
}

// clipboardData holds raw data copied from clipboard
type clipboardData struct {
	width    int
	height   int
	bitCount int
	rowSize  int
	rawData  []byte
}

// copyClipboardData quickly copies raw bytes from clipboard (minimizes lock time)
// Tries CF_DIBV5 first, then CF_DIB as fallback
func copyClipboardData() *clipboardData {
	ret, _, _ := procOpenClipboard.Call(0)
	if ret == 0 {
		fmt.Println("[CLIPBOARD] Failed to open")
		return nil
	}
	// IMPORTANT: Close clipboard as soon as possible!
	defer func() {
		procCloseClipboard.Call()
		fmt.Println("[CLIPBOARD] Closed")
	}()

	// Try formats in order of preference: DIBV5 (modern), DIB (legacy)
	var hMem uintptr
	var formatUsed string

	// Try CF_DIBV5 first (modern 32-bit format)
	hasDIBV5, _, _ := procIsClipboardFormatAvailable.Call(CF_DIBV5)
	if hasDIBV5 != 0 {
		hMem, _, _ = procGetClipboardData.Call(CF_DIBV5)
		if hMem != 0 {
			formatUsed = "CF_DIBV5"
		}
	}

	// Fallback to CF_DIB
	if hMem == 0 {
		hasDIB, _, _ := procIsClipboardFormatAvailable.Call(CF_DIB)
		if hasDIB != 0 {
			hMem, _, _ = procGetClipboardData.Call(CF_DIB)
			if hMem != 0 {
				formatUsed = "CF_DIB"
			}
		}
	}

	if hMem == 0 {
		fmt.Println("[CLIPBOARD] No image format available")
		return nil
	}

	fmt.Printf("[CLIPBOARD] Using format: %s\n", formatUsed)

	ptr, _, _ := procGlobalLock.Call(hMem)
	if ptr == 0 {
		fmt.Println("[CLIPBOARD] GlobalLock failed")
		return nil
	}
	defer procGlobalUnlock.Call(hMem)

	// Get size of memory block
	size, _, _ := procGlobalSize.Call(hMem)
	if size == 0 {
		fmt.Println("[CLIPBOARD] GlobalSize returned 0")
		return nil
	}

	// Parse DIB header
	header := (*BITMAPINFOHEADER)(unsafe.Pointer(ptr))
	width := int(header.Width)
	height := int(header.Height)
	if height < 0 {
		height = -height
	}

	bitCount := int(header.BitCount)
	if bitCount != 24 && bitCount != 32 {
		fmt.Printf("[CLIPBOARD] Unsupported bitCount: %d\n", bitCount)
		return nil
	}

	headerSize := int(header.Size)
	rowSize := ((width*bitCount + 31) / 32) * 4
	dataSize := rowSize * height

	// Quick copy of raw bytes - this is fast!
	rawData := make([]byte, dataSize)
	dataPtr := ptr + uintptr(headerSize)

	// Use copy via unsafe pointer - much faster than pixel-by-pixel
	for i := 0; i < dataSize; i++ {
		rawData[i] = *(*byte)(unsafe.Pointer(dataPtr + uintptr(i)))
	}

	return &clipboardData{
		width:    width,
		height:   height,
		bitCount: bitCount,
		rowSize:  rowSize,
		rawData:  rawData,
	}
}

// GetClipboardImage retrieves image from clipboard
// This function minimizes clipboard lock time by:
// 1. Quickly copying raw bytes while holding the lock
// 2. Releasing the lock immediately
// 3. Converting pixels AFTER the lock is released
// Retries up to 5 times with delays (helps after text copy/paste)
func GetClipboardImage() (*image.RGBA, error) {
	var data *clipboardData

	// Retry loop - sometimes clipboard isn't ready immediately after text operations
	for attempt := 0; attempt < 5; attempt++ {
		data = copyClipboardData()
		if data != nil {
			if attempt > 0 {
				fmt.Printf("[CLIPBOARD] Got data on attempt %d\n", attempt+1)
			}
			break
		}
		if attempt < 4 {
			fmt.Printf("[CLIPBOARD] Retry %d/5...\n", attempt+1)
			time.Sleep(100 * time.Millisecond)
		}
	}

	if data == nil {
		return nil, nil
	}

	// Step 2: Slow pixel conversion (NO clipboard lock!)
	// This can take 50-200ms but doesn't block other apps
	img := image.NewRGBA(image.Rect(0, 0, data.width, data.height))

	for y := 0; y < data.height; y++ {
		srcY := data.height - 1 - y // DIB is bottom-up
		srcRowOffset := srcY * data.rowSize

		for x := 0; x < data.width; x++ {
			var r, g, b, a byte
			if data.bitCount == 32 {
				pixOffset := srcRowOffset + x*4
				b = data.rawData[pixOffset]
				g = data.rawData[pixOffset+1]
				r = data.rawData[pixOffset+2]
				a = 255
			} else {
				pixOffset := srcRowOffset + x*3
				b = data.rawData[pixOffset]
				g = data.rawData[pixOffset+1]
				r = data.rawData[pixOffset+2]
				a = 255
			}

			idx := (y*data.width + x) * 4
			img.Pix[idx] = r
			img.Pix[idx+1] = g
			img.Pix[idx+2] = b
			img.Pix[idx+3] = a
		}
	}

	return img, nil
}
