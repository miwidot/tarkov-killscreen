# Tarkov Kill Screen Analyzer

A lightweight Windows system tray application that helps players track their kills by analyzing end-of-raid screenshots.

## Important Notice for BSG / Battlestate Games

**This tool does NOT interact with Escape from Tarkov in any way.**

- **No game memory access** - We never read or write to the game's memory
- **No process injection** - We never inject code into the game process
- **No network interception** - We never intercept or modify game traffic
- **No file modification** - We never modify game files
- **No automation** - We never send inputs to the game

**What this tool actually does:**
1. Registers a global hotkey (Print Screen, F12, Scroll Lock, or Pause)
2. When user presses the hotkey, captures the screen directly via Windows API
3. Uploads the screenshot to our web service for OCR text recognition
4. Displays the extracted kill information to the user

This is functionally identical to a user manually uploading screenshots to a website - we just automate the upload process.

---

## Features

- **Hotkey Capture** - Press a configurable hotkey to capture screenshots directly
- **Smart Batching** - Collects multiple screenshots within a 20-second window (for scrollable kill lists, max 10 per batch)
- **Capture Feedback** - Screen flash, sound, and mini-overlay on capture (individually configurable)
- **Snipping Tool Override** - Automatically disables Windows 11 Snipping Tool when PrintScreen is used
- **OCR Analysis** - Sends screenshots to API for text extraction
- **Kill Tracking** - Saves analyzed kills to user's profile
- **Re-Capture Prevention** - Detects screenshots of screenshots via pixel signature
- **System Tray** - Runs quietly in the background
- **Auto-Update** - Notifies when new versions are available
- **Autostart** - Optional Windows autostart
- **i18n** - UI available in German (default) and English

## Client-Side Filters

To minimize server costs, the app filters invalid screenshots locally before uploading:

| Filter | Description |
|--------|-------------|
| **Tarkov Process Check** | Only processes screenshots when `EscapeFromTarkov.exe` is running |
| **Minimum Size** | Rejects images smaller than 800x400 pixels |
| **Aspect Ratio** | Accepts ratios between 1.2 and 3.8 (covers 4:3 to 32:9, rejects multi-monitor) |
| **Re-Capture Detection** | Detects and ignores screenshots of screenshots via embedded pixel signature |
| **Image Viewer Check** | Blocks capture when an image viewer is open to prevent accidental re-capture |

---

## Installation

