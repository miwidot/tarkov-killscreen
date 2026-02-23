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

	newToken := currentToken
	newEnabled := cfg.API.Enabled
	newHotkey := cfg.Hotkeys.CaptureKey
	if newHotkey == "" {
		newHotkey = "PrintScreen"
	}
	newLang := cfg.Language
	if newLang == "" {
		newLang = "de"
	}

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
		hotkeyLabels[i] = HotkeyLabels[opt]
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

	if err := (Dialog{
		AssignTo: &dlg,
		Title:    T("settings.title"),
		MinSize:  Size{Width: 400, Height: 300},
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
		SaveConfig(cfg)

		// Apply hotkey change immediately
		SetHotkey(newHotkey)

		// Apply language change immediately
		SetLanguage(newLang)

		return true, nil
	}

	return false, nil
}
