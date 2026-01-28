// windows.go - Windows Process Enumeration
//
// This file provides functions to list running processes on Windows.
// It uses the standard Windows Toolhelp API (kernel32.dll) to enumerate
// all running processes and check if a specific process is running.
//
// IMPORTANT: This code only READS the list of process names. It does not:
// - Open any process handles with elevated permissions
// - Read or write process memory
// - Inject code into processes
// - Interact with any process in any way
//
// The Toolhelp API is a standard Windows API used by Task Manager and
// many other legitimate applications to list running processes.
// We use it solely to check if "EscapeFromTarkov.exe" is in the list,
// so we know whether to process screenshots or ignore them.
package main

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procCreateToolhelp32Snapshot = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32FirstW          = kernel32.NewProc("Process32FirstW")
	procProcess32NextW           = kernel32.NewProc("Process32NextW")
	procCloseHandle              = kernel32.NewProc("CloseHandle")
	procCreateMutexW             = kernel32.NewProc("CreateMutexW")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procIsIconic                 = user32.NewProc("IsIconic")
	procGetWindowRect            = user32.NewProc("GetWindowRect")
	procMonitorFromWindow        = user32.NewProc("MonitorFromWindow")
	procGetMonitorInfoW          = user32.NewProc("GetMonitorInfoW")
)

const (
	MONITOR_DEFAULTTONEAREST = 2
)

const (
	ERROR_ALREADY_EXISTS = 183
	TH32CS_SNAPPROCESS   = 0x00000002
	MAX_PATH             = 260
)

type PROCESSENTRY32W struct {
	Size              uint32
	CntUsage          uint32
	ProcessID         uint32
	DefaultHeapID     uintptr
	ModuleID          uint32
	CntThreads        uint32
	ParentProcessID   uint32
	PriorityClassBase int32
	Flags             uint32
	ExeFile           [MAX_PATH]uint16
}

type RECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type MONITORINFO struct {
	Size    uint32
	Monitor RECT
	Work    RECT
	Flags   uint32
}

// IsAlreadyRunning checks if another instance of the app is already running
// Uses a named mutex - if we can't create it because it exists, another instance is running
func IsAlreadyRunning() bool {
	mutexName, _ := syscall.UTF16PtrFromString("Global\\TarkovKillScreenAnalyzer_Mutex")
	handle, _, err := procCreateMutexW.Call(0, 1, uintptr(unsafe.Pointer(mutexName)))

	if handle == 0 {
		fmt.Println("[MUTEX] Failed to create mutex")
		return true // Failed to create mutex
	}

	// In Go syscall, the error code is in err (as syscall.Errno)
	if errno, ok := err.(syscall.Errno); ok && errno == ERROR_ALREADY_EXISTS {
		fmt.Println("[MUTEX] Another instance is already running")
		procCloseHandle.Call(handle)
		return true
	}

	fmt.Println("[MUTEX] This is the first instance")
	// Keep the mutex handle open - it will be released when the app exits
	return false
}

// IsTarkovRunning checks if EscapeFromTarkov.exe is currently running
func IsTarkovRunning() bool {
	snapshot, _, err := procCreateToolhelp32Snapshot.Call(TH32CS_SNAPPROCESS, 0)
	if snapshot == 0 || snapshot == ^uintptr(0) {
		fmt.Printf("[TARKOV] Snapshot failed: %v\n", err)
		return false
	}
	defer procCloseHandle.Call(snapshot)

	var entry PROCESSENTRY32W
	entry.Size = uint32(unsafe.Sizeof(entry))

	ret, _, _ := procProcess32FirstW.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	if ret == 0 {
		return false
	}

	for {
		exeName := syscall.UTF16ToString(entry.ExeFile[:])
		// Check for Tarkov - also check partial match in case of different naming
		if strings.EqualFold(exeName, "EscapeFromTarkov.exe") ||
			strings.Contains(strings.ToLower(exeName), "escapefromtarkov") {
			fmt.Printf("[TARKOV] Found: %s\n", exeName)
			return true
		}

		ret, _, _ = procProcess32NextW.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
		if ret == 0 {
			break
		}
	}

	return false
}

// GetTarkovDisplayIndex finds which display Tarkov is running on
// Returns the display index (0-based) or -1 if not found
func GetTarkovDisplayIndex() int {
	var tarkovHwnd uintptr
	var tarkovPid uint32

	// First, find the Tarkov process ID
	snapshot, _, _ := procCreateToolhelp32Snapshot.Call(TH32CS_SNAPPROCESS, 0)
	if snapshot == 0 || snapshot == ^uintptr(0) {
		return -1
	}

	var entry PROCESSENTRY32W
	entry.Size = uint32(unsafe.Sizeof(entry))

	ret, _, _ := procProcess32FirstW.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	for ret != 0 {
		exeName := strings.ToLower(syscall.UTF16ToString(entry.ExeFile[:]))
		if strings.Contains(exeName, "escapefromtarkov") {
			tarkovPid = entry.ProcessID
			break
		}
		ret, _, _ = procProcess32NextW.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	}
	procCloseHandle.Call(snapshot)

	if tarkovPid == 0 {
		return -1
	}

	// Find the window belonging to Tarkov
	callback := syscall.NewCallback(func(hwnd uintptr, lParam uintptr) uintptr {
		var pid uint32
		procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
		if pid == tarkovPid {
			visible, _, _ := procIsWindowVisible.Call(hwnd)
			if visible != 0 {
				tarkovHwnd = hwnd
				return 0 // Stop enumeration
			}
		}
		return 1 // Continue
	})
	procEnumWindows.Call(callback, 0)

	if tarkovHwnd == 0 {
		fmt.Println("[TARKOV] Window not found, using display 0")
		return 0
	}

	// Get window rect
	var rect RECT
	ret2, _, _ := procGetWindowRect.Call(tarkovHwnd, uintptr(unsafe.Pointer(&rect)))
	if ret2 == 0 {
		return 0
	}

	// Calculate window center
	centerX := (rect.Left + rect.Right) / 2
	centerY := (rect.Top + rect.Bottom) / 2

	fmt.Printf("[TARKOV] Window at (%d,%d)-(%d,%d), center: (%d,%d)\n",
		rect.Left, rect.Top, rect.Right, rect.Bottom, centerX, centerY)

	// Find which display contains this point
	// We'll use the screenshot library's display bounds
	return findDisplayAtPoint(int(centerX), int(centerY))
}