1. Download the latest release from [Releases](https://github.com/miwidot/tarkov-killscreen/releases)
2. Run `screenshoter.exe`
3. On first run, enter your API token (get it from [tarkov-stammtisch.de](https://tarkov-stammtisch.de/en/profile/killcounter))
4. The app starts in the system tray, ready to capture

## Usage

1. Start the app (it will minimize to system tray)
2. Play Escape from Tarkov
3. At the end of a raid, press the capture hotkey (default: Print Screen)
4. If you have many kills (scrollable list), take multiple screenshots
5. After 20 seconds of no new screenshots, all images are uploaded together
6. You'll receive a notification with your kill summary

**Tip:** Use "Process Now" from the tray menu to skip the 20-second wait.

---

## Project Structure

```
screenshoter/
├── main.go            # Entry point, single-instance check
├── app.go             # Core logic: tray menu, batching, notifications
├── hotkey.go          # Global hotkey registration, screen capture, Snipping Tool override
├── clipboard.go       # Windows clipboard API (legacy fallback)
├── windows.go         # Windows process API (Tarkov detection, display info)
├── upload.go          # HTTP upload to OCR API, save kills
├── config.go          # Configuration file handling (config.json)
├── credential.go      # Windows Credential Manager (secure token storage)
├── settings.go        # Settings dialog UI (token, hotkey, language, feedback)
├── i18n.go            # Internationalization (DE/EN translations)
├── signature.go       # Pixel signature embedding/verification
├── version.go         # Auto-update checker (GitHub Releases)
├── splash.go          # Splash screen on startup
├── flash.go           # Screen flash effect on capture (Win32 layered window)
├── sound.go           # Capture sound via winmm.dll
├── overlay.go         # Mini overlay popup (dark/light theme aware)
├── icon.go            # Tray icon generation
├── log.go             # Logging utilities
├── debug_debug.go     # Debug build: verbose console output
├── debug_release.go   # Release build: quiet output
└── config.json        # User configuration (not committed)
```

---

## Building from Source

### Requirements
- Go 1.21 or later
- Windows 10/11 (uses Windows-specific APIs)

### Build Commands

```bash
# Release build (quiet output)
go build -o screenshoter.exe

# Debug build (verbose console output)
go build -tags debug -o screenshoter_debug.exe
```

### Dependencies

```
github.com/lxn/walk        # Windows GUI library
github.com/lxn/win         # Windows API bindings
github.com/kbinani/screenshot  # Screen capture
```

---

## Configuration

Configuration is stored in `config.json` next to the executable:

```json
{
  "hotkeys": {
    "capture_key": "PrintScreen"
  },
  "api": {
    "enabled": true,
    "mode": "kills",
    "max_width": 1920,
    "jpeg_quality": 85
  },
  "feedback": {
    "flash_enabled": true,
    "sound_enabled": true,
    "overlay_enabled": true
  },
  "language": "de",
  "autostart": false
}
```

| Field | Description |
|-------|-------------|
| `hotkeys.capture_key` | Capture hotkey: `PrintScreen`, `F12`, `ScrollLock`, `Pause`, `PageUp`, `PageDown`, `Insert`, `Delete` |
| `api.enabled` | Enable/disable API uploads |
| `api.mode` | OCR mode (always `kills`) |
| `api.max_width` | Max image width before resize (saves bandwidth) |
| `api.jpeg_quality` | JPEG compression quality (1-100) |
| `feedback.flash_enabled` | Screen flash on capture |
| `feedback.sound_enabled` | Sound on capture |
| `feedback.overlay_enabled` | Mini overlay popup on capture |
| `language` | UI language: `de` (German) or `en` (English) |
| `autostart` | Start with Windows |

**Note:** The API token is NOT stored in this file. It is stored securely in Windows Credential Manager.

The API URL is hardcoded per build and not configurable.

---

## Security & Privacy

- **API Token**: Stored in Windows Credential Manager (encrypted by Windows)
- **Screenshots**: Uploaded via HTTPS only to our server
- **No Telemetry**: We don't collect any data beyond what you explicitly upload
- **Open Source**: Full source code is available for review

---

## Technical Details

### Windows APIs Used

| DLL | Function | Purpose |
|-----|----------|---------|
| user32.dll | RegisterHotKey/UnregisterHotKey | Global hotkey registration |
| user32.dll | GetAsyncKeyState | Hotkey polling |
| user32.dll | CreateWindowEx/RegisterClassEx | Flash overlay, capture overlay, splash |
| user32.dll | SetLayeredWindowAttributes | Semi-transparent flash effect |
| user32.dll | GetClipboardSequenceNumber | Detect clipboard changes (fallback) |
| user32.dll | OpenClipboard/CloseClipboard | Access clipboard (fallback) |
| kernel32.dll | CreateToolhelp32Snapshot | List running processes |
| kernel32.dll | Process32First/Next | Enumerate processes |
| gdi32.dll | CreateSolidBrush/CreateFontIndirect | Overlay rendering |
| winmm.dll | PlaySoundW | Capture sound feedback |
| advapi32.dll | CredWrite/CredRead/CredDelete | Credential Manager |

All APIs used are standard, documented Windows APIs. No undocumented or game-specific APIs are used.

### What We Do NOT Do

- Read game memory
- Inject DLLs or code
- Hook game functions
- Intercept network traffic
- Modify game files
- Send automated inputs
- Bypass anti-cheat

---

## License

MIT License - See [LICENSE](LICENSE) for details.

## Disclaimer

This project is not affiliated with Battlestate Games. Escape from Tarkov is a trademark of Battlestate Games Limited.

This tool only processes screenshots that the user manually captures. It does not interact with the game in any way that would violate the game's Terms of Service.
