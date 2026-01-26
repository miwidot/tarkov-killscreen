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

	// Temporäre Variablen für Werte
	newToken := currentToken
	newPath := cfg.ScreenshotPath
	newURL := cfg.API.URL
	newEnabled := cfg.API.Enabled

	if err := (Dialog{
		AssignTo: &dlg,
		Title:    "Settings",
		MinSize:  Size{Width: 400, Height: 250},
		Layout:   VBox{},
		Children: []Widget{
			Label{Text: "API Token:"},
			LineEdit{AssignTo: &tokenLE, Text: currentToken, PasswordMode: true},
			Label{Text: "API URL:"},
			LineEdit{AssignTo: &urlLE, Text: cfg.API.URL},
			CheckBox{AssignTo: &enabledCB, Text: "Enable API", Checked: cfg.API.Enabled},
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
		SaveConfig(cfg)
		return true, nil
	}

	return false, nil
}
