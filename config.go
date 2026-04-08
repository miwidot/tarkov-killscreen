// config.go - Configuration File Handling
//
// This file manages the application configuration stored in config.json.
// The config file is stored in %APPDATA%\TarkovKillcounter\ so it persists
// across exe updates regardless of where the user runs the exe from.
//
// Migration: On first run after the update, if config.json exists next to the
// exe but not in %APPDATA%, it is automatically moved.
//
// The API token is primarily stored in Windows Credential Manager.
// An encrypted backup is kept in config.json as fallback (e.g. after CCleaner).
package main

import (
	"encoding/json"
	"fmt"
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
	FlashEnabled   bool   `json:"flash_enabled"`
	SoundEnabled   bool   `json:"sound_enabled"`
	OverlayEnabled bool   `json:"overlay_enabled"`
	OverlayDuration int   `json:"overlay_duration"`
	KillSoundPath  string `json:"kill_sound_path,omitempty"`
}

// Config is the top-level application configuration, serialized as config.json.
type Config struct {
	Hotkeys        HotkeyConfig   `json:"hotkeys"`
	API            APIConfig      `json:"api"`
	Feedback       FeedbackConfig `json:"feedback"`
	Language       string         `json:"language"`
	Autostart      bool           `json:"autostart"`
	EncryptedToken string         `json:"encrypted_token,omitempty"`
	KillEventID    string         `json:"kill_event_id,omitempty"`
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
		FlashEnabled:    true,
		SoundEnabled:    true,
		OverlayEnabled:  true,
		OverlayDuration: 3,
	},
	Language: "de",
}

// getConfigDir returns the %APPDATA%\TarkovKillcounter directory.
func getConfigDir() string {
	appdata := os.Getenv("APPDATA")
	if appdata == "" {
		// Fallback: next to exe
		exe, _ := os.Executable()
		return filepath.Dir(exe)
	}
	return filepath.Join(appdata, "TarkovKillcounter")
}

func getConfigPath() string {
	return filepath.Join(getConfigDir(), "config.json")
}

// getLegacyConfigPath returns the old config path next to the executable.
func getLegacyConfigPath() string {
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "config.json")
}

// migrateConfigFromExeDir moves config.json from next to the exe to %APPDATA%.
// This runs once — old file is deleted after successful migration.
func migrateConfigFromExeDir() {
	newPath := getConfigPath()
	if _, err := os.Stat(newPath); err == nil {
		return // Already exists in %APPDATA%, nothing to migrate
	}

	oldPath := getLegacyConfigPath()
	data, err := os.ReadFile(oldPath)
	if err != nil {
		return // No old config either
	}

	// Create %APPDATA%\TarkovKillcounter directory
	if err := os.MkdirAll(getConfigDir(), 0755); err != nil {
		fmt.Printf("[CONFIG] Failed to create config dir: %v\n", err)
		return
	}

	// Write to new location
	if err := os.WriteFile(newPath, data, 0644); err != nil {
		fmt.Printf("[CONFIG] Failed to migrate config: %v\n", err)
		return
	}

	// Remove old file
	os.Remove(oldPath)
	fmt.Println("[CONFIG] Migrated config.json to", getConfigDir())
}

// LoadConfig reads config.json from %APPDATA%\TarkovKillcounter.
// On first run after update, migrates from the old exe-relative location.
// If no config exists anywhere, a default config is created and returned.
// Missing fields in existing configs are backfilled with defaults.
func LoadConfig() (*Config, error) {
	// Migrate old config from exe directory to %APPDATA%
	migrateConfigFromExeDir()

	configPath := getConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create config dir and default config
			os.MkdirAll(getConfigDir(), 0755)
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

	// Backfill overlay duration for configs that predate this field
	if cfg.Feedback.OverlayDuration == 0 {
		cfg.Feedback.OverlayDuration = defaultConfig.Feedback.OverlayDuration
	}

	return &cfg, nil
}

// SaveConfig writes the configuration to config.json in %APPDATA%\TarkovKillcounter.
func SaveConfig(cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getConfigPath(), data, 0600)
}
