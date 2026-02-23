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
const APIURL = "https://dev.tarkov-stammtisch.de/api/ocr"