// findDisplayAtPoint returns the display index containing the given point
func findDisplayAtPoint(x, y int) int {
	n := getNumDisplays()
	for i := 0; i < n; i++ {
		bounds := getDisplayBounds(i)
		if x >= int(bounds.Left) && x < int(bounds.Right) && y >= int(bounds.Top) && y < int(bounds.Bottom) {
			fmt.Printf("[TARKOV] Found on display %d\n", i)
			return i
		}
	}
	return 0 // Default to primary
}

// Helper to avoid import cycle - we'll call screenshot package from hotkey.go
var getNumDisplays func() int
var getDisplayBounds func(int) RECT

// SetDisplayFuncs sets the display helper functions (called from main)
func SetDisplayFuncs(numFunc func() int, boundsFunc func(int) RECT) {
	getNumDisplays = numFunc
	getDisplayBounds = boundsFunc
}

// Image viewer/editor process names to block (re-capture prevention)
var imageViewerProcesses = []string{
	// Windows built-in
	"microsoft.photos.exe",
	"photos.exe",
	"mspaint.exe",
	"photoviewer.dll", // Old Windows Photo Viewer

	// Popular free viewers
	"irfanview.exe",
	"i_view64.exe",
	"i_view32.exe",
	"xnview.exe",
	"xnviewmp.exe",
	"honeyview.exe",
	"jpegview.exe",
	"faststone.exe",
	"fsviewer.exe",
	"fsimageresize.exe",
	"imageglass.exe",
	"nomacs.exe",
	"picasa3.exe",
	"123photoviewer.exe",
	"apowersoftviewer.exe",

	// Paid viewers
	"acdsee.exe",
	"acdseeultimate.exe",
	"acdseestandard.exe",

	// Adobe products
	"photoshop.exe",
	"lightroom.exe",
	"bridge.exe",
	"photoshopelements.exe",
	"illustrator.exe",
	"adobephotoshopexpress.exe",

	// Other editors
	"gimp-2.10.exe",
	"gimp-2.8.exe",
	"gimp.exe",
	"paint.net.exe",
	"paintdotnet.exe",
	"krita.exe",
	"affinity photo.exe",
	"photo.exe",
	"afphoto.exe",
	"coreldraw.exe",
	"paintshoppro.exe",
	"pspx.exe",
	"captureone.exe",
	"darktable.exe",
	"rawtherapee.exe",

	// Screenshot tools (might have image open)
	"snagit32.exe",
	"snagit64.exe",
	"snagiteditor.exe",
	"greenshot.exe",
	"sharex.exe",
	"lightshot.exe",
	"picpick.exe",
	"screenpresso.exe",
}

// processHasVisibleWindow checks if a process has any visible, non-minimized windows
func processHasVisibleWindow(processID uint32) bool {
	hasVisible := false

	// Callback for EnumWindows
	callback := syscall.NewCallback(func(hwnd uintptr, lParam uintptr) uintptr {
		var pid uint32
		procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))

		if pid == uint32(lParam) {
			visible, _, _ := procIsWindowVisible.Call(hwnd)
			minimized, _, _ := procIsIconic.Call(hwnd)

			// Only count as visible if window is visible AND not minimized
			if visible != 0 && minimized == 0 {
				hasVisible = true
				return 0 // Stop enumeration
			}
		}
		return 1 // Continue enumeration
	})

	procEnumWindows.Call(callback, uintptr(processID))
	return hasVisible
}

// IsImageViewerRunning checks if any image viewer/editor has a visible window
// Used to prevent re-capture attempts (screenshot of screenshot)
func IsImageViewerRunning() (bool, string) {
	snapshot, _, err := procCreateToolhelp32Snapshot.Call(TH32CS_SNAPPROCESS, 0)
	if snapshot == 0 || snapshot == ^uintptr(0) {
		fmt.Printf("[VIEWER] Snapshot failed: %v\n", err)
		return false, ""
	}
	defer procCloseHandle.Call(snapshot)

	var entry PROCESSENTRY32W
	entry.Size = uint32(unsafe.Sizeof(entry))

	ret, _, _ := procProcess32FirstW.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	if ret == 0 {
		return false, ""
	}

	for {
		exeName := strings.ToLower(syscall.UTF16ToString(entry.ExeFile[:]))

		for _, viewer := range imageViewerProcesses {
			if exeName == viewer || strings.Contains(exeName, strings.TrimSuffix(viewer, ".exe")) {
				// Check if this process has a visible window
				if processHasVisibleWindow(entry.ProcessID) {
					return true, exeName
				}
			}
		}

		ret, _, _ = procProcess32NextW.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
		if ret == 0 {
			break
		}
	}

	return false, ""
}
