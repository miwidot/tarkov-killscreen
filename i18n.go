// i18n.go - Internationalization (DE/EN)
//
// Simple string-map based translation system.
// Default language is German ("de"). Supports "de" and "en".
// Language is stored in config.json and applied on startup.
package main

// currentLang is the active UI language. Default: German.
var currentLang = "de"

// T returns the translated string for the given key in the current language.
// Falls back to the key itself if no translation is found.
func T(key string) string {
	if lang, ok := translations[currentLang]; ok {
		if val, ok := lang[key]; ok {
			return val
		}
	}
	// Fallback to German if key missing in current language
	if de, ok := translations["de"]; ok {
		if val, ok := de[key]; ok {
			return val
		}
	}
	return key
}

// SetLanguage updates the active UI language.
func SetLanguage(lang string) {
	if _, ok := translations[lang]; ok {
		currentLang = lang
	}
}

// LanguageOptions for the settings dropdown.
var LanguageOptions = []string{"de", "en"}

// LanguageLabels maps language codes to display names.
var LanguageLabels = map[string]string{
	"de": "Deutsch",
	"en": "English",
}

var translations = map[string]map[string]string{
	"de": {
		// Balloon Notifications
		"screenshot.captured":     "Screenshot aufgenommen",
		"screenshot.waiting":      "Warte 20s auf weitere Screenshots...",
		"screenshot.batch":        "%d Screenshots im Batch. Warte 20s...",
		"screenshot.queued":       "Screenshot in Warteschlange",
		"screenshot.queued.count": "%d Screenshot(s) warten",
		"screenshot.limit":        "Maximum erreicht",
		"screenshot.limit.msg":    "Maximal %d Screenshots pro Batch. Warte auf Upload...",
		"batch.new":               "Neuer Batch gestartet",
		"batch.new.msg":           "%d Screenshot(s) aus Warteschlange. Warte 20s...",
		"batch.processing":        "Verarbeitung",
		"batch.uploading":         "Lade %d Screenshot(s) hoch...",
		"batch.novalid":           "Keine gültigen Screenshots zum Hochladen",
		"batch.done":              "Fertig",
		"batch.nokills":           "Keine Kills erkannt",
		"kills.saved":             "Kills gespeichert!",
		"kills.analysis":          "Kill-Analyse",
		"analysis.complete":       "Analyse abgeschlossen",
		"error":                   "Fehler",
		"upload.failed":           "Upload fehlgeschlagen",
		"not.tarkov":              "Keine Tarkov-Screenshots",
		"invalid.screenshot":      "Ungültiger Screenshot",
		"invalid.aspect":          "Seitenverhältnis nicht unterstützt. 16:9, 16:10, 21:9 oder 4:3 verwenden",
		"ready.capture":           "Drücke %s zum Aufnehmen! Screenshots werden auto-gebatcht (20s Fenster).",
		"settings.updated":        "Einstellungen aktualisiert",

		// Capture Blocking
		"capture.blocked":        "Aufnahme blockiert",
		"capture.closeviewer":    "Schließe %s zuerst um Doppelaufnahme zu verhindern",
		"capture.recapture":      "Doppelaufnahme erkannt",
		"capture.recapture.msg":  "Dies scheint ein Screenshot eines Screenshots zu sein. Ignoriert.",

		// Tray Menu
		"tray.hotkey":       "Hotkey: %s (20s Batch)",
		"tray.processnow":    "Jetzt verarbeiten",
		"tray.settings":      "Einstellungen...",
		"tray.exit":          "Beenden",
		"tray.stats":        "Uploads: %d | Kills: %d | Fehler: %d",
		"tray.token.notset": "Token: NICHT GESETZT",
		"tray.tooltip":      "Tarkov Screenshoter %s - Aufnahme aktiv",

		// Settings Dialog
		"settings.title":   "Einstellungen",
		"settings.token":   "API Token:",
		"settings.enable":  "API aktivieren",
		"settings.hotkey":  "Aufnahme-Hotkey:",
		"settings.lang":      "Sprache:",
		"settings.autostart": "Mit Windows starten",
		"settings.feedback":      "Aufnahme-Feedback:",
		"settings.feedback.desc": "Visuelles und akustisches Feedback beim Screenshot. Ersetzt die Windows-Benachrichtigung.",
		"settings.flash":         "Bildschirmblitz bei Aufnahme",
		"settings.sound":             "Bestätigungston abspielen",
		"settings.overlay":           "Aufnahme-Overlay anzeigen",
		"settings.overlay.duration":  "Overlay-Dauer (Sekunden):",
		"settings.event":        "Kill Event:",
		"settings.save":     "Speichern",
		"settings.cancel":   "Abbrechen",

		// First-Run Dialog
		"welcome.title":       "Willkommen beim Tarkov Killcounter!",
		"welcome.prompt":      "Bitte gib deinen API-Token von tarkov-stammtisch.de ein:",
		"welcome.link":        "Token erstellen auf tarkov-stammtisch.de",
		"welcome.save":        "Speichern",
		"welcome.exit":        "Beenden",
		"welcome.missing":     "Token fehlt",
		"welcome.missing.msg": "Bitte gib einen API-Token ein.",

		// Update
		"update.title":     "Update verfügbar",
		"update.message":   "Eine neue Version ist verfügbar!\n\nAktuelle Version: %s\nNeue Version: %s\n\nJetzt herunterladen?",
		"update.available": "⬆ Update: %s — Klicken zum Download",

		// Hotkey Labels
		"hotkey.PrintScreen":     "Druck",
		"hotkey.F2":              "F2",
		"hotkey.F3":              "F3",
		"hotkey.F4":              "F4",
		"hotkey.F5":              "F5",
		"hotkey.F6":              "F6",
		"hotkey.F7":              "F7",
		"hotkey.F8":              "F8",
		"hotkey.F9":              "F9",
		"hotkey.F10":             "F10",
		"hotkey.F11":             "F11",
		"hotkey.F12":             "F12",
		"hotkey.ScrollLock":      "Rollen",
		"hotkey.Pause":           "Pause",
		"hotkey.PageUp":          "Bild auf",
		"hotkey.PageDown":        "Bild ab",
		"hotkey.Home":            "Pos1",
		"hotkey.End":             "Ende",
		"hotkey.Insert":          "Einfg",
		"hotkey.Delete":          "Entf",
		"hotkey.Numpad0":         "Numpad 0",
		"hotkey.Numpad1":         "Numpad 1",
		"hotkey.Numpad2":         "Numpad 2",
		"hotkey.Numpad3":         "Numpad 3",
		"hotkey.Numpad4":         "Numpad 4",
		"hotkey.Numpad5":         "Numpad 5",
		"hotkey.Numpad6":         "Numpad 6",
		"hotkey.Numpad7":         "Numpad 7",
		"hotkey.Numpad8":         "Numpad 8",
		"hotkey.Numpad9":         "Numpad 9",
		"hotkey.NumpadMultiply":  "Numpad *",
		"hotkey.NumpadAdd":       "Numpad +",
		"hotkey.NumpadSubtract":  "Numpad -",
		"hotkey.NumpadDivide":    "Numpad /",

		// Overlay
		"overlay.captured":       "✓ Screenshot %d aufgenommen",
		"overlay.single":         "✓ Screenshot aufgenommen",
		"overlay.waiting":        "Nächster in %ds...",
		"overlay.upload.failed":  "✗ Upload fehlgeschlagen",

		// Admin / Elevation
		"tray.admin":          "Als Admin neu starten",
		"admin.hint":          "Tarkov läuft als Admin. Hotkey funktioniert evtl. nicht ohne Admin-Rechte.",
		"admin.restart.failed": "Neustart fehlgeschlagen",

		// Events
		"tray.event":    "Kill Event",
		"event.none":    "Kein Event",
		"event.refresh": "Events aktualisieren",
		"event.error":   "Event-Fehler",

		// Device Lock
		"device.locked":       "Token gesperrt",
		"device.locked.msg":   "Dieser Token ist an ein anderes Gerät gebunden. Bitte auf tarkov-stammtisch.de einen neuen Token erstellen.",

		// Already Running
		"already.running.title": "Tarkov Kill Screen Analyzer",
		"already.running.msg":   "Die Anwendung läuft bereits.",
	},
	"en": {
		// Balloon Notifications
		"screenshot.captured":     "Screenshot captured",
		"screenshot.waiting":      "Waiting 20s for more screenshots...",
		"screenshot.batch":        "%d screenshots in batch. Waiting 20s...",
		"screenshot.queued":       "Screenshot queued",
		"screenshot.limit":        "Limit reached",
		"screenshot.limit.msg":    "Maximum %d screenshots per batch. Waiting for upload...",
		"screenshot.queued.count": "%d screenshot(s) waiting",
		"batch.new":               "New batch started",
		"batch.new.msg":           "%d screenshot(s) from queue. Waiting 20s...",
		"batch.processing":        "Processing",
		"batch.uploading":         "Uploading %d screenshot(s)...",
		"batch.novalid":           "No valid screenshots to upload",
		"batch.done":              "Done",
		"batch.nokills":           "No kills detected",
		"kills.saved":             "Kills Saved!",
		"kills.analysis":          "Kill Analysis",
		"analysis.complete":       "Analysis Complete",
		"error":                   "Error",
		"upload.failed":           "Upload Failed",
		"not.tarkov":              "Not Tarkov Screenshots",
		"invalid.screenshot":      "Invalid Screenshot",
		"invalid.aspect":          "Aspect ratio not supported. Use 16:9, 16:10, 21:9, or 4:3",
		"ready.capture":           "Press %s to capture! Screenshots are auto-batched (20s window).",
		"settings.updated":        "Configuration updated",

		// Capture Blocking
		"capture.blocked":        "Capture Blocked",
		"capture.closeviewer":    "Close %s first to prevent re-capture",
		"capture.recapture":      "Re-Capture Detected",
		"capture.recapture.msg":  "This appears to be a screenshot of a screenshot. Ignored.",

		// Tray Menu
		"tray.hotkey":       "Hotkey: %s (20s batch)",
		"tray.processnow":    "Process Now (skip wait)",
		"tray.settings":      "Settings...",
		"tray.exit":          "Exit",
		"tray.stats":        "Uploads: %d | Kills: %d | Failed: %d",
		"tray.token.notset": "Token: NOT SET",
		"tray.tooltip":      "Tarkov Screenshoter %s - Auto-capture active",

		// Settings Dialog
		"settings.title":   "Settings",
		"settings.token":   "API Token:",
		"settings.enable":  "Enable API",
		"settings.hotkey":  "Capture Hotkey:",
		"settings.lang":      "Language:",
		"settings.autostart": "Start with Windows",
		"settings.feedback":      "Capture Feedback:",
		"settings.feedback.desc": "Visual and audio feedback on screenshot. Replaces the Windows notification.",
		"settings.flash":         "Screen flash on capture",
		"settings.sound":             "Play confirmation sound",
		"settings.overlay":           "Show capture overlay",
		"settings.overlay.duration":  "Overlay duration (seconds):",
		"settings.event":        "Kill Event:",
		"settings.save":     "Save",
		"settings.cancel":   "Cancel",

		// First-Run Dialog
		"welcome.title":       "Welcome to Tarkov Killcounter!",
		"welcome.prompt":      "Please enter your API token from tarkov-stammtisch.de:",
		"welcome.link":        "Create token on tarkov-stammtisch.de",
		"welcome.save":        "Save",
		"welcome.exit":        "Exit",
		"welcome.missing":     "Token missing",
		"welcome.missing.msg": "Please enter an API token.",

		// Update
		"update.title":     "Update available",
		"update.message":   "A new version is available!\n\nCurrent version: %s\nNew version: %s\n\nDownload now?",
		"update.available": "⬆ Update: %s — Click to download",

		// Hotkey Labels
		"hotkey.PrintScreen":     "Print Screen",
		"hotkey.F2":              "F2",
		"hotkey.F3":              "F3",
		"hotkey.F4":              "F4",
		"hotkey.F5":              "F5",
		"hotkey.F6":              "F6",
		"hotkey.F7":              "F7",
		"hotkey.F8":              "F8",
		"hotkey.F9":              "F9",
		"hotkey.F10":             "F10",
		"hotkey.F11":             "F11",
		"hotkey.F12":             "F12",
		"hotkey.ScrollLock":      "Scroll Lock",
		"hotkey.Pause":           "Pause/Break",
		"hotkey.PageUp":          "Page Up",
		"hotkey.PageDown":        "Page Down",
		"hotkey.Home":            "Home",
		"hotkey.End":             "End",
		"hotkey.Insert":          "Insert",
		"hotkey.Delete":          "Delete",
		"hotkey.Numpad0":         "Numpad 0",
		"hotkey.Numpad1":         "Numpad 1",
		"hotkey.Numpad2":         "Numpad 2",
		"hotkey.Numpad3":         "Numpad 3",
		"hotkey.Numpad4":         "Numpad 4",
		"hotkey.Numpad5":         "Numpad 5",
		"hotkey.Numpad6":         "Numpad 6",
		"hotkey.Numpad7":         "Numpad 7",
		"hotkey.Numpad8":         "Numpad 8",
		"hotkey.Numpad9":         "Numpad 9",
		"hotkey.NumpadMultiply":  "Numpad *",
		"hotkey.NumpadAdd":       "Numpad +",
		"hotkey.NumpadSubtract":  "Numpad -",
		"hotkey.NumpadDivide":    "Numpad /",

		// Overlay
		"overlay.captured":       "✓ Screenshot %d captured",
		"overlay.single":         "✓ Screenshot captured",
		"overlay.waiting":        "Next in %ds...",
		"overlay.upload.failed":  "✗ Upload failed",

		// Admin / Elevation
		"tray.admin":          "Restart as Admin",
		"admin.hint":          "Tarkov runs as admin. Hotkey may not work without admin rights.",
		"admin.restart.failed": "Restart failed",

		// Events
		"tray.event":    "Kill Event",
		"event.none":    "No Event",
		"event.refresh": "Refresh Events",
		"event.error":   "Event Error",

		// Device Lock
		"device.locked":       "Token Locked",
		"device.locked.msg":   "This token is bound to another device. Please create a new token on tarkov-stammtisch.de.",

		// Already Running
		"already.running.title": "Tarkov Kill Screen Analyzer",
		"already.running.msg":   "The application is already running.",
	},
}
