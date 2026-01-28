package main

import (
	"os"
	"path/filepath"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

func ShowSettingsDialog(owner walk.Form, cfg *Config) (saved bool, err error) {
	currentToken, _ := LoadToken()

	home, _ := os.UserHomeDir()
	desktopPath := filepath.Join(home, "Desktop")

	var dlg *walk.Dialog
	var tokenLE, pathLE, urlLE *walk.LineEdit
	var enabledCB *walk.CheckBox
	var hotkeyCB *walk.ComboBox

	// Temporäre Variablen für Werte
	newToken := currentToken
	newPath := cfg.ScreenshotPath
	newURL := cfg.API.URL
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
		MinSize:  Size{Width: 400, Height: 300},
		Layout:   VBox{},
		Children: []Widget{
			Label{Text: "API Token:"},
			LineEdit{AssignTo: &tokenLE, Text: currentToken, PasswordMode: true},
			Label{Text: "API URL:"},
			LineEdit{AssignTo: &urlLE, Text: cfg.API.URL},
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
			VSeparator{},
			Label{Text: "Screenshot Path:"},
			LineEdit{AssignTo: &pathLE, Text: cfg.ScreenshotPath},
			PushButton{
				Text: "Use Desktop",
				OnClicked: func() {
					pathLE.SetText(desktopPath)
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
							// Werte kopieren
							newToken = tokenLE.Text()
							newPath = pathLE.Text()
							newURL = urlLE.Text()
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
		// Speichern NACH dialog.Run()
		if newToken != "" {
			SaveToken(newToken)
		} else {
			DeleteToken()
		}
		cfg.ScreenshotPath = newPath
		cfg.API.URL = newURL
		cfg.API.Enabled = newEnabled
		cfg.Hotkeys.CaptureKey = newHotkey
		SaveConfig(cfg)

		// Apply hotkey change immediately
		SetHotkey(newHotkey)

		return true, nil
	}

	return false, nil
}
