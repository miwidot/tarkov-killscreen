// overlay.go - Mini Overlay Popup
//
// Displays a small notification popup (320x80) at the bottom-right
// corner of the primary screen. Shows capture count and countdown timer.
// Updates in-place when multiple screenshots are taken quickly.
// Auto-closes after 3 seconds or on click.
// Adapts to the Windows light/dark theme automatically.
package main

import (
	"fmt"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/lxn/win"
	"golang.org/x/sys/windows/registry"
)

var (
	gdi32                = syscall.NewLazyDLL("gdi32.dll")
	procCreateSolidBrush = gdi32.NewProc("CreateSolidBrush")
	procInvalidateRect   = user32.NewProc("InvalidateRect")
	procFillRect         = user32.NewProc("FillRect")
)

const (
	overlayWidth  = 320
	overlayHeight = 80
	overlayMargin = 20
)

// overlayTheme holds the resolved colors for the current Windows theme.
type overlayTheme struct {
	bgR, bgG, bgB       byte
	textColor            win.COLORREF
	textColorDim         win.COLORREF
}

var (
	darkTheme = overlayTheme{
		bgR: 30, bgG: 30, bgB: 30,
		textColor:    win.COLORREF(0x00FFFFFF), // White
		textColorDim: win.COLORREF(0x00AAAAAA), // Light gray
	}
	lightTheme = overlayTheme{
		bgR: 243, bgG: 243, bgB: 243,
		textColor:    win.COLORREF(0x001E1E1E), // Near-black
		textColorDim: win.COLORREF(0x00666666), // Dark gray
	}
)

