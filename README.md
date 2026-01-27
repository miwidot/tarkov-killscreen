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
1. Monitors the Windows clipboard for images (standard Windows API)
2. When user manually takes a screenshot (PrintScreen/Win+Shift+S), we detect it
3. Uploads the screenshot to our web service for OCR text recognition
4. Displays the extracted kill information to the user

This is functionally identical to a user manually uploading screenshots to a website - we just automate the upload process.

---

## Features

- **Clipboard Monitoring** - Detects when user takes screenshots
- **Smart Batching** - Collects multiple screenshots within 15 seconds (for scrollable kill lists)
- **OCR Analysis** - Sends screenshots to API for text extraction
- **Kill Tracking** - Saves analyzed kills to user's profile
- **System Tray** - Runs quietly in the background
- **Auto-Update** - Notifies when new versions are available

## Client-Side Filters

To minimize server costs, the app filters invalid screenshots locally before uploading:

| Filter | Description |
|--------|-------------|
| **Tarkov Process Check** | Only processes screenshots when `EscapeFromTarkov.exe` is running |
| **Minimum Size** | Rejects images smaller than 800x400 pixels |
| **Aspect Ratio** | Accepts ratios between 1.2 and 3.8 (rejects multi-monitor captures) |
| **Duplicate Detection** | Skips identical images using pixel hash comparison |

---

## Installation

1. Download the latest release from [Releases](https://github.com/miwidot/tarkov-killscreen/releases)
2. Run `screenshoter.exe`
3. Right-click the tray icon → Settings
4. Enter your API token (get it from the web app)
5. Click Save

## Usage

1. Start the app (it will minimize to system tray)
2. Play Escape from Tarkov
3. At the end of a raid, take screenshots of your kill screen
   - Use `Win+Shift+S` to select just the kill list area, or
   - Use `PrintScreen` for full screen capture
4. If you have many kills (scrollable list), take multiple screenshots
5. After 15 seconds of no new screenshots, all images are uploaded
6. You'll receive a notification with your kill summary

---

## Project Structure

```
screenshoter/
├── main.go          # Entry point
├── app.go           # Main application logic, tray menu, clipboard watcher
├── clipboard.go     # Windows clipboard API (read images)
├── windows.go       # Windows process API (check if Tarkov is running)
├── upload.go        # HTTP upload to OCR API, save kills to database
├── config.go        # Configuration file handling
├── credential.go    # Windows Credential Manager (secure token storage)
├── settings.go      # Settings dialog UI
├── splash.go        # Splash screen on startup
├── version.go       # Version checking, auto-update notifications
├── icon.go          # Tray icon generation
├── screenshot.go    # Screenshot encoding (JPEG compression)
├── logo.png         # Application logo (embedded)
└── config.json      # User configuration (not committed)
```

## File Descriptions

### main.go
Entry point. Simply calls `RunApp()`.

### app.go
Core application logic:
- Initializes system tray icon and menu
- Runs clipboard watcher in background goroutine
- Implements auto-batching (15 second window for multiple screenshots)
- Handles image validation (size, aspect ratio, duplicates)
- Coordinates upload and displays notifications

### clipboard.go
Windows clipboard interaction using `user32.dll`:
- `GetClipboardSequenceNumber()` - Detects clipboard changes
- `HasClipboardImage()` - Checks if clipboard contains an image
- `GetClipboardImage()` - Extracts image data from clipboard (CF_DIB format)

**Note:** This only reads from clipboard - it does not interact with any application.

### windows.go
Windows process enumeration using `kernel32.dll`:
- `IsTarkovRunning()` - Checks if `EscapeFromTarkov.exe` is in the process list

Uses `CreateToolhelp32Snapshot` and `Process32First/Next` - standard Windows APIs for listing running processes. This does NOT access the game process memory.

### upload.go
HTTP communication with the backend API:
- `UploadScreenshot()` - Uploads single image for OCR analysis
- `UploadMultipleScreenshots()` - Uploads batch of images
- `SaveKills()` - Saves analyzed kills to user's database

All communication is HTTPS to our own server. No game servers are contacted.

### config.go
Handles `config.json` for user preferences:
- Screenshot save path
- API URL and settings
- JPEG quality settings

### credential.go
Secure token storage using Windows Credential Manager (`advapi32.dll`):
- `SaveToken()` - Stores API token securely
- `LoadToken()` - Retrieves API token
- `HasToken()` - Checks if token exists

Tokens are never stored in plain text files.

### settings.go
Settings dialog using [walk](https://github.com/lxn/walk) GUI library:
- API token input
- API URL configuration
- Enable/disable toggle

### version.go
Automatic update checker:
- Queries GitHub Releases API
- Compares version strings
- Shows Windows dialog if update available
- Checks on startup and every 30 minutes

### splash.go
Shows logo briefly on startup using Windows API.

### icon.go
Generates the tray icon programmatically (green circle with "T").

### screenshot.go
Image processing:
- Resizes large images to max width (saves bandwidth)
- Encodes as JPEG with configurable quality

---

## Building from Source

### Requirements
- Go 1.21 or later
- Windows 10/11 (uses Windows-specific APIs)

### Build Commands

```bash
# Release build (no console window)
go build -ldflags -H=windowsgui -o screenshoter.exe

# Debug build (with console for logging)
go build -o screenshoter_debug.exe
```

### Dependencies

```
github.com/lxn/walk    # Windows GUI library
github.com/lxn/win     # Windows API bindings
```

---

## Configuration

Configuration is stored in `config.json` next to the executable:

```json
{
  "screenshot_path": "C:\\Users\\YourName\\Pictures\\Screenshots",
  "api": {
    "enabled": true,
    "url": "https://tarkov-stammtisch.de/api/ocr",
    "mode": "kills",
    "max_width": 1920,
    "jpeg_quality": 85
  }
}
```

**Note:** The API token is NOT stored in this file. It is stored securely in Windows Credential Manager.

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
| user32.dll | GetClipboardSequenceNumber | Detect clipboard changes |
| user32.dll | OpenClipboard/CloseClipboard | Access clipboard |
| user32.dll | GetClipboardData | Read clipboard content |
| user32.dll | IsClipboardFormatAvailable | Check for image data |
| kernel32.dll | GlobalLock/GlobalUnlock | Access clipboard memory |
| kernel32.dll | CreateToolhelp32Snapshot | List running processes |
| kernel32.dll | Process32First/Next | Enumerate processes |
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
