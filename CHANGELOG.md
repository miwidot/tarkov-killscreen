# Changelog

## 1.0.13 (unveroeffentlicht / unreleased)

### Sicherheit: Updates werden vor der Installation geprueft / Security: Updates are verified before install
- Das Selbst-Update prueft jetzt die digitale Signatur der heruntergeladenen Exe
- Installiert wird nur, was gueltig signiert und auf `Open Source Developer Martin Wilke` ausgestellt ist
- Eine ausgetauschte oder veraenderte Datei wird geloescht statt installiert — auch falls das GitHub-Konto des Projekts uebernommen wuerde
- Vorher wurde nur geprueft, ob die Datei ueberhaupt eine Exe ist

- Self-update now checks the digital signature of the downloaded exe
- Only a file that is validly signed and issued to `Open Source Developer Martin Wilke` gets installed
- A swapped or modified file is deleted instead of installed — even if the project's GitHub account were taken over
- Previously the only check was whether the file is an exe at all

### Sicherheit: Aktuelle Go-Version / Security: Current Go version
- Gebaut mit Go 1.26.11 statt 1.26.0
- Damit sind 27 bekannte Schwachstellen der Go-Standardbibliothek behoben, die Verbindungen und die Zertifikatspruefung betrafen
- `golang.org/x/sys` und weitere Abhaengigkeiten aktualisiert

- Built with Go 1.26.11 instead of 1.26.0
- This fixes 27 known vulnerabilities in the Go standard library affecting connections and certificate checks
- `golang.org/x/sys` and other dependencies updated

### Neu: Upload wird bei Verbindungsproblemen wiederholt / New: Upload is retried on connection problems
- Bricht die Verbindung ab oder antwortet der Server gerade nicht, versucht die App es nach 5, 15 und 30 Sekunden erneut
- Vorher waren in diesem Fall alle Screenshots des Raids verloren
- Eine Meldung zeigt an, dass die Screenshots erhalten bleiben
- Das Speichern der Kills wird nur wiederholt, wenn die Anfrage sicher nicht angekommen ist, damit kein Raid doppelt gezaehlt wird

- If the connection drops or the server is not answering, the app tries again after 5, 15 and 30 seconds
- Previously all screenshots of that raid were lost in this case
- A notification tells you the screenshots are being kept
- Saving the kills is only repeated when the request provably did not arrive, so no raid is counted twice

### Fix: Token-Sicherung ging verloren / Token backup got lost
- Die Sicherungskopie des Tokens in der `config.json` wurde beim naechsten Speichern der Einstellungen oder bei der Event-Auswahl wieder geloescht oder durch ein altes Token ersetzt
- Dadurch musste das Token nach einem Aufraeum-Programm wie CCleaner doch neu eingegeben werden
- Die Sicherung bleibt jetzt erhalten und wird beim Start automatisch repariert, falls sie fehlt oder veraltet ist

- The backup copy of the token in `config.json` was deleted again, or replaced by an old token, the next time settings were saved or an event was selected
- As a result the token still had to be re-entered after a cleanup tool such as CCleaner
- The backup is now kept and is repaired automatically on start if it is missing or outdated

### Fix: Snipping-Tool-Einstellung wird korrekt zurueckgesetzt / Snipping Tool setting is restored correctly
- Die App merkt sich jetzt, wie die Druck-Taste vorher eingestellt war, und stellt genau das wieder her
- Vorher wurde das Snipping Tool beim Beenden immer eingeschaltet, auch wenn es vorher abgeschaltet war
- Nach einem Absturz oder Update wird die Einstellung beim naechsten sauberen Beenden nachgeholt

- The app now remembers how the Print Screen key was configured before and restores exactly that
- Previously the Snipping Tool was always switched on at exit, even if it had been off before
- After a crash or an update the setting is caught up on the next clean exit

### Fix: "Jetzt verarbeiten" waehrend eines Uploads / "Process Now" during an upload
- Ein Klick auf "Jetzt verarbeiten" waehrend ein Upload lief konnte dazu fuehren, dass zwei Uploads gleichzeitig liefen
- Der Klick wird jetzt ignoriert, solange ein Upload laeuft

- Clicking "Process Now" while an upload was running could lead to two uploads running at the same time
- The click is now ignored while an upload is in progress

