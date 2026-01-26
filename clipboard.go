package main

import (
	"image"
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
)

// GetClipboardSequenceNumber returns a number that changes when clipboard changes
func GetClipboardSequenceNumber() uint32 {
	ret, _, _ := procGetClipboardSequenceNumber.Call()
	return uint32(ret)
}

const (
	CF_DIB = 8
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

// GetClipboardImage retrieves image from clipboard
func GetClipboardImage() (*image.RGBA, error) {
	ret, _, _ := procOpenClipboard.Call(0)
	if ret == 0 {
		return nil, nil
	}
	defer procCloseClipboard.Call()

	hMem, _, _ := procGetClipboardData.Call(CF_DIB)
	if hMem == 0 {
		return nil, nil
	}

	ptr, _, _ := procGlobalLock.Call(hMem)
	if ptr == 0 {
		return nil, nil
	}
	defer procGlobalUnlock.Call(hMem)

	// Parse DIB header
	header := (*BITMAPINFOHEADER)(unsafe.Pointer(ptr))
	width := int(header.Width)
	height := int(header.Height)
	if height < 0 {
		height = -height
	}

	bitCount := header.BitCount
	if bitCount != 24 && bitCount != 32 {
		return nil, nil
	}

	// Calculate data offset and row size
	headerSize := int(header.Size)
	rowSize := ((width*int(bitCount) + 31) / 32) * 4
	dataPtr := ptr + uintptr(headerSize)

	// Create image
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Copy pixels (DIB is bottom-up)
	for y := 0; y < height; y++ {
		srcY := height - 1 - y
		srcRow := dataPtr + uintptr(srcY*rowSize)

		for x := 0; x < width; x++ {
			var r, g, b, a byte
			if bitCount == 32 {
				pixPtr := srcRow + uintptr(x*4)
				b = *(*byte)(unsafe.Pointer(pixPtr))
				g = *(*byte)(unsafe.Pointer(pixPtr + 1))
				r = *(*byte)(unsafe.Pointer(pixPtr + 2))
				a = 255
			} else {
				pixPtr := srcRow + uintptr(x*3)
				b = *(*byte)(unsafe.Pointer(pixPtr))
				g = *(*byte)(unsafe.Pointer(pixPtr + 1))
				r = *(*byte)(unsafe.Pointer(pixPtr + 2))
				a = 255
			}

			idx := (y*width + x) * 4
			img.Pix[idx] = r
			img.Pix[idx+1] = g
			img.Pix[idx+2] = b
			img.Pix[idx+3] = a
		}
	}

	return img, nil
}
