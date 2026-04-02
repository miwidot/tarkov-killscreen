// flash.go - Screen Flash Effect
//
// Displays a brief semi-transparent white overlay across the entire virtual
// screen (all monitors) for 150ms to provide visual capture feedback.
// Uses WS_EX_NOACTIVATE to avoid stealing focus from fullscreen games.
package main

import (
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

	"github.com/lxn/win"
)

var (
	procSetLayeredWindowAttributes = user32.NewProc("SetLayeredWindowAttributes")
	procSetTimer                   = user32.NewProc("SetTimer")
)

const (
	smXVirtualScreen  = 76
	smYVirtualScreen  = 77
	smCXVirtualScreen = 78
	smCYVirtualScreen = 79
	lwaNAlpha         = 0x00000002
)

var (
	flashOnce     sync.Once
	flashCallback uintptr
	flashActive   int32 // atomic: 0 = idle, 1 = showing
)

func initFlash() {
	flashCallback = syscall.NewCallback(flashWndProc)
	className, _ := syscall.UTF16PtrFromString("TarkovFlash")
	wndClass := win.WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(win.WNDCLASSEX{})),
		LpfnWndProc:   flashCallback,
		HInstance:     win.GetModuleHandle(nil),
		LpszClassName: className,
		HbrBackground: win.HBRUSH(win.GetStockObject(win.WHITE_BRUSH)),
	}
	win.RegisterClassEx(&wndClass)
}

// ShowFlash displays a brief white flash across all monitors. Non-blocking.
// Only one flash at a time — rapid presses are ignored while active.
func ShowFlash() {
	flashOnce.Do(initFlash)

	// Skip if a flash is already showing
	if !atomic.CompareAndSwapInt32(&flashActive, 0, 1) {
		return
	}
	go flashWorker()
}

func flashWorker() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer atomic.StoreInt32(&flashActive, 0)

	className, _ := syscall.UTF16PtrFromString("TarkovFlash")
	windowName, _ := syscall.UTF16PtrFromString("")

	// Virtual screen metrics for multi-monitor
	x := int32(win.GetSystemMetrics(smXVirtualScreen))
	y := int32(win.GetSystemMetrics(smYVirtualScreen))
	w := int32(win.GetSystemMetrics(smCXVirtualScreen))
	h := int32(win.GetSystemMetrics(smCYVirtualScreen))

	// WS_POPUP | WS_VISIBLE
	style := uint32(0x80000000 | 0x10000000)
	exStyle := win.WS_EX_TOPMOST | win.WS_EX_TOOLWINDOW | win.WS_EX_LAYERED | win.WS_EX_NOACTIVATE | win.WS_EX_TRANSPARENT

	hwnd := win.CreateWindowEx(
		uint32(exStyle),
		className,
		windowName,
		style,
		x, y, w, h,
		0, 0, win.GetModuleHandle(nil), nil,
	)
	if hwnd == 0 {
		return
	}

	// Semi-transparent: alpha 180/255
	procSetLayeredWindowAttributes.Call(uintptr(hwnd), 0, 180, lwaNAlpha)

	win.ShowWindow(hwnd, win.SW_SHOWNOACTIVATE)
	win.UpdateWindow(hwnd)

	// Auto-close after 150ms via Windows timer (runs on same thread as message loop)
	procSetTimer.Call(uintptr(hwnd), 1, 150, 0)

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
}

func flashWndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_TIMER:
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
