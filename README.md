# Tarkov Kill Screen Analyzer

A lightweight Windows background tool that automatically captures and analyzes Escape from Tarkov kill screens.

## Features

- **Auto-capture**: Automatically detects screenshots in clipboard
- **Smart batching**: Collects multiple screenshots within 15 seconds (for scrollable kill lists)
- **Kill analysis**: Sends screenshots to API for OCR analysis
- **Auto-save**: Saves analyzed kills to database
- **System tray**: Runs quietly in the background

## Installation

1. Download the latest release
2. Run `screenshoter.exe`
3. Right-click the tray icon → Settings
4. Enter your API token (get it from the web app)
5. Set your screenshot save path

## Usage

1. Play Tarkov
2. At the end of a raid, take screenshots of your kill screen (Win+Shift+S or PrintScreen)
3. If you have many kills, scroll and take multiple screenshots - they'll be batched automatically
4. After 15 seconds of no new screenshots, all images are uploaded and analyzed
5. You'll get a notification with your kill summary

## Building from Source

Requirements:
- Go 1.21+
- Windows (uses Windows APIs)

```bash
# Build GUI version (no console)
go build -ldflags -H=windowsgui -o screenshoter.exe

# Build debug version (with console)
go build -o screenshoter_debug.exe
```

## Configuration

Config file is created at `config.json` next to the executable:

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

## API Token

The API token is stored securely in Windows Credential Manager, not in the config file.

## License

MIT
