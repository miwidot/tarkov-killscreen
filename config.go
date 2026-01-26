package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type HotkeyConfig struct {
	Fullscreen   string `json:"fullscreen"`
	ActiveWindow string `json:"active_window"`
}

type APIConfig struct {
	Enabled     bool   `json:"enabled"`
	URL         string `json:"url"`
	Mode        string `json:"mode"`
	MaxWidth    int    `json:"max_width"`
	JPEGQuality int    `json:"jpeg_quality"`
}

type Config struct {
	ScreenshotPath string       `json:"screenshot_path"`
	FilenameFormat string       `json:"filename_format"`
	Hotkeys        HotkeyConfig `json:"hotkeys"`
	API            APIConfig    `json:"api"`
}

var defaultConfig = Config{
	ScreenshotPath: "",
	FilenameFormat: "screenshot_2006-01-02_15-04-05",
	Hotkeys: HotkeyConfig{
		Fullscreen:   "Ctrl+PrintScreen",
		ActiveWindow: "Alt+PrintScreen",
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
			cfg.ScreenshotPath = getDefaultScreenshotPath()
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

func getDefaultScreenshotPath() string {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, "Pictures", "Screenshots")
	os.MkdirAll(path, 0755)
	return path
}
