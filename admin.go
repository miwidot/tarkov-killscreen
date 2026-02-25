//go:build admin

// admin.go - Admin override (NEVER RELEASE)
//
// Skips Tarkov process check for testing without the game running.
// Build with: go build -tags admin
package main

func init() {
	// Override the Tarkov check — always returns true
	skipTarkovCheck = true
}
