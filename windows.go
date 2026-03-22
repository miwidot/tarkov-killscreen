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
	"os"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")

	procCreateToolhelp32Snapshot = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32FirstW          = kernel32.NewProc("Process32FirstW")
	procProcess32NextW           = kernel32.NewProc("Process32NextW")
	procCloseHandle              = kernel32.NewProc("CloseHandle")
	procCreateMutexW             = kernel32.NewProc("CreateMutexW")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procIsIconic                 = user32.NewProc("IsIconic")
	procMonitorFromWindow        = user32.NewProc("MonitorFromWindow")
	procGetMonitorInfoW          = user32.NewProc("GetMonitorInfoW")
	procOpenProcessToken         = advapi32.NewProc("OpenProcessToken")
	procGetTokenInformation      = advapi32.NewProc("GetTokenInformation")
	procShellExecuteW            = shell32.NewProc("ShellExecuteW")
)

const (
	MONITOR_DEFAULTTONEAREST = 2
)

const (
	ERROR_ALREADY_EXISTS = 183
	TH32CS_SNAPPROCESS   = 0x00000002
	MAX_PATH             = 260
)

// PROCESSENTRY32W is the Windows Toolhelp PROCESSENTRY32W structure used
// to enumerate running processes.
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

// RECT is the Windows RECT structure (left, top, right, bottom).
type RECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

// MONITORINFO is the Windows MONITORINFO structure used by GetMonitorInfoW.
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
		debugLn("[MUTEX] Failed to create mutex")
		return true // Failed to create mutex
	}

	// In Go syscall, the error code is in err (as syscall.Errno)
	if errno, ok := err.(syscall.Errno); ok && errno == ERROR_ALREADY_EXISTS {
		debugLn("[MUTEX] Another instance is already running")
		procCloseHandle.Call(handle)
		return true
	}

	debugLn("[MUTEX] This is the first instance")
	// Keep the mutex handle open - it will be released when the app exits
	return false
}

// skipTarkovCheck can be set to true via admin build tag to skip the process check.
var skipTarkovCheck bool

// IsTarkovRunning checks if EscapeFromTarkov.exe is currently running
func IsTarkovRunning() bool {
	if skipTarkovCheck {
		return true
	}
	snapshot, _, err := procCreateToolhelp32Snapshot.Call(TH32CS_SNAPPROCESS, 0)
	if snapshot == 0 || snapshot == ^uintptr(0) {
		debugLog("[TARKOV] Snapshot failed: %v\n", err)
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
			debugLog("[TARKOV] Found: %s\n", exeName)
			return true
		}

		ret, _, _ = procProcess32NextW.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
		if ret == 0 {
			break
		}
	}

	return false
}

// Callbacks registered once to avoid syscall.NewCallback memory leak
var (
	tarkovWindowCbOnce sync.Once
	tarkovWindowCb     uintptr
	tarkovWindowResult uintptr // hwnd found by callback
	tarkovWindowPids   []uint32
)

// GetTarkovDisplayIndex finds which display Tarkov is running on.
// Returns the display index (0-based) or -1 if not found.
func GetTarkovDisplayIndex() int {
	// Register callback once
	tarkovWindowCbOnce.Do(func() {
		tarkovWindowCb = syscall.NewCallback(func(hwnd uintptr, lParam uintptr) uintptr {
			var pid uint32
			procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
			for _, tarkovPid := range tarkovWindowPids {
				if pid == tarkovPid {
					visible, _, _ := procIsWindowVisible.Call(hwnd)
					if visible != 0 {
						tarkovWindowResult = hwnd
						debugLog("[TARKOV] Found window for PID %d\n", pid)
						return 0 // Stop enumeration
					}
				}
			}
			return 1 // Continue
		})
	})

	// Collect ALL Tarkov process IDs (BE launcher + actual game have different PIDs)
	tarkovWindowPids = nil
	tarkovWindowResult = 0

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
			debugLog("[TARKOV] Found PID %d: %s\n", entry.ProcessID, exeName)
			tarkovWindowPids = append(tarkovWindowPids, entry.ProcessID)
		}
		ret, _, _ = procProcess32NextW.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	}
	procCloseHandle.Call(snapshot)

	if len(tarkovWindowPids) == 0 {
		return -1
	}

	// Find a visible window belonging to any Tarkov process
	procEnumWindows.Call(tarkovWindowCb, 0)

	if tarkovWindowResult == 0 {
		debugLn("[TARKOV] Window not found, using display 0")
		return 0
	}

	// Use MonitorFromWindow - works for windowed, borderless, and fullscreen
	hMonitor, _, _ := procMonitorFromWindow.Call(tarkovWindowResult, MONITOR_DEFAULTTONEAREST)
	if hMonitor == 0 {
		return 0
	}

	// Get monitor info
	var mi MONITORINFO
	mi.Size = uint32(unsafe.Sizeof(mi))
	ret2, _, _ := procGetMonitorInfoW.Call(hMonitor, uintptr(unsafe.Pointer(&mi)))
	if ret2 == 0 {
		return 0
	}

	debugLog("[TARKOV] Monitor: (%d,%d)-(%d,%d)\n",
		mi.Monitor.Left, mi.Monitor.Top, mi.Monitor.Right, mi.Monitor.Bottom)

	// Match monitor rect to screenshot library display index
	// Try exact match first (all 4 corners), then nearest fallback
	n := getNumDisplays()
	for i := 0; i < n; i++ {
		bounds := getDisplayBounds(i)
		if bounds.Left == mi.Monitor.Left && bounds.Top == mi.Monitor.Top &&
			bounds.Right == mi.Monitor.Right && bounds.Bottom == mi.Monitor.Bottom {
			debugLog("[TARKOV] Exact match: display %d\n", i)
			return i
		}
	}

	// Fallback: find display with smallest distance (handles DPI scaling mismatches)
	bestIndex := 0
	bestDist := int64(1<<62 - 1)
	for i := 0; i < n; i++ {
		bounds := getDisplayBounds(i)
		dx := int64(bounds.Left - mi.Monitor.Left)
		dy := int64(bounds.Top - mi.Monitor.Top)
		dist := dx*dx + dy*dy
		debugLog("[TARKOV] Display %d: (%d,%d)-(%d,%d) dist=%d\n",
			i, bounds.Left, bounds.Top, bounds.Right, bounds.Bottom, dist)
		if dist < bestDist {
			bestDist = dist
			bestIndex = i
		}
	}

	debugLog("[TARKOV] Nearest match: display %d (dist=%d)\n", bestIndex, bestDist)
	return bestIndex
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

	// Game recording/overlay tools (overlays corrupt OCR)
	// NOTE: OBS intentionally excluded — users stream with it
	"medal.exe",
	"medalencoder.exe",
	"outplayed.exe",
	"overwolf.exe",
	"overwolfbrowser.exe",
	"insightscapture.exe",
	"capture.exe",
	"insights capture.exe",
	"plays.exe",
	"playstv.exe",
	"action.exe",        // Mirillis Action
	"bandicam.exe",
	"bdcam.exe",
	"fraps.exe",
	"xsplit.exe",
	"xsplitbroadcaster.exe",
	"xsplitgamecaster.exe",
	"gamebar.exe",       // Xbox Game Bar
	"gamebarpresencewriter.exe",
	"d3dgear.exe",       // D3DGear
	"litecam.exe",
	"raptr.exe",         // AMD Raptr/Gaming Evolved
	"plays_ep64.exe",
	"relive.exe",        // AMD ReLive
	"nvspcaps64.exe",    // NVIDIA ShadowPlay Capture Server
	"nvshare.exe",       // NVIDIA Share overlay (Alt+Z)
	"nvidia share.exe",  // NVIDIA Share overlay
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

