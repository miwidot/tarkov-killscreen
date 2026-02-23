// settings.go - Settings Dialog
//
// Provides a GUI dialog for configuring the application:
// - API token (stored securely via Windows Credential Manager)
// - Enable/disable API uploads
// - Capture hotkey selection (PrintScreen, F12, ScrollLock, Pause)
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
			Title:    "Willkommen beim Tarkov Killcounter!",
			MinSize:  Size{Width: 450, Height: 200},
			Layout:   VBox{},
			Children: []Widget{
				Label{Text: "Bitte gib deinen API-Token von tarkov-stammtisch.de ein:"},
				LineEdit{AssignTo: &tokenLE, PasswordMode: true},
				PushButton{
					Text: "Token erstellen auf tarkov-stammtisch.de",
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
							Text: "Speichern",
							OnClicked: func() {
								enteredToken = tokenLE.Text()
								dlg.Accept()
							},
						},
						PushButton{
							Text: "Beenden",
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
			walk.MsgBox(nil, "Token fehlt", "Bitte gib einen API-Token ein.", walk.MsgBoxIconWarning)
			continue
		}

		// User clicked "Beenden" — exit app
		fmt.Println("[TOKEN] Kein Token eingegeben, beende...")
		os.Exit(0)
	}
}

// ShowSettingsDialog opens a modal settings dialog. Returns true if the user
// saved changes. Token, API, and hotkey settings are applied immediately.
func ShowSettingsDialog(owner walk.Form, cfg *Config) (saved bool, err error) {
	currentToken, _ := LoadToken()

	var dlg *walk.Dialog
	var tokenLE *walk.LineEdit
	var enabledCB *walk.CheckBox
	var hotkeyCB *walk.ComboBox

	// Temporäre Variablen für Werte
	newToken := currentToken
	newEnabled := cfg.API.Enabled
	newHotkey := cfg.Hotkeys.CaptureKey
	if newHotkey == "" {
		newHotkey = "PrintScreen"
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

	if err := (Dialog{
		AssignTo: &dlg,
		Title:    "Settings",
		MinSize:  Size{Width: 400, Height: 250},
		Layout:   VBox{},
		Children: []Widget{
			Label{Text: "API Token:"},
			LineEdit{AssignTo: &tokenLE, Text: currentToken, PasswordMode: true},
			CheckBox{AssignTo: &enabledCB, Text: "Enable API", Checked: cfg.API.Enabled},
			VSeparator{},
			Label{Text: "Capture Hotkey:"},
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
			VSpacer{},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						Text: "Save",
						OnClicked: func() {
							newToken = tokenLE.Text()
							newEnabled = enabledCB.Checked()
							if hotkeyCB.CurrentIndex() >= 0 {
								newHotkey = HotkeyOptions[hotkeyCB.CurrentIndex()]
							}
							dlg.Accept()
						},
					},
					PushButton{
						Text: "Cancel",
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
		SaveConfig(cfg)

		// Apply hotkey change immediately
		SetHotkey(newHotkey)

		return true, nil
	}

	return false, nil
}
