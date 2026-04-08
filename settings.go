// settings.go - Settings Dialog
//
// Provides a GUI dialog for configuring the application:
// - API token (stored securely via Windows Credential Manager)
// - Enable/disable API uploads
// - Capture hotkey selection (PrintScreen, F12, ScrollLock, Pause)
// - UI language (Deutsch / English)
//
// Changes are applied immediately after saving.
package main

import (
	"fmt"
	"os"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

// promptForToken shows a first-run dialog asking the user for their API token.
// Loops until a token is entered or the user closes the dialog (exits the app).
func promptForToken(cfg *Config) {
	for {
		var dlg *walk.Dialog
		var tokenLE *walk.LineEdit
		var enteredToken string

		if err := (Dialog{
			AssignTo: &dlg,
			Title:    T("welcome.title"),
			MinSize:  Size{Width: 450, Height: 200},
			Layout:   VBox{},
			Children: []Widget{
				Label{Text: T("welcome.prompt")},
				LineEdit{AssignTo: &tokenLE, PasswordMode: true},
				PushButton{
					Text: T("welcome.link"),
					OnClicked: func() {
						openBrowser("https://tarkov-stammtisch.de/en/profile/killcounter")
					},
				},
				VSpacer{},
				Composite{
					Layout: HBox{},
					Children: []Widget{
						HSpacer{},
						PushButton{
							Text: T("welcome.save"),
							OnClicked: func() {
								enteredToken = tokenLE.Text()
								dlg.Accept()
							},
						},
						PushButton{
							Text: T("welcome.exit"),
							OnClicked: func() {
								dlg.Cancel()
							},
						},
					},
				},
			},
		}).Create(nil); err != nil {
			return
		}

		if dlg.Run() == walk.DlgCmdOK {
			if enteredToken != "" {
				SaveToken(enteredToken)
				fmt.Println("[TOKEN] API-Token gespeichert")
				return
			}
			// Empty token — show dialog again
			walk.MsgBox(nil, T("welcome.missing"), T("welcome.missing.msg"), walk.MsgBoxIconWarning)
			continue
		}

		// User clicked exit — exit app
		fmt.Println("[TOKEN] Kein Token eingegeben, beende...")
		os.Exit(0)
	}
}

// ShowSettingsDialog opens a modal settings dialog. Returns true if the user
// saved changes. Token, API, hotkey, and language settings are applied immediately.
func ShowSettingsDialog(owner walk.Form, cfg *Config) (saved bool, err error) {
	currentToken, _ := LoadToken()

	var dlg *walk.Dialog
	var tokenLE *walk.LineEdit
	var enabledCB *walk.CheckBox
	var hotkeyCB *walk.ComboBox
	var langCB *walk.ComboBox
	var autostartCB *walk.CheckBox
	var flashCB *walk.CheckBox
	var soundCB *walk.CheckBox
	var overlayCB *walk.CheckBox
	var overlayDurationNE *walk.NumberEdit
	var killSoundLE *walk.LineEdit
	var eventCB *walk.ComboBox

	newToken := currentToken
	newEnabled := cfg.API.Enabled
	newAutostart := GetAutostart()
	newFlash := cfg.Feedback.FlashEnabled
	newSound := cfg.Feedback.SoundEnabled
	newOverlay := cfg.Feedback.OverlayEnabled
	newOverlayDuration := cfg.Feedback.OverlayDuration
	if newOverlayDuration == 0 {
		newOverlayDuration = 3
	}
	newKillSoundPath := cfg.Feedback.KillSoundPath
	newHotkey := cfg.Hotkeys.CaptureKey
	if newHotkey == "" {
		newHotkey = "PrintScreen"
	}
	newLang := cfg.Language
	if newLang == "" {
		newLang = "de"
	}
	newEventID := GetSelectedEventID()

	// Find current hotkey index
	hotkeyIndex := 0
	for i, opt := range HotkeyOptions {
		if opt == newHotkey {
			hotkeyIndex = i
			break
		}
	}

	// Build hotkey labels for dropdown
	hotkeyLabels := make([]string, len(HotkeyOptions))
	for i, opt := range HotkeyOptions {
		hotkeyLabels[i] = GetHotkeyLabel(opt)
	}

	// Find current language index
	langIndex := 0
	for i, opt := range LanguageOptions {
		if opt == newLang {
			langIndex = i
			break
		}
	}

	// Build language labels for dropdown
	langLabels := make([]string, len(LanguageOptions))
	for i, opt := range LanguageOptions {
		langLabels[i] = LanguageLabels[opt]
	}

	// Build event dropdown: ["Kein Event", "Event1 — Prize", "Event2 — Prize", ...]
	events := GetActiveEvents()
	eventLabels := make([]string, 0, len(events)+1)
	eventIDs := make([]string, 0, len(events)+1)
	eventLabels = append(eventLabels, T("event.none"))
	eventIDs = append(eventIDs, "")
	eventIndex := 0
	for _, e := range events {
		label := e.Name
		if e.Prize != "" {
			label += " — " + e.Prize
		}
		eventLabels = append(eventLabels, label)
		eventIDs = append(eventIDs, e.ID)
		if e.ID == newEventID {
			eventIndex = len(eventIDs) - 1
		}
	}

	if err := (Dialog{
		AssignTo: &dlg,
		Title:    T("settings.title"),
		MinSize:  Size{Width: 400, Height: 440},
		Layout:   VBox{},
		Children: []Widget{
			Label{Text: T("settings.token")},
			LineEdit{AssignTo: &tokenLE, Text: currentToken, PasswordMode: true},
			CheckBox{AssignTo: &enabledCB, Text: T("settings.enable"), Checked: cfg.API.Enabled},
			VSeparator{},
			Label{Text: T("settings.hotkey")},
			ComboBox{
				AssignTo:     &hotkeyCB,
				Model:        hotkeyLabels,
				CurrentIndex: hotkeyIndex,
				OnCurrentIndexChanged: func() {
					if hotkeyCB.CurrentIndex() >= 0 && hotkeyCB.CurrentIndex() < len(HotkeyOptions) {
						newHotkey = HotkeyOptions[hotkeyCB.CurrentIndex()]
					}
				},
			},
			VSeparator{},
			Label{Text: T("settings.lang")},
			ComboBox{
				AssignTo:     &langCB,
				Model:        langLabels,
				CurrentIndex: langIndex,
				OnCurrentIndexChanged: func() {
					if langCB.CurrentIndex() >= 0 && langCB.CurrentIndex() < len(LanguageOptions) {
						newLang = LanguageOptions[langCB.CurrentIndex()]
					}
				},
			},
			VSeparator{},
			CheckBox{AssignTo: &autostartCB, Text: T("settings.autostart"), Checked: newAutostart},
			VSeparator{},
			Label{Text: T("settings.event")},
			ComboBox{
				AssignTo:     &eventCB,
				Model:        eventLabels,
				CurrentIndex: eventIndex,
				OnCurrentIndexChanged: func() {
					idx := eventCB.CurrentIndex()
					if idx >= 0 && idx < len(eventIDs) {
						newEventID = eventIDs[idx]
					}
				},
			},
			VSeparator{},
			Label{Text: T("settings.feedback")},
			Label{Text: T("settings.feedback.desc"), Font: Font{PointSize: 8}, TextColor: walk.RGB(130, 130, 130)},
			CheckBox{AssignTo: &flashCB, Text: T("settings.flash"), Checked: cfg.Feedback.FlashEnabled},
			CheckBox{AssignTo: &soundCB, Text: T("settings.sound"), Checked: cfg.Feedback.SoundEnabled},
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					CheckBox{AssignTo: &overlayCB, Text: T("settings.overlay"), Checked: cfg.Feedback.OverlayEnabled},
					HSpacer{Size: 10},
					Label{Text: T("settings.overlay.duration")},
					NumberEdit{
						AssignTo: &overlayDurationNE,
						MinValue: 1,
						MaxValue: 10,
						Value:    float64(newOverlayDuration),
						Decimals: 0,
						MaxSize:  Size{Width: 60},
					},
					HSpacer{},
				},
			},
			VSeparator{},
			Label{Text: T("settings.killsound")},
			Label{Text: T("settings.killsound.desc"), Font: Font{PointSize: 8}, TextColor: walk.RGB(130, 130, 130)},
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					LineEdit{AssignTo: &killSoundLE, Text: newKillSoundPath, ReadOnly: true},
					PushButton{
						Text:    T("settings.killsound.browse"),
						MaxSize: Size{Width: 100},
						OnClicked: func() {
							dlgFile := new(walk.FileDialog)
							dlgFile.Filter = "WAV Files (*.wav)|*.wav"
							dlgFile.Title = T("settings.killsound.browse")
							if accepted, _ := dlgFile.ShowOpen(dlg); accepted {
								newKillSoundPath = dlgFile.FilePath
								killSoundLE.SetText(newKillSoundPath)
							}
						},
					},
					PushButton{
						Text:    T("settings.killsound.clear"),
						MaxSize: Size{Width: 60},
						OnClicked: func() {
							newKillSoundPath = ""
							killSoundLE.SetText("")
						},
					},
					PushButton{
						Text:    "▶",
						MaxSize: Size{Width: 30},
						OnClicked: func() {
							if newKillSoundPath != "" {
								PlayKillSound(newKillSoundPath)
							}
						},
					},
				},
			},
			VSpacer{},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						Text: T("settings.save"),
						OnClicked: func() {
							newToken = tokenLE.Text()
							newEnabled = enabledCB.Checked()
							newAutostart = autostartCB.Checked()
							newFlash = flashCB.Checked()
							newSound = soundCB.Checked()
							newOverlay = overlayCB.Checked()
							newOverlayDuration = int(overlayDurationNE.Value())
							if hotkeyCB.CurrentIndex() >= 0 {
								newHotkey = HotkeyOptions[hotkeyCB.CurrentIndex()]
							}
							if langCB.CurrentIndex() >= 0 {
								newLang = LanguageOptions[langCB.CurrentIndex()]
							}
							dlg.Accept()
						},
					},
					PushButton{
						Text: T("settings.cancel"),
						OnClicked: func() {
							dlg.Cancel()
						},
					},
				},
			},
		},
	}).Create(owner); err != nil {
		return false, err
	}

	if dlg.Run() == walk.DlgCmdOK {
		if newToken != "" {
			SaveToken(newToken)
		} else {
			DeleteToken()
		}
		cfg.API.Enabled = newEnabled
		cfg.Hotkeys.CaptureKey = newHotkey
		cfg.Language = newLang
		SetSelectedEventID(newEventID)
		cfg.KillEventID = newEventID
		go buildEventMenu()
		cfg.Autostart = newAutostart
		cfg.Feedback.FlashEnabled = newFlash
		cfg.Feedback.SoundEnabled = newSound
		cfg.Feedback.OverlayEnabled = newOverlay
		cfg.Feedback.OverlayDuration = newOverlayDuration
		cfg.Feedback.KillSoundPath = newKillSoundPath
		SaveConfig(cfg)

		// Apply hotkey change immediately
		SetHotkey(newHotkey)

		// Apply language change immediately
		SetLanguage(newLang)

		// Apply autostart change immediately
		SetAutostart(newAutostart)

		return true, nil
	}

	return false, nil
}
