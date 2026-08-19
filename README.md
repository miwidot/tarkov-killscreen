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
1. Registers a global hotkey (Print Screen, F12, Scroll Lock, Pause, PageUp, PageDown, Insert, or Delete)
2. When user presses the hotkey, captures the screen directly via Windows API
3. Uploads the screenshot to our web service for OCR text recognition
4. Displays the extracted kill information to the user

This is functionally identical to a user manually uploading screenshots to a website — we just automate the upload process.

---

## Features

- **Hotkey Capture** — Press a configurable hotkey to capture screenshots directly from the game display
- **Smart Batching** — Collects multiple screenshots within a 20-second window (for scrollable kill lists, max 10 per batch)
- **Multi-Monitor Support** — Automatically captures from the display where Tarkov is running
- **Capture Feedback** — Screen flash, sound, and mini-overlay on capture (individually configurable)
- **Snipping Tool Override** — Automatically disables Windows 11 Snipping Tool when PrintScreen is used
- **OCR Analysis** — Sends screenshots to API for text extraction and kill parsing
- **Kill Tracking** — Saves analyzed kills to user's profile with full kill details
- **Kill Events** — Participate in community events with automatic kill assignment
- **Re-Capture Prevention** — Detects screenshots of screenshots via embedded pixel signature
- **Device Lock** — Token bound to a single device via UUID for abuse protection
- **Auto-Update** — Notifies when new versions are available (supports pre-releases)
- **Autostart** — Optional Windows autostart via registry
- **i18n** — UI available in German (default) and English
- **System Tray** — Runs quietly in the background with session statistics

## Client-Side Filters

To minimize server costs, the app filters invalid screenshots locally before uploading:

| Filter | Description |
|--------|-------------|
| **Tarkov Process Check** | Only processes screenshots when `EscapeFromTarkov.exe` is running |
| **Minimum Size** | Rejects images smaller than 800×400 pixels |
| **Aspect Ratio** | Accepts ratios between 1.2 and 3.8 (covers 4:3 to 32:9, rejects multi-monitor spans) |
| **Re-Capture Detection** | Detects and ignores screenshots of screenshots via embedded pixel signature |
| **Image Viewer Check** | Blocks capture when an image viewer is open to prevent accidental re-capture |

---

## Installation

