# Changelog

## Alpha 5 (2026-01-29)

### 🔥 Breaking Change: Clipboard Monitoring → Direct Hotkey Capture

**Das alte Clipboard-System hatte viele Probleme:**
- Clipboard wurde für 50-200ms blockiert während Pixel verarbeitet wurden
- Strg+C/Strg+V funktionierte systemweit nicht während der Verarbeitung
- Nach Text-Kopieren wurden Screenshots oft nicht mehr erkannt
- Race Conditions wenn mehrere Screenshots schnell hintereinander gemacht wurden
- Sequence Number Bugs bei verschiedenen Clipboard-Formaten

**Die neue Lösung:**
- **GetAsyncKeyState Polling** (50ms Intervall) für Print Screen Taste
- **Direkter Screenshot** mit `kbinani/screenshot` Library - kein Clipboard involviert!
- Keine Clipboard-Locks, keine Interferenz mit anderen Apps
- Volle Kontrolle über Bildqualität

### ✨ Neue Features

#### Hotkey-Auswahl in Settings
- Print Screen (Standard)
- F12
- Scroll Lock
- Pause/Break

Wählbar im Settings-Dialog, wird sofort angewendet.

#### Re-Capture Detection (Screenshot von Screenshot)
Zwei Schutzmechanismen:

1. **Image Viewer Detection** - Blockiert Capture wenn Bildbearbeitung offen ist:
   - Windows Photos, Paint
   - IrfanView, XnView, ImageGlass, FastStone, etc.
   - Adobe Photoshop, Lightroom, Bridge
   - GIMP, Paint.NET, Krita
   - Screenshot Tools (Snagit, Greenshot, ShareX)

   Nur aktiv wenn das Programm ein **sichtbares Fenster** hat (nicht minimiert).

2. **Signatur-System** - Eingebettete Signatur im Blue Channel:
   - 8 Bytes: "TRKV" Magic + CRC32 Hash
   - Position dynamisch basierend auf Bildgröße
   - Erkennt Screenshots von Screenshots mit Toleranz für JPEG-Artefakte

#### Debug-Modus
Screenshots werden in `debug/` Ordner gespeichert zur Inspektion.

### 🐛 Bug Fixes

- **Fix:** Gültige Kills werden jetzt gespeichert auch wenn manche Bilder invalid sind
- **Fix:** Image Viewer werden nur blockiert wenn Fenster sichtbar UND nicht minimiert
- **Entfernt:** Lokaler Prefilter (war unzuverlässig bei verschiedenen Monitor-Einstellungen)
- **Entfernt:** Screenshot Path Setting (nicht mehr benötigt, API-only Modus)
- **Entfernt:** "Open Screenshots Folder" Menü

### 📁 Code Cleanup

- `screenshot.go` gelöscht (unused)
- `prefilter.go` gelöscht (unzuverlässig)
- ScreenshotPath aus Config entfernt
- Kommentare aktualisiert

---

## Alpha 4 (2026-01-28)

### 🔧 Clipboard Reliability Improvements

**Performance-Optimierungen:**
- Schnelles Clipboard Copy mit `memcpy` statt Pixel-Loop
- Clipboard wird sofort geschlossen, Pixel-Konvertierung danach
- Lock-Zeit reduziert von 50-200ms auf <5ms

**Multi-Format Support:**
- CF_DIBV5 (primär)
- CF_DIB (Fallback)
- Retry-Logic mit 5 Versuchen

**Pending Queue:**
- Screenshots während Upload werden in Queue gespeichert
- Nach Upload-Ende wird Queue verarbeitet
- Keine Screenshots mehr verloren

### ✨ Features

- Single Instance Check (Mutex) - App kann nur einmal laufen
- pHash Support hinzugefügt (später wieder entfernt - zu unzuverlässig bei gescrollten Listen)
- Batch Window auf 20 Sekunden erhöht

### 🐛 Bug Fixes

- Race Condition bei `batchUploading` Flag behoben
- Clipboard Sequence Number Bugs behoben

---

## Technische Details

### Warum Clipboard → Hotkey?

| Problem (Clipboard) | Lösung (Hotkey) |
|---------------------|-----------------|
| Lock blockiert System | Kein Lock nötig |
| Sequence Number Bugs | Kein Clipboard Format |
| Race Conditions | Direkter Capture |
| Format-Kompatibilität | Immer RGBA |
| Andere Apps interferieren | Isoliert |

### Screenshot Signatur Format

```
Position: y = height - 3
          x = [10%, 30%, 50%, 70%, 20%, 40%, 60%, 80%] * width

Daten:    [0x54, 0x52, 0x4B, 0x56] + CRC32(timestamp, width, height)
           "T"   "R"   "K"   "V"     4 Bytes Hash

Kanal:    Blue Channel only (am wenigsten sichtbar)
```

### Image Viewer Blocklist

50+ Programme werden erkannt, aber nur blockiert wenn:
1. Prozess läuft UND
2. Fenster sichtbar (IsWindowVisible) UND
3. Fenster nicht minimiert (IsIconic = false)
