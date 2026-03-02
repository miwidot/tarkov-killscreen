// splash.go - Startup Splash Screen
//
// Displays the application logo as a centered popup window on startup.
// The splash is shown for 750ms and can be dismissed by clicking.
// Uses the embedded logo.png (pre-scaled to 50%), displayed at 80%.
// The DIB is created once and cached for efficient WM_PAINT redraws.
package main

import (
	"embed"
	"image"
	"image/png"
	"syscall"
	"time"
	"unsafe"

	"github.com/lxn/win"
)

//go:embed logo.png
var logoFS embed.FS

var splashHwnd win.HWND
var splashWidth, splashHeight int

// Cached DIB resources (created once, reused across WM_PAINT calls)
var splashMemDC win.HDC
var splashBitmap win.HBITMAP
var splashSrcWidth, splashSrcHeight int

// ShowSplash displays the logo as a centered, borderless popup window.
// The splash auto-closes after 750ms or on click.
func ShowSplash() {
	// Load logo
	f, err := logoFS.Open("logo.png")
	if err != nil {
		return
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		return
	}

	bounds := img.Bounds()
	splashSrcWidth = bounds.Dx()
	splashSrcHeight = bounds.Dy()

	// Logo is pre-scaled to 50%, display at 80% = same visual size as old 40%
	splashWidth = splashSrcWidth * 80 / 100
	splashHeight = splashSrcHeight * 80 / 100

	// Create cached DIB from image (done once, before window creation)
	createSplashDIB(img)

	// Get screen size
	screenWidth := int(win.GetSystemMetrics(win.SM_CXSCREEN))
	screenHeight := int(win.GetSystemMetrics(win.SM_CYSCREEN))
	x := (screenWidth - splashWidth) / 2
	y := (screenHeight - splashHeight) / 2

	// Create popup window class
	className, _ := syscall.UTF16PtrFromString("TarkovSplash")
	windowName, _ := syscall.UTF16PtrFromString("")

	wndClass := win.WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(win.WNDCLASSEX{})),
		LpfnWndProc:   syscall.NewCallback(splashWndProc),
		HInstance:     win.GetModuleHandle(nil),
		LpszClassName: className,
		HbrBackground: win.HBRUSH(win.GetStockObject(win.BLACK_BRUSH)),
	}
	win.RegisterClassEx(&wndClass)

	// WS_POPUP | WS_VISIBLE
	style := uint32(0x80000000 | 0x10000000)

	hwnd := win.CreateWindowEx(
		win.WS_EX_TOPMOST|win.WS_EX_TOOLWINDOW,
		className,
		windowName,
		style,
		int32(x), int32(y), int32(splashWidth), int32(splashHeight),
		0, 0, win.GetModuleHandle(nil), nil,
	)

	if hwnd == 0 {
		destroySplashDIB()
		return
	}

	splashHwnd = hwnd

	win.ShowWindow(hwnd, win.SW_SHOW)
	win.UpdateWindow(hwnd)

	// Timer to close
	go func() {
		time.Sleep(750 * time.Millisecond)
		if splashHwnd != 0 {
			win.PostMessage(splashHwnd, win.WM_CLOSE, 0, 0)
		}
	}()

	// Run message loop until window closes
	var msg win.MSG
	for splashHwnd != 0 {
		if win.PeekMessage(&msg, 0, 0, 0, win.PM_REMOVE) {
			win.TranslateMessage(&msg)
			win.DispatchMessage(&msg)
		} else {
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// createSplashDIB creates a memory DC + DIB section from the decoded image.
// This is done once and cached so WM_PAINT just does a single StretchBlt.
func createSplashDIB(img image.Image) {
	screenDC := win.GetDC(0)
	splashMemDC = win.CreateCompatibleDC(screenDC)
	win.ReleaseDC(0, screenDC)

	bmi := win.BITMAPINFOHEADER{
		BiSize:        uint32(unsafe.Sizeof(win.BITMAPINFOHEADER{})),
		BiWidth:       int32(splashSrcWidth),
		BiHeight:      -int32(splashSrcHeight), // Top-down
		BiPlanes:      1,
		BiBitCount:    32,
		BiCompression: win.BI_RGB,
	}

	var bits unsafe.Pointer
	splashBitmap = win.CreateDIBSection(splashMemDC, &bmi, win.DIB_RGB_COLORS, &bits, 0, 0)
	if splashBitmap == 0 || bits == nil {
		return
	}

	win.SelectObject(splashMemDC, win.HGDIOBJ(splashBitmap))

	// Copy pixels — convert RGBA → BGRA
	bounds := img.Bounds()
	stride := splashSrcWidth * 4
	pixels := unsafe.Slice((*byte)(bits), splashSrcHeight*stride)

	// Try fast path for *image.RGBA / *image.NRGBA
	switch src := img.(type) {
	case *image.RGBA:
		for y := 0; y < splashSrcHeight; y++ {
			srcRow := src.Pix[y*src.Stride : y*src.Stride+splashSrcWidth*4]
			dstRow := pixels[y*stride : y*stride+splashSrcWidth*4]
			copy(dstRow, srcRow)
			// Swap R↔B in-place (RGBA → BGRA)
			for i := 0; i < len(dstRow); i += 4 {
				dstRow[i], dstRow[i+2] = dstRow[i+2], dstRow[i]
			}
		}
	case *image.NRGBA:
		for y := 0; y < splashSrcHeight; y++ {
			srcRow := src.Pix[y*src.Stride : y*src.Stride+splashSrcWidth*4]
			dstRow := pixels[y*stride : y*stride+splashSrcWidth*4]
			copy(dstRow, srcRow)
			for i := 0; i < len(dstRow); i += 4 {
				dstRow[i], dstRow[i+2] = dstRow[i+2], dstRow[i]
			}
		}
	default:
		// Generic fallback (per-pixel)
		for y := 0; y < splashSrcHeight; y++ {
			for x := 0; x < splashSrcWidth; x++ {
				r, g, b, a := img.At(x+bounds.Min.X, y+bounds.Min.Y).RGBA()
				idx := y*stride + x*4
				pixels[idx] = byte(b >> 8)
				pixels[idx+1] = byte(g >> 8)
				pixels[idx+2] = byte(r >> 8)
				pixels[idx+3] = byte(a >> 8)
			}
		}
	}
}

// destroySplashDIB frees cached GDI resources.
func destroySplashDIB() {
	if splashBitmap != 0 {
		win.DeleteObject(win.HGDIOBJ(splashBitmap))
		splashBitmap = 0
	}
	if splashMemDC != 0 {
		win.DeleteDC(splashMemDC)
		splashMemDC = 0
	}
}

func splashWndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_PAINT:
		if splashMemDC != 0 {
			var ps win.PAINTSTRUCT
			hdc := win.BeginPaint(hwnd, &ps)
			win.SetStretchBltMode(hdc, win.HALFTONE)
			win.StretchBlt(hdc, 0, 0, int32(splashWidth), int32(splashHeight),
				splashMemDC, 0, 0, int32(splashSrcWidth), int32(splashSrcHeight), win.SRCCOPY)
			win.EndPaint(hwnd, &ps)
		}
		return 0

	case win.WM_LBUTTONDOWN, win.WM_CLOSE:
		win.DestroyWindow(hwnd)
		return 0

	case win.WM_DESTROY:
		splashHwnd = 0
		destroySplashDIB()
		return 0
	}

	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}
