// device.go - Device ID for Token Binding
//
// Generates a unique device ID (UUID v4) on first start and stores it in
// %APPDATA%\TarkovKillcounter\device.id. This ID is sent as X-Device-ID
// header with every authenticated API request, allowing the server to
// bind tokens to specific devices.
package main

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var deviceID string

// LoadOrCreateDeviceID reads the device ID from disk, or generates a new
// UUID v4 and persists it. Called once at startup.
func LoadOrCreateDeviceID() string {
	path := filepath.Join(getConfigDir(), "device.id")

	data, err := os.ReadFile(path)
	if err == nil {
		id := strings.TrimSpace(string(data))
		if len(id) == 36 {
			deviceID = id
			debugLog("[DEVICE] Loaded device ID: %s\n", id)
			return id
		}
	}

	// Generate UUID v4
	var uuid [16]byte
	if _, err := rand.Read(uuid[:]); err != nil {
		fmt.Printf("[DEVICE] Failed to generate UUID: %v\n", err)
		return ""
	}
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant 1

	id := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])

	os.MkdirAll(getConfigDir(), 0755)
	if err := os.WriteFile(path, []byte(id), 0600); err != nil {
		fmt.Printf("[DEVICE] Failed to save device ID: %v\n", err)
	}

	deviceID = id
	fmt.Printf("[DEVICE] Generated new device ID: %s\n", id)
	return id
}

// GetDeviceID returns the cached device ID.
func GetDeviceID() string {
	return deviceID
}
