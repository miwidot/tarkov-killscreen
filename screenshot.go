package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"time"
)

func saveScreenshot(img *image.RGBA, cfg *Config) (string, error) {
	// Ensure directory exists
	if err := os.MkdirAll(cfg.ScreenshotPath, 0755); err != nil {
		return "", err
	}

	// Generate filename
	filename := time.Now().Format(cfg.FilenameFormat) + ".png"
	savePath := filepath.Join(cfg.ScreenshotPath, filename)

	// Save file
	file, err := os.Create(savePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		return "", err
	}

	return savePath, nil
}

func processScreenshot(img *image.RGBA, cfg *Config) (*OCRResponse, error) {
	// Save locally first
	path, err := saveScreenshot(img, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to save screenshot: %v", err)
	}

	// Upload to API if enabled
	if cfg.API.Enabled && HasToken() {
		resp, err := UploadScreenshot(img, cfg)
		if err != nil {
			return nil, fmt.Errorf("saved to %s, upload failed: %v", path, err)
		}
		return resp, nil
	}

	return nil, nil
}
