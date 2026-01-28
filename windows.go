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

// processHasVisibleWindow checks if a process has any visible windows
func processHasVisibleWindow(processID uint32) bool {
	hasVisible := false

	// Callback for EnumWindows
	callback := syscall.NewCallback(func(hwnd uintptr, lParam uintptr) uintptr {
		var pid uint32
		procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))

		if pid == uint32(lParam) {
			visible, _, _ := procIsWindowVisible.Call(hwnd)
			if visible != 0 {
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
