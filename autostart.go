// autostart.go - Windows Autostart via Registry
//
// Manages the "Run on Windows startup" feature by adding/removing
// an entry in HKCU\Software\Microsoft\Windows\CurrentVersion\Run.
package main

import (
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const autostartRegistryKey = `Software\Microsoft\Windows\CurrentVersion\Run`
const autostartValueName = "TarkovKillcounter"

// SetAutostart enables or disables the Windows autostart entry.
func SetAutostart(enabled bool) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, autostartRegistryKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	if enabled {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		return key.SetStringValue(autostartValueName, `"`+exe+`"`)
	}

	// Delete the value (ignore error if it doesn't exist)
	err = key.DeleteValue(autostartValueName)
	if err == registry.ErrNotExist {
		return nil
	}
	return err
}

// GetAutostart checks if the autostart registry entry exists.
func GetAutostart() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, autostartRegistryKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()

	_, _, err = key.GetStringValue(autostartValueName)
	return err == nil
}

// RefreshAutostart updates the autostart registry path to the current exe location.
// Should be called on every app start to handle exe moves/updates.
func RefreshAutostart() {
	if !GetAutostart() {
		return
	}

	key, err := registry.OpenKey(registry.CURRENT_USER, autostartRegistryKey, registry.QUERY_VALUE)
	if err != nil {
		return
	}

	val, _, err := key.GetStringValue(autostartValueName)
	key.Close()
	if err != nil {
		return
	}

	exe, err := os.Executable()
	if err != nil {
		return
	}

	// Compare stored path with current exe path (strip quotes)
	storedPath := strings.Trim(val, `"`)
	if !strings.EqualFold(storedPath, exe) {
		debugLog("[AUTOSTART] Path changed: %s → %s\n", storedPath, exe)
		SetAutostart(true)
	}
}
