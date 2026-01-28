// config.go - Configuration File Handling
//
// This file manages the application configuration stored in config.json.
// The config file is stored next to the executable and contains:
// - Hotkey settings
// - API URL and settings
//
// Note: The API token is NOT stored in this file for security reasons.
// It is stored separately in Windows Credential Manager (see credential.go).
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type HotkeyConfig struct {
	CaptureKey string `json:"capture_key"` // "PrintScreen", "F12", "F11", "ScrollLock"
}

type APIConfig struct {
	Enabled     bool   `json:"enabled"`
	URL         string `json:"url"`
	Mode        string `json:"mode"`
	MaxWidth    int    `json:"max_width"`
	JPEGQuality int    `json:"jpeg_quality"`
}

type Config struct {
	Hotkeys HotkeyConfig `json:"hotkeys"`
	API     APIConfig    `json:"api"`
}

var defaultConfig = Config{
	Hotkeys: HotkeyConfig{
		CaptureKey: "PrintScreen",
	},
	API: APIConfig{
		Enabled:     true,
		URL:         "https://dev.tarkov-stammtisch.de/api/ocr",
		Mode:        "kills",
		MaxWidth:    1920,
		JPEGQuality: 85,
	},
}

func getConfigPath() string {
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "config.json")
}

func LoadConfig() (*Config, error) {
	configPath := getConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create default config
			cfg := defaultConfig
			SaveConfig(&cfg)
			return &cfg, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func SaveConfig(cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getConfigPath(), data, 0644)
}
