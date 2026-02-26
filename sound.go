// sound.go - Capture Sound Effect
//
// Plays the Windows system notification sound (SystemAsterisk) as audio
// feedback when a screenshot is captured. Uses winmm.dll PlaySoundW.
package main

import (
	"syscall"
	"unsafe"
)

var (
	winmm         = syscall.NewLazyDLL("winmm.dll")
	procPlaySound = winmm.NewProc("PlaySoundW")
)

const (
	sndAlias     = 0x00010000
	sndAsync     = 0x00000001
	sndNoDefault = 0x00000002
)

// PlayCaptureSound plays the system notification sound. Non-blocking.
func PlayCaptureSound() {
	go func() {
		name, _ := syscall.UTF16PtrFromString("SystemAsterisk")
		procPlaySound.Call(
			uintptr(unsafe.Pointer(name)),
			0,
			sndAlias|sndAsync|sndNoDefault,
		)
	}()
}