// Callback for processHasVisibleWindow, registered once
var (
	visibleWindowCbOnce   sync.Once
	visibleWindowCb       uintptr
	visibleWindowCheckPid uint32
	visibleWindowFound    bool
)

// processHasVisibleWindow checks if a process has any visible, non-minimized
// windows by enumerating all top-level windows.
func processHasVisibleWindow(processID uint32) bool {
	visibleWindowCbOnce.Do(func() {
		visibleWindowCb = syscall.NewCallback(func(hwnd uintptr, lParam uintptr) uintptr {
			var pid uint32
			procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))

			if pid == visibleWindowCheckPid {
				visible, _, _ := procIsWindowVisible.Call(hwnd)
				minimized, _, _ := procIsIconic.Call(hwnd)

				// Only count as visible if window is visible AND not minimized
				if visible != 0 && minimized == 0 {
					visibleWindowFound = true
					return 0 // Stop enumeration
				}
			}
			return 1 // Continue enumeration
		})
	})

	visibleWindowCheckPid = processID
	visibleWindowFound = false
	procEnumWindows.Call(visibleWindowCb, 0)
	return visibleWindowFound
}

// IsImageViewerRunning checks if any image viewer/editor has a visible window
// Used to prevent re-capture attempts (screenshot of screenshot)
func IsImageViewerRunning() (bool, string) {
	snapshot, _, err := procCreateToolhelp32Snapshot.Call(TH32CS_SNAPPROCESS, 0)
	if snapshot == 0 || snapshot == ^uintptr(0) {
		debugLog("[VIEWER] Snapshot failed: %v\n", err)
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

// isElevated checks if the current process is running with admin (elevated) privileges.
// Uses advapi32 OpenProcessToken + GetTokenInformation(TokenElevation).
func isElevated() bool {
	handle, err := syscall.GetCurrentProcess()
	if err != nil {
		return false
	}

	var token syscall.Token
	ret, _, _ := procOpenProcessToken.Call(
		uintptr(handle),
		syscall.TOKEN_QUERY,
		uintptr(unsafe.Pointer(&token)),
	)
	if ret == 0 {
		return false
	}
	defer token.Close()

	// TokenElevation class = 20, returns a uint32 (0 = not elevated, nonzero = elevated)
	var elevation uint32
	var returnLen uint32
	ret, _, _ = procGetTokenInformation.Call(
		uintptr(token),
		20, // TokenElevation
		uintptr(unsafe.Pointer(&elevation)),
		4,
		uintptr(unsafe.Pointer(&returnLen)),
	)
	if ret == 0 {
		return false
	}

	return elevation != 0
}

// restartAsAdmin re-launches the current executable with "runas" verb (UAC prompt)
// and exits the current process.
func restartAsAdmin() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("os.Executable: %w", err)
	}

	verb, _ := syscall.UTF16PtrFromString("runas")
	file, _ := syscall.UTF16PtrFromString(exe)
	// SW_SHOWNORMAL = 1
	ret, _, _ := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		0, // no parameters
		0, // no directory
		1, // SW_SHOWNORMAL
	)
	// ShellExecuteW returns > 32 on success
	if ret <= 32 {
		return fmt.Errorf("ShellExecuteW returned %d", ret)
	}

	os.Exit(0)
	return nil // unreachable
}
