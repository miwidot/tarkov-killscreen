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

// HotkeyConfig holds the configured capture hotkey.
type HotkeyConfig struct {
	CaptureKey string `json:"capture_key"` // "PrintScreen", "F12", "F11", "ScrollLock"
}

// APIConfig holds the OCR API connection settings.
type APIConfig struct {
	Enabled     bool   `json:"enabled"`
	Mode        string `json:"mode"`
	MaxWidth    int    `json:"max_width"`
	JPEGQuality int    `json:"jpeg_quality"`
}

// Config is the top-level application configuration, serialized as config.json.
type Config struct {
	Hotkeys   HotkeyConfig `json:"hotkeys"`
	API       APIConfig    `json:"api"`
	Language  string       `json:"language"`
	Autostart bool         `json:"autostart"`
}

var defaultConfig = Config{
	Hotkeys: HotkeyConfig{
		CaptureKey: "PrintScreen",
	},
	API: APIConfig{
		Enabled:     true,
		Mode:        "kills",
		MaxWidth:    1920,
		JPEGQuality: 85,
	},
	Language: "de",
}

func getConfigPath() string {
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "config.json")
}

// LoadConfig reads config.json from the executable directory.
// If the file does not exist, a default config is created and returned.
// Missing fields in existing configs are backfilled with defaults.
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

	// Apply defaults for missing/zero values (old configs may not have these fields)
	if cfg.API.MaxWidth == 0 {
		cfg.API.MaxWidth = defaultConfig.API.MaxWidth
	}
	if cfg.API.JPEGQuality == 0 {
		cfg.API.JPEGQuality = defaultConfig.API.JPEGQuality
	}
	if cfg.API.Mode == "" {
		cfg.API.Mode = defaultConfig.API.Mode
	}
	if cfg.Language == "" {
		cfg.Language = defaultConfig.Language
	}

	return &cfg, nil
}

// SaveConfig writes the configuration to config.json next to the executable.
func SaveConfig(cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getConfigPath(), data, 0644)
}
