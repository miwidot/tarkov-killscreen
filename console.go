// console.go - Console Window Visibility
//
// Hides the console window on startup and provides a toggle
// via the system tray menu. Logs continue to be written to
// the console even when hidden — they become visible when
// the user shows the console again.
package main

var (
	procGetConsoleWindow = kernel32.NewProc("GetConsoleWindow")
	procShowWindowAsync  = user32.NewProc("ShowWindowAsync")
)

const (
	swHide = 0
	swShow = 5
)

var consoleVisible = true

// HideConsole hides the console window.
func HideConsole() {
	hwnd, _, _ := procGetConsoleWindow.Call()
	if hwnd != 0 {
		procShowWindowAsync.Call(hwnd, swHide)
		consoleVisible = false
	}
}

// ShowConsole shows the console window.
func ShowConsole() {
	hwnd, _, _ := procGetConsoleWindow.Call()
	if hwnd != 0 {
		procShowWindowAsync.Call(hwnd, swShow)
		consoleVisible = true
	}
}

// ToggleConsole switches console visibility.
func ToggleConsole() {
	if consoleVisible {
		HideConsole()
	} else {
		ShowConsole()
	}
}