### Fix: Sehr kurzes Token / Very short token
- Ein Token mit weniger als vier Zeichen liess den Upload ohne Fehlermeldung abbrechen
- Es kommt jetzt die normale Fehlermeldung des Servers

- A token shorter than four characters made the upload abort without any error message
- The regular server error is shown now

### Doku / Docs
- `TRANSPARENCY.md` listet jetzt auch auf, was die App auf dem Rechner aendert (Dateien, Registry), und nennt alle Verbindungen inklusive Update-Download und DNS-Ausweichweg
- README an den aktuellen Code angepasst

- `TRANSPARENCY.md` now also lists what the app changes on your machine (files, registry) and names every connection including the update download and the DNS fallback
- README brought in line with the current code

---

## 1.0.12 (2026-09-30)

### Aenderung: Bis zu 20 Screenshots pro Upload / Change: Up to 20 screenshots per upload
- Das Limit pro Batch wurde von 10 auf 20 Screenshots erhoeht
- Lange, scrollbare Kill-Listen passen damit in einen einzigen Upload
- Der automatische Upload startet jetzt erst bei 20 Bildern (oder wie bisher nach 20 Sekunden ohne neuen Screenshot)

- The per-batch limit was raised from 10 to 20 screenshots
- Long, scrollable kill lists now fit into a single upload
- The automatic upload now triggers at 20 images (or, as before, after 20 seconds without a new screenshot)

---

## 1.0.11 (2026-08-20)

### Hinweis: Virenscanner-Fehlalarm / Note: Antivirus False Positive
- Einzelne Virenscanner melden die Exe faelschlich als `Trojan-Spy.WinGo.Agent` oder aehnlich
- Grund: die App macht Screenshots, liest eine globale Taste mit und laedt Bilder hoch — genau dieses Verhalten stufen manche Scanner pauschal als Spionage-Software ein, unabhaengig davon wofuer es benutzt wird
- Der Name `Agent` bedeutet in der Scanner-Sprache "keiner bekannten Schadsoftware zuzuordnen", es ist also kein Fund eines konkreten Virus
- Die Exe ist mit einem Zertifikat auf Martin Wilke signiert, der Quellcode ist vollstaendig einsehbar
- Details und Pruefschritte stehen im README unter "Antivirus False Positives"
- Ein Fehlalarm-Report ist bei den betroffenen Herstellern eingereicht

- Some antivirus engines falsely flag the exe as `Trojan-Spy.WinGo.Agent` or similar
- Reason: the app takes screenshots, watches a global hotkey and uploads images — some scanners classify exactly this behaviour as spyware regardless of what it is used for
- The name `Agent` is scanner shorthand for "cannot be attributed to any known malware", so it is not a hit on an actual virus
- The exe is signed with a certificate issued to Martin Wilke and the full source code is public
- Details and verification steps are in the README under "Antivirus False Positives"
- A false positive report has been filed with the affected vendors

### Fix: Korrekte Versions-Angabe in der Exe / Correct version info in the exe
- Die Datei-Eigenschaften zeigten noch 1.0.3 obwohl 1.0.10 installiert war
- Copyright-Feld war leer, Anwendungsname im Manifest veraltet
- Version wird jetzt beim Bauen automatisch aus dem Quellcode uebernommen und kann nicht mehr abweichen
- Saubere Datei-Eigenschaften helfen zusaetzlich gegen Fehlalarme von Virenscannern

- File properties still showed 1.0.3 even though 1.0.10 was installed
- Copyright field was empty, application name in the manifest outdated
- The version is now taken from the source code automatically at build time and can no longer drift
- Clean file properties also help against antivirus false positives

### Fix: Update laedt garantiert die richtige Datei / Update reliably picks the right file
- Das Selbst-Update sucht im Release jetzt gezielt nach `screenshoter.exe`
- Vorher wurde die erste passende `.exe` genommen — sobald ein Release Zusatz-Tools wie `diagnose.exe` enthaelt, haette das die falsche Datei sein koennen

- Self-update now looks for `screenshoter.exe` by exact name in the release
- Previously it took the first matching `.exe` — once a release ships extra tools such as `diagnose.exe`, that could have been the wrong file

---

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
