//go:build debug

// debug_debug.go - Debug Build Configuration
//
// Included when building with -tags debug. Sets debugMode=true so that
// all verbose logging is printed (tokens, response bodies, PID detection,
// clipboard internals, signature checks, etc.).
//
// Build: go build -tags debug -o screenshoter_debug.exe
package main

var debugMode = true

// APIURL points to the dev API in debug builds.
const APIURL = "https://kcdev.tarkov-stammtisch.de/api/ocr"

func init() {
	// Use separate Credential Manager key so debug and release don't overwrite each other
	credentialTarget = "TarkovScreenshoter_APIToken_Dev"
}
