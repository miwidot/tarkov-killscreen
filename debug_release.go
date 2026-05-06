//go:build !debug

// debug_release.go - Release Build Configuration
//
// Included in release builds (default). Sets debugMode=false so that
// verbose logging (tokens, response bodies, PID detection, etc.) is suppressed.
// Only user-facing messages (errors, results, status) are printed.
//
// Build: go build -o screenshoter.exe
package main

var debugMode = false

// APIURL is the production API endpoint, hardcoded in release builds.
const APIURL = "https://kc.tarkov-stammtisch.de/api/ocr"

// APIBackupURL is used when the primary host is unreachable (DNS/connect fail).
// Different TLD (.cc instead of .de) so DENIC outages don't take down both.
const APIBackupURL = "https://kc.notgood.cc/api/ocr"
