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
	procGetLastError             = kernel32.NewProc("GetLastError")
)

const (
	ERROR_ALREADY_EXISTS = 183
)

const (
	TH32CS_SNAPPROCESS = 0x00000002
	MAX_PATH           = 260
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
	mutexName, _ := syscall.UTF16PtrFromString("TarkovKillScreenAnalyzer_SingleInstance")
	handle, _, _ := procCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(mutexName)))

	if handle == 0 {
		return true // Failed to create mutex
	}

	lastErr, _, _ := procGetLastError.Call()
	if lastErr == ERROR_ALREADY_EXISTS {
		procCloseHandle.Call(handle)
		return true // Another instance is running
	}

	// Keep the mutex handle open (don't close it) - it will be released when the app exits
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
