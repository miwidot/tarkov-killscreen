// config.go - Configuration File Handling
//
// This file manages the application configuration stored in config.json.
// The config file is stored next to the executable and contains:
// - Hotkey settings
// - API URL and settings
//
// The API token is primarily stored in Windows Credential Manager.
// An encrypted backup is kept in config.json as fallback (e.g. after CCleaner).
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

// FeedbackConfig controls capture feedback mechanisms.
type FeedbackConfig struct {
	FlashEnabled   bool `json:"flash_enabled"`
	SoundEnabled   bool `json:"sound_enabled"`
	OverlayEnabled bool `json:"overlay_enabled"`
}

// Config is the top-level application configuration, serialized as config.json.
type Config struct {
	Hotkeys        HotkeyConfig   `json:"hotkeys"`
	API            APIConfig      `json:"api"`
	Feedback       FeedbackConfig `json:"feedback"`
	Language       string         `json:"language"`
	Autostart      bool           `json:"autostart"`
	EncryptedToken string         `json:"encrypted_token,omitempty"`
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
	Feedback: FeedbackConfig{
		FlashEnabled:   true,
		SoundEnabled:   true,
		OverlayEnabled: true,
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

	// Migrate old configs: if "feedback" key is missing, set defaults
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err == nil {
		if _, ok := raw["feedback"]; !ok {
			cfg.Feedback = defaultConfig.Feedback
		}
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