1. Download the latest release from [Releases](https://github.com/miwidot/tarkov-killscreen/releases)
2. Run `screenshoter.exe`
3. On first run, enter your API token (get it from [tarkov-stammtisch.de](https://tarkov-stammtisch.de/en/profile/killcounter))
4. The app starts in the system tray, ready to capture

## Antivirus False Positives

A small number of antivirus engines flag `screenshoter.exe` as malware, typically
under a name like `Trojan-Spy.WinGo.Agent`. This is a false positive. Here is
exactly why it happens, so you can judge it for yourself instead of taking our
word for it.

### Why it happens

Antivirus engines do not only match known malware — they also classify programs
by what they are *capable* of. This app does five things that, taken together,
look identical to an information stealer:

| What the app does | How a scanner reads it |
|---|---|
| Captures the screen | Screen grabbing |
| Watches a global hotkey (Print Screen) | Keyboard monitoring |
| Uploads images to a server | Data exfiltration |
| Optionally starts with Windows | Persistence |
| Stores your API token in Credential Manager | Credential access |

That is the honest description of a kill screen uploader. It is also the honest
description of spyware. A scanner that has never seen this specific program has
no way to tell the difference from behaviour alone, so some err on the side of
warning you.

Two things make it more likely: the app is written in Go, and a large share of
today's commodity malware is written in Go too, so Go binaries get generic
signatures thrown at them. And every new release starts with zero reputation —
few downloads, so nothing vouches for the file yet.

### Decoding the detection name

`Trojan-Spy.WinGo.Agent` is not the name of a virus. It breaks down as:

- `Trojan-Spy` — the category for screen/input capturing behaviour
- `WinGo` — a Windows executable written in Go
- `Agent` — scanner shorthand for **no known malware family could be matched**

In other words: "a Go program on Windows that can capture the screen, belonging
to nothing we recognise." A real detection names a real family.

### How to verify this yourself

Do not trust this README either — check it:

1. **Check the detection ratio.** Upload the exe to [virustotal.com](https://www.virustotal.com)
   or search for its SHA256. A handful of hits out of ~72 engines, all with
   generic names, is the signature of a false positive. A genuine threat is
   flagged by most major engines with a consistent family name.
2. **Check the signature.** Right-click `screenshoter.exe` → Properties →
   Digital Signatures. It is signed by *Open Source Developer Martin Wilke*
   (issued by Certum). Malware distributors do not sign with a certificate that
   identifies them by name. In PowerShell:
   ```powershell
   Get-AuthenticodeSignature .\screenshoter.exe | Format-List Status, SignerCertificate
   ```
   Status must read `Valid`.
3. **Read the source.** This entire repository is the program. The upload
   endpoint, what gets sent, and when, are all in plain Go — see `upload.go`
   and `hotkey.go`. See also "What We Do NOT Do" below.
4. **Watch the traffic.** The app only talks to the kill counter API and the
   GitHub releases API. Any firewall or Wireshark will confirm it.

### What we do about it

False positive reports are filed with the affected vendors, and each release
ships with correct, signed file metadata so there is as little to be suspicious
of as possible. Detections usually disappear after a vendor reviews the report
or once a release has enough downloads to build reputation.

If your antivirus quarantines the app and you have satisfied yourself with the
checks above, add an exclusion for `screenshoter.exe`. If you are not
comfortable doing that, please do not — build it from source instead
(see [Building from Source](#building-from-source)), or simply do not use it.
Do not disable your antivirus.

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
├── main.go              # Entry point, single-instance check
├── app.go               # Core logic: tray menu, batching, upload coordination
├── hotkey.go            # Global hotkey registration, screen capture, Snipping Tool override
├── clipboard.go         # Windows clipboard API (legacy fallback)
├── windows.go           # Process enumeration, display mapping, viewer detection
├── upload.go            # HTTP multipart upload, OCR response parsing, kill save
├── config.go            # Configuration file handling (%APPDATA%\TarkovKillcounter)
├── credential.go        # Windows Credential Manager + AES-GCM encrypted backup
├── device.go            # Device ID generation (UUID v4) for token binding
├── events.go            # Kill event fetching and selection
├── settings.go          # Settings dialog UI (token, hotkey, language, feedback, events)
├── i18n.go              # Internationalization (DE/EN translations)
├── signature.go         # Pixel signature embedding/verification (re-capture detection)
├── version.go           # Auto-update checker (GitHub Releases API)
├── splash.go            # Splash screen with cached DIB
├── flash.go             # Screen flash effect (Win32 layered window)
├── overlay.go           # Mini overlay popup with countdown
├── sound.go             # Capture sound via winmm.dll
├── icon.go              # Tray icon generation (crosshair + update indicator)
├── autostart.go         # Windows autostart via registry
├── log.go               # Debug logging utilities
├── admin.go             # Admin build: skip Tarkov check (testing only)
├── debug_debug.go       # Debug build: verbose output, dev API URL
└── debug_release.go     # Release build: quiet output, production API URL
```

---

## Building from Source

### Requirements
- Go 1.24 or later
- Windows 10/11 (uses Windows-specific APIs)
- [go-winres](https://github.com/tc-hib/go-winres) — generates the icon and version
  resource: `go install github.com/tc-hib/go-winres@latest`. The build scripts derive
  the version from `version.go`, so the embedded file properties always match the
  release.

### Build Commands

```bash
# Release build (production API, no console output)
GOOS=windows GOARCH=amd64 go build -ldflags "-H=windowsgui -s -w" -o screenshoter.exe

# Debug build (dev API, console output)
GOOS=windows GOARCH=amd64 go build -ldflags "-H=windowsgui -s -w" -tags debug -o screenshoter_debug.exe

# Admin+Debug build (skip Tarkov check, console output, dev API)
GOOS=windows GOARCH=amd64 go build -ldflags "-H=windowsgui -s -w" -tags "admin,debug" -o screenshoter_admin.exe
```

### Build Tags

| Tag | Effect |
|-----|--------|
| (none) | Release build: production API, quiet output |
| `debug` | Dev API URL, verbose console logging, debug screenshots saved as JPEG |
| `admin` | Skips Tarkov process check (for testing without game running) |

### Dependencies

```
github.com/lxn/walk           # Windows GUI library (dialogs, tray icon)
github.com/lxn/win            # Windows API bindings
github.com/kbinani/screenshot  # Screen capture
golang.org/x/image             # Image scaling (CatmullRom)
golang.org/x/sys               # Windows registry access
```

---

## Configuration

Configuration is stored in `%APPDATA%\TarkovKillcounter\config.json`:

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
    "overlay_enabled": true,
    "overlay_duration": 3
  },
  "language": "de",
  "autostart": false,
  "kill_event_id": ""
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
| `feedback.overlay_duration` | Overlay display time in seconds (1-10) |
| `language` | UI language: `de` (German) or `en` (English) |
| `autostart` | Start with Windows |
| `kill_event_id` | Currently selected kill event (empty = no event) |

**Note:** The API token is stored securely in Windows Credential Manager, not in this file. An encrypted backup is kept in the config for recovery (e.g. after CCleaner wipes credentials).

---

## Security & Privacy

- **API Token** — Stored in Windows Credential Manager (encrypted by Windows), with AES-GCM encrypted backup in config
- **Device ID** — UUID v4 stored in `%APPDATA%\TarkovKillcounter\device.id`, sent as `X-Device-ID` header for token binding
- **Screenshots** — Uploaded via HTTPS only, compressed to JPEG before upload
- **No Telemetry** — No data collected beyond what you explicitly upload
- **Open Source** — Full source code available for review

---

## Technical Details

### Screenshot Processing Pipeline

1. **Hotkey Press** — `WatchHotkey()` polls `GetAsyncKeyState` at 50ms intervals
2. **Display Detection** — Identifies which monitor Tarkov is running on
3. **Screen Capture** — `screenshot.CaptureRect()` captures the correct display
4. **Validation** — Checks minimum size (800×400), aspect ratio (1.2–3.8), re-capture signature
5. **Signature Embedding** — Writes 8-byte signature into blue channel pixels (4 magic + 4 CRC32 hash)
6. **JPEG Compression** — Scales if wider than `max_width`, encodes as JPEG, immediately frees raw image
7. **Batching** — Adds to batch, resets 20-second timer (max 10 images, auto-uploads at 10)
8. **Upload** — Multipart POST to `/api/ocr`, frees JPEG buffers during construction
9. **Save Kills** — POST parsed kill data to `/api/kills/save` with device ID and optional event ID
10. **Notification** — Overlay popup with kill summary and raid details

### Memory Management

- Raw RGBA images (~15 MB at 1080p) freed immediately after JPEG compression
- Single-pass BGR→RGB clipboard conversion (no intermediate copy)
- JPEG buffers freed during multipart upload construction
- Splash screen DIB cached and reused across repaints
- Overlay fonts created once and reused

### Thread Safety

- `sync/atomic` for session counters (uploads, kills, failures) and watching state
- `sync.Mutex` for batch queue, pending queue, and event selection
- `sync.RWMutex` for active events list
- `mainWindow.Synchronize()` for all UI updates from goroutines

### Windows APIs Used

| DLL | Function | Purpose |
|-----|----------|---------|
| user32.dll | RegisterHotKey / UnregisterHotKey | Global hotkey registration |
| user32.dll | GetAsyncKeyState | Hotkey polling fallback |
| user32.dll | CreateWindowEx / RegisterClassEx | Flash, overlay, splash windows |
| user32.dll | SetLayeredWindowAttributes | Semi-transparent flash effect |
| user32.dll | OpenClipboard / CloseClipboard | Clipboard access (legacy fallback) |
| user32.dll | MonitorFromWindow / GetMonitorInfoW | Multi-monitor display detection |
| user32.dll | EnumWindows / GetWindowThreadProcessId | Image viewer detection |
| kernel32.dll | CreateToolhelp32Snapshot | List running processes |
| kernel32.dll | CreateMutexW | Single-instance check |
| gdi32.dll | CreateDIBSection / CreateCompatibleDC | Splash screen DIB caching |
| gdi32.dll | CreateSolidBrush / CreateFontIndirectW | Overlay rendering |
| winmm.dll | PlaySoundW | Capture sound feedback |
| advapi32.dll | CredWriteW / CredReadW / CredDeleteW | Credential Manager |

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

## Author

**Martin Wilke** — Malaysia

## License

MIT License — See [LICENSE](LICENSE) for details.

## Disclaimer

This project is not affiliated with Battlestate Games. Escape from Tarkov is a trademark of Battlestate Games Limited.

This tool only processes screenshots that the user manually captures. It does not interact with the game in any way that would violate the game's Terms of Service.
