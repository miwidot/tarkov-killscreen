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

var splashImg image.Image
var splashHwnd win.HWND
var splashWidth, splashHeight int
var splashDone chan struct{}

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
	splashImg = img

	bounds := img.Bounds()
	// Scale down to 40%
	width := bounds.Dx() * 40 / 100
	height := bounds.Dy() * 40 / 100
	splashWidth = width
	splashHeight = height

	// Get screen size
	screenWidth := int(win.GetSystemMetrics(win.SM_CXSCREEN))
	screenHeight := int(win.GetSystemMetrics(win.SM_CYSCREEN))
	x := (screenWidth - width) / 2
	y := (screenHeight - height) / 2

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
		int32(x), int32(y), int32(width), int32(height),
		0, 0, win.GetModuleHandle(nil), nil,
	)

	if hwnd == 0 {
		return
	}

	splashHwnd = hwnd
	splashDone = make(chan struct{})

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

func splashWndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_PAINT:
		if splashImg != nil {
			var ps win.PAINTSTRUCT
			hdc := win.BeginPaint(hwnd, &ps)

			bounds := splashImg.Bounds()
			srcWidth := bounds.Dx()
			srcHeight := bounds.Dy()

			// Create DIB with original size
			bmi := win.BITMAPINFOHEADER{
				BiSize:        uint32(unsafe.Sizeof(win.BITMAPINFOHEADER{})),
				BiWidth:       int32(srcWidth),
				BiHeight:      -int32(srcHeight), // Top-down
				BiPlanes:      1,
				BiBitCount:    32,
				BiCompression: win.BI_RGB,
			}

			var bits unsafe.Pointer
			memDC := win.CreateCompatibleDC(hdc)
			hBitmap := win.CreateDIBSection(memDC, &bmi, win.DIB_RGB_COLORS, &bits, 0, 0)

			if hBitmap != 0 && bits != nil {
				// Copy pixels
				pixels := (*[1 << 26]byte)(bits)
				for y := 0; y < srcHeight; y++ {
					for x := 0; x < srcWidth; x++ {
						r, g, b, a := splashImg.At(x+bounds.Min.X, y+bounds.Min.Y).RGBA()
						idx := (y*srcWidth + x) * 4
						// BGRA format
						pixels[idx] = byte(b >> 8)
						pixels[idx+1] = byte(g >> 8)
						pixels[idx+2] = byte(r >> 8)
						pixels[idx+3] = byte(a >> 8)
					}
				}

				oldBmp := win.SelectObject(memDC, win.HGDIOBJ(hBitmap))
				// Use StretchBlt to scale
				win.SetStretchBltMode(hdc, win.HALFTONE)
				win.StretchBlt(hdc, 0, 0, int32(splashWidth), int32(splashHeight),
					memDC, 0, 0, int32(srcWidth), int32(srcHeight), win.SRCCOPY)
				win.SelectObject(memDC, oldBmp)
				win.DeleteObject(win.HGDIOBJ(hBitmap))
			}

			win.DeleteDC(memDC)
			win.EndPaint(hwnd, &ps)
		}
		return 0

	case win.WM_LBUTTONDOWN, win.WM_CLOSE:
		// Click or timer to close
		win.DestroyWindow(hwnd)
		return 0

	case win.WM_DESTROY:
		splashHwnd = 0
		return 0
	}

	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}
