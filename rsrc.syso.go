//go:build ignore

package main

// Run: go generate
// This embeds the manifest into the exe

//go:generate rsrc -manifest screenshoter.manifest -o rsrc.syso
