# Changelog

## 1.0.10 (2026-05-08)

### Fix: Capture nur wenn Tarkov im Vordergrund / Capture only when Tarkov is foreground
- Captures werden nur noch gemacht wenn Tarkov das aktive Fenster ist
- Verhindert versehentliche Uploads von Twitch / YouTube / Browser wenn Tarkov nur im Hintergrund laeuft
- Captures only happen when Tarkov is the active foreground window
- Prevents accidental uploads of Twitch / YouTube / browser screens when Tarkov is only running in the background

### Fix: Xbox Game Bar nicht mehr in Blocklist / Xbox Game Bar removed from blocklist
- `gamebar.exe` und `gamebarpresencewriter.exe` aus der Blocklist entfernt
- War unnoetig: laeuft auf jedem Windows by default und triggert keinen echten Re-Capture-Schutz
- `gamebar.exe` and `gamebarpresencewriter.exe` removed from the blocklist
- Was unnecessary: runs on every Windows by default and never actually triggered a re-capture issue

### Diagnose-Tool / Diagnostic Tool
- Neues Tool `diagnose.exe` zum Pruefen warum Captures eventuell blockiert werden
- Listet nur Programme die WIRKLICH blocken (Hintergrundprozesse ohne sichtbares Fenster werden ignoriert)
- New tool `diagnose.exe` to check why captures may be blocked
- Lists only programs that ACTUALLY block (background processes without visible windows are ignored)

---

## 1.0.9 (2026-05-06)

### Robusterer Verbindungsaufbau / More Robust Connectivity
- Bei DNS-Fehlern wird automatisch Cloudflare DNS (1.1.1.1) als Fallback verwendet
- Bei Erreichbarkeitsproblemen wird automatisch auf einen Backup-Endpoint ausgewichen
- Schuetzt vor kaputten ISP-/Router-DNS-Konfigurationen und groesseren Internet-Stoerungen
- Im Normalbetrieb kein Performance-Overhead — Fallback greift nur wenn der primaere Pfad fehlschlaegt

- On DNS errors, Cloudflare DNS (1.1.1.1) is used automatically as fallback
- On connectivity issues, requests fall over to a backup endpoint automatically
- Protects against broken ISP/router DNS configurations and broader internet outages
- Zero performance overhead in normal operation — fallback only kicks in when the primary path fails

### User-Agent Header
- HTTP-Requests senden jetzt einen aussagekraeftigen User-Agent mit Versions-Info
- Hilft uns beim Debugging und beim Erkennen veralteter Clients in Server-Logs

- HTTP requests now send a descriptive User-Agent including version info
- Helps with debugging and identifying outdated clients in server logs

---

## 1.0.8 (2026-04-23)

### API-Umstellung / API Migration
- Killcounter-API auf eigene Subdomain umgezogen
- Trennt die Killcounter-Infrastruktur von der Haupt-Tarkov-Stammtisch-Seite
- Keine Aktion noetig — Update installieren reicht

- Kill counter API moved to a dedicated subdomain
- Separates kill counter infrastructure from the main Tarkov-Stammtisch site
- No action needed — just install the update

---

## 1.0.7 (2026-04-22)

### SPT-Block / SPT Block
- Single Player Tarkov (SPT.Launcher.exe / SPT.Server.exe) wird erkannt und blockt Captures
- SPT-Kills sind keine Online-Kills und werden nicht hochgeladen
- Single Player Tarkov (SPT.Launcher.exe / SPT.Server.exe) is detected and blocks captures
- SPT kills are not online kills and will not be uploaded

### Upload-Fix / Upload Fix
- Content-Type wird jetzt korrekt als image/jpeg gesetzt (vorher application/octet-stream)
- Server muss nicht mehr auf Dateiendung zurueckfallen
- Content-Type is now correctly set as image/jpeg (previously application/octet-stream)
- Server no longer needs to fall back to file extension

### Kill-Daten / Kill Data
- Feld bodyPartSide wird jetzt an den Server weitergegeben
- Field bodyPartSide is now forwarded to the server

---

## 1.0.6 (2026-04-12)

### Automatisches Self-Update / Automatic Self-Update
- App kann sich jetzt selbst updaten ohne manuellen Download
- Bei neuem Update: Dialog -> "Ja" -> EXE wird automatisch heruntergeladen, ersetzt und App startet neu
- Fallback auf Browser-Download falls das automatische Update fehlschlaegt
- Heruntergeladene EXE wird vor dem Ersetzen auf Gueltigkeit geprueft (MZ-Header, Dateigroesse)

- App can now update itself without manual download
- On new update: dialog -> "Yes" -> EXE is downloaded automatically, replaced, and app restarts
- Falls back to browser download if automatic update fails
- Downloaded EXE is verified before replacing (MZ header, file size)

---

## 1.0.5 (2026-04-08)

### ✨ Neue Features

- **Kill-Sound** — Eigene WAV-Datei abspielen wenn Kills erkannt werden
  - Einstellbar in den Settings (Durchsuchen / Löschen / Vorschau ▶)
  - Non-blocking, stört die Aufnahme nicht

### 🔒 Code Signing

- **EXE ist jetzt digital signiert** (Certum Open Source Developer Certificate)
- Windows SmartScreen zeigt "Open Source Developer Martin Wilke" statt "Unbekannter Herausgeber"
- Weniger Fehlalarme von Windows Defender und Antivirus-Software

---

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