// isWindowsDarkMode reads the Windows theme from the registry.
func isWindowsDarkMode() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`,
		registry.QUERY_VALUE)
	if err != nil {
		return true // Default to dark
	}
	defer key.Close()

	val, _, err := key.GetIntegerValue("AppsUseLightTheme")
	if err != nil {
		return true
	}
	return val == 0 // 0 = dark, 1 = light
}

func getOverlayTheme() overlayTheme {
	if isWindowsDarkMode() {
		return darkTheme
	}
	return lightTheme
}

var (
	overlayOnce     sync.Once
	overlayCallback uintptr
	overlayMutex    sync.Mutex
	overlayHwnd     win.HWND
	overlayCount    int
	overlayTimer    *time.Timer
	overlayLine1    string
	overlayLine2    string
	overlayBrush    win.HBRUSH
	overlayCurrentTheme overlayTheme
)

func initOverlay() {
	overlayCallback = syscall.NewCallback(overlayWndProc)
	className, _ := syscall.UTF16PtrFromString("TarkovOverlay")
	wndClass := win.WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(win.WNDCLASSEX{})),
		LpfnWndProc:   overlayCallback,
		HInstance:     win.GetModuleHandle(nil),
		LpszClassName: className,
		HbrBackground: 0, // No class brush — handled per-instance in WM_ERASEBKGND
	}
	win.RegisterClassEx(&wndClass)
}

// ShowOverlay displays or updates the capture overlay popup.
// count is the number of screenshots in the current batch.
func ShowOverlay(count int) {
	overlayOnce.Do(initOverlay)

	overlayMutex.Lock()
	defer overlayMutex.Unlock()

	overlayCount = count

	// Build display text
	if count <= 1 {
		overlayLine1 = T("overlay.single")
	} else {
		overlayLine1 = fmt.Sprintf(T("overlay.captured"), count)
	}
	overlayLine2 = fmt.Sprintf(T("overlay.waiting"), 20)

	if overlayHwnd != 0 {
		// Update existing overlay: refresh text + reset timer
		procInvalidateRect.Call(uintptr(overlayHwnd), 0, 1)
		resetOverlayTimer()
		return
	}

	// Launch new overlay window
	go overlayWorker()
}

// HideOverlay closes the overlay if visible.
func HideOverlay() {
	overlayMutex.Lock()
	defer overlayMutex.Unlock()

	if overlayHwnd != 0 {
		win.PostMessage(overlayHwnd, win.WM_CLOSE, 0, 0)
	}
}

func resetOverlayTimer() {
	if overlayTimer != nil {
		overlayTimer.Stop()
	}
	overlayTimer = time.AfterFunc(3*time.Second, func() {
		overlayMutex.Lock()
		hwnd := overlayHwnd
		overlayMutex.Unlock()
		if hwnd != 0 {
			win.PostMessage(hwnd, win.WM_CLOSE, 0, 0)
		}
	})
}

func overlayWorker() {
	// Detect theme at creation time
	theme := getOverlayTheme()

	overlayMutex.Lock()
	overlayCurrentTheme = theme
	overlayMutex.Unlock()

	// Create background brush matching Windows theme
	colorRef := uintptr(uint32(theme.bgR) | uint32(theme.bgG)<<8 | uint32(theme.bgB)<<16)
	hBrush, _, _ := procCreateSolidBrush.Call(colorRef)
	overlayBrush = win.HBRUSH(hBrush)

	className, _ := syscall.UTF16PtrFromString("TarkovOverlay")
	windowName, _ := syscall.UTF16PtrFromString("")

	// Position: bottom-right of primary screen
	screenW := int32(win.GetSystemMetrics(win.SM_CXSCREEN))
	screenH := int32(win.GetSystemMetrics(win.SM_CYSCREEN))
	x := screenW - overlayWidth - overlayMargin
	y := screenH - overlayHeight - overlayMargin - 40 // extra offset for taskbar

	// WS_POPUP | WS_VISIBLE
	style := uint32(0x80000000 | 0x10000000)
	exStyle := win.WS_EX_TOPMOST | win.WS_EX_TOOLWINDOW | win.WS_EX_NOACTIVATE

	hwnd := win.CreateWindowEx(
		uint32(exStyle),
		className,
		windowName,
		style,
		x, y, overlayWidth, overlayHeight,
		0, 0, win.GetModuleHandle(nil), nil,
	)
	if hwnd == 0 {
		win.DeleteObject(win.HGDIOBJ(overlayBrush))
		return
	}

	overlayMutex.Lock()
	overlayHwnd = hwnd
	resetOverlayTimer()
	overlayMutex.Unlock()

	win.ShowWindow(hwnd, win.SW_SHOWNOACTIVATE)
	win.UpdateWindow(hwnd)

	// Message loop
	var msg win.MSG
	for {
		ret := win.GetMessage(&msg, 0, 0, 0)
		if ret == 0 || ret == -1 {
			break
		}
		win.TranslateMessage(&msg)
		win.DispatchMessage(&msg)
	}

	// Cleanup
	overlayMutex.Lock()
	overlayHwnd = 0
	if overlayTimer != nil {
		overlayTimer.Stop()
		overlayTimer = nil
	}
	overlayMutex.Unlock()

	win.DeleteObject(win.HGDIOBJ(overlayBrush))
}

func createOverlayFont(height int32, weight int32) win.HFONT {
	var lf win.LOGFONT
	lf.LfHeight = height
	lf.LfWeight = weight
	lf.LfCharSet = win.DEFAULT_CHARSET
	lf.LfQuality = win.CLEARTYPE_QUALITY
	face, _ := syscall.UTF16FromString("Segoe UI")
	copy(lf.LfFaceName[:], face)
	return win.CreateFontIndirect(&lf)
}

func overlayWndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_ERASEBKGND:
		// Paint background with current theme brush (not class brush)
		hdc := win.HDC(wParam)
		var rc win.RECT
		win.GetClientRect(hwnd, &rc)
		procFillRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(&rc)), uintptr(overlayBrush))
		return 1 // We handled it

	case win.WM_PAINT:
		var ps win.PAINTSTRUCT
		hdc := win.BeginPaint(hwnd, &ps)

		// Use theme colors
		overlayMutex.Lock()
		theme := overlayCurrentTheme
		line1 := overlayLine1
		line2 := overlayLine2
		overlayMutex.Unlock()

		win.SetTextColor(hdc, theme.textColor)
		win.SetBkMode(hdc, win.TRANSPARENT)

		// Bold font for line 1
		hFont := createOverlayFont(-16, 700)
		oldFont := win.SelectObject(hdc, win.HGDIOBJ(hFont))

		line1Ptr, _ := syscall.UTF16PtrFromString(line1)
		rect1 := win.RECT{Left: 15, Top: 12, Right: overlayWidth - 15, Bottom: 40}
		win.DrawTextEx(hdc, line1Ptr, -1, &rect1, 0, nil)

		// Normal font for line 2
		hFont2 := createOverlayFont(-13, 400)
		win.SelectObject(hdc, win.HGDIOBJ(hFont2))

		// Dimmer color for line 2
		win.SetTextColor(hdc, theme.textColorDim)

		line2Ptr, _ := syscall.UTF16PtrFromString(line2)
		rect2 := win.RECT{Left: 15, Top: 42, Right: overlayWidth - 15, Bottom: 70}
		win.DrawTextEx(hdc, line2Ptr, -1, &rect2, 0, nil)

		// Cleanup GDI
		win.SelectObject(hdc, oldFont)
		win.DeleteObject(win.HGDIOBJ(hFont))
		win.DeleteObject(win.HGDIOBJ(hFont2))

		win.EndPaint(hwnd, &ps)
		return 0

	case win.WM_LBUTTONDOWN:
		// Click to dismiss
		win.DestroyWindow(hwnd)
		return 0

	case win.WM_CLOSE:
		win.DestroyWindow(hwnd)
		return 0

	case win.WM_DESTROY:
		win.PostQuitMessage(0)
		return 0
	}

	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}
