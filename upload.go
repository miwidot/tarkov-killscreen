// upload.go - HTTP Upload to OCR API
//
// This file handles all communication with our backend API:
// - UploadScreenshot: Upload single image for OCR analysis
// - UploadMultipleScreenshots: Upload batch of images
// - SaveKills: Save analyzed kills to user's database
//
// All communication is HTTPS to our own server (tarkov-stammtisch.de).
// No game servers or third-party services are contacted.
//
// The API flow is:
// 1. Upload screenshot(s) to /api/ocr endpoint
// 2. Server uses OCR to extract kill information
// 3. Client receives structured kill data
// 4. Client sends kill data to /api/kills/save endpoint
// 5. Kills are stored in user's profile
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"golang.org/x/image/draw"
)

type KillData struct {
	Number   int     `json:"number"`
	Location string  `json:"location"`
	Time     string  `json:"time"`
	Player   string  `json:"player"`
	Level    int     `json:"level"`
	Faction  string  `json:"faction"`
	Weapon   string  `json:"weapon"`
	BodyPart string  `json:"bodyPart"`
	Distance float64 `json:"distance"`
	Status   string  `json:"status"`
}

type KillSummary struct {
	PMCKills      int                `json:"pmcKills"`
	ScavKills     int                `json:"scavKills"`
	BossKills     int                `json:"bossKills"`
	GuardKills    int                `json:"guardKills"`
	CultistKills  int                `json:"cultistKills"`
	SniperKills   int                `json:"sniperKills"`
	Headshots     int                `json:"headshots"`
	TotalDistance float64            `json:"totalDistance"`
	DistanceCount int                `json:"distanceCount"`
	Weapons       map[string]int     `json:"weapons"`
	RaidDuration  string             `json:"raidDuration"`
}

type OCRData struct {
	TotalKills int         `json:"totalKills"`
	Kills      []KillData  `json:"kills"`
	Summary    KillSummary `json:"summary"`
}

type InvalidDetail struct {
	Index  int    `json:"index"`
	Hash   string `json:"hash"`
	Reason string `json:"reason"`
}

type Validation struct {
	ValidImages    int             `json:"validImages"`
	InvalidImages  int             `json:"invalidImages"`
	AllValid       bool            `json:"allValid"`
	InvalidDetails []InvalidDetail `json:"invalidDetails"`
}

type ImageHash struct {
	Index     int    `json:"index"`
	Hash      string `json:"hash"`
	KillCount int    `json:"killCount"`
	IsValid   bool   `json:"isValid"`
	ImagePath string `json:"imagePath,omitempty"`
}

type ImagesInfo struct {
	Total            int         `json:"total"`
	Processed        int         `json:"processed"`
	DuplicateImages  int         `json:"duplicateImages"`
	AlreadyProcessed int         `json:"alreadyProcessed"`
	Hashes           []ImageHash `json:"hashes"`
}

type OCRResponse struct {
	Success    bool        `json:"success"`
	Mode       string      `json:"mode"`
	Data       OCRData     `json:"data"`
	Validation *Validation `json:"validation,omitempty"`
	Images     *ImagesInfo `json:"images,omitempty"`
	Error      string      `json:"error,omitempty"`
	Usage      struct {
		InputTokens  int `json:"inputTokens"`
		OutputTokens int `json:"outputTokens"`
	} `json:"usage"`
}

// Save request types
type SaveImageHash struct {
	Hash      string `json:"hash"`
	FileName  string `json:"fileName"`
	ImagePath string `json:"imagePath,omitempty"`
}

type SaveKillsRequest struct {
	Map          string            `json:"map"`
	RaidDate     string            `json:"raidDate"`
	TotalKills   int               `json:"totalKills"`
	PMCKills     int               `json:"pmcKills"`
	ScavKills    int               `json:"scavKills"`
	BossKills    int               `json:"bossKills"`
	GuardKills   int               `json:"guardKills"`
	CultistKills int               `json:"cultistKills"`
	SniperKills  int               `json:"sniperKills"`
	Headshots    int               `json:"headshots"`
	Weapons      map[string]int    `json:"weapons"`
	Kills        []KillData        `json:"kills"`
	ImageHashes  []SaveImageHash   `json:"imageHashes"`
}

type SaveKillsResponse struct {
	Success bool   `json:"success"`
	RaidID  string `json:"raidId"`
	Error   string `json:"error,omitempty"`
}

func compressImage(img image.Image, cfg *Config) ([]byte, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Scale down if wider than MaxWidth
	if width > cfg.API.MaxWidth {
		ratio := float64(cfg.API.MaxWidth) / float64(width)
		newWidth := cfg.API.MaxWidth
		newHeight := int(float64(height) * ratio)

		dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
		draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
		img = dst
	}

	// Encode as JPEG
	var buf bytes.Buffer
	opts := &jpeg.Options{Quality: cfg.API.JPEGQuality}
	if err := jpeg.Encode(&buf, img, opts); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func UploadScreenshot(img image.Image, cfg *Config) (*OCRResponse, error) {
	if !cfg.API.Enabled {
		return nil, fmt.Errorf("API upload disabled")
	}

	// Load token from credential manager
	token, err := LoadToken()
	if err != nil || token == "" {
		return nil, fmt.Errorf("no API token configured - please set token first")
	}

	// Trim any whitespace that might have been added
	token = strings.TrimSpace(token)

	fmt.Printf("[UPLOAD] Token length: %d, first 4: %s, last 4: %s\n", len(token), token[:4], token[len(token)-4:])
	fmt.Printf("[UPLOAD] URL: %s, Mode: %s\n", cfg.API.URL, cfg.API.Mode)

	// Compress image
	imageData, err := compressImage(img, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to compress image: %v", err)
	}

	// Create multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add mode field
	if err := writer.WriteField("mode", cfg.API.Mode); err != nil {
		return nil, err
	}

	// Add image file
	part, err := writer.CreateFormFile("image", "screenshot.jpg")
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(imageData); err != nil {
		return nil, err
	}

	writer.Close()

	// Create request
	req, err := http.NewRequest("POST", cfg.API.URL, &body)
	if err != nil {
		return nil, err
	}

	authHeader := "Bearer " + token
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	fmt.Printf("[UPLOAD] Auth header: Bearer %s...%s\n", token[:4], token[len(token)-4:])

	// Send request with timeout
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	fmt.Printf("[UPLOAD] Response status: %d\n", resp.StatusCode)
	fmt.Printf("[UPLOAD] Response body: %s\n", string(respBody))

	// Parse response
	var ocrResp OCRResponse
	if err := json.Unmarshal(respBody, &ocrResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	// Check for errors
	if resp.StatusCode != 200 {
		if ocrResp.Error != "" {
			return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, ocrResp.Error)
		}
		return nil, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	return &ocrResp, nil
}

// UploadMultipleScreenshots uploads multiple images in one request
func UploadMultipleScreenshots(images []image.Image, cfg *Config) (*OCRResponse, error) {
	if !cfg.API.Enabled {
		return nil, fmt.Errorf("API upload disabled")
	}

	if len(images) == 0 {
		return nil, fmt.Errorf("no images to upload")
	}

	token, err := LoadToken()
	if err != nil || token == "" {
		return nil, fmt.Errorf("no API token configured")
	}
	token = strings.TrimSpace(token)

	fmt.Printf("[UPLOAD] Uploading %d images...\n", len(images))

	// Create multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add mode field
	if err := writer.WriteField("mode", cfg.API.Mode); err != nil {
		return nil, err
	}

	// Add all images
	for i, img := range images {
		imageData, err := compressImage(img, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to compress image %d: %v", i, err)
		}

		// Use "images" field for multiple images
		part, err := writer.CreateFormFile("images", fmt.Sprintf("screenshot_%d.jpg", i))
		if err != nil {
			return nil, err
		}
		if _, err := part.Write(imageData); err != nil {
			return nil, err
		}
	}

	writer.Close()

	req, err := http.NewRequest("POST", cfg.API.URL, &body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 60 * time.Second} // Longer timeout for multiple images
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	fmt.Printf("[UPLOAD] Response status: %d\n", resp.StatusCode)
	fmt.Printf("[UPLOAD] Response body: %s\n", string(respBody))

	var ocrResp OCRResponse
	if err := json.Unmarshal(respBody, &ocrResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	if resp.StatusCode != 200 {
		if ocrResp.Error != "" {
			return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, ocrResp.Error)
		}
		return nil, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	return &ocrResp, nil
}

func FormatKillSummary(resp *OCRResponse) string {
	if resp == nil || !resp.Success {
		return "No kills detected"
	}

	// Check validation
	if resp.Validation != nil && !resp.Validation.AllValid && len(resp.Validation.InvalidDetails) > 0 {
		return resp.Validation.InvalidDetails[0].Reason
	}

	d := resp.Data
	s := d.Summary

	result := fmt.Sprintf("Kills: %d (PMC: %d, Scav: %d", d.TotalKills, s.PMCKills, s.ScavKills)

	if s.BossKills > 0 {
		result += fmt.Sprintf(", Boss: %d", s.BossKills)
	}
	if s.Headshots > 0 {
		result += fmt.Sprintf(", HS: %d", s.Headshots)
	}

	result += ")"

	return result
}

func IsValidTarkovScreenshot(resp *OCRResponse) bool {
	if resp == nil || resp.Validation == nil {
		return true // No validation info, assume valid
	}
	return resp.Validation.AllValid
}

// SaveKills saves the analyzed kills to the database
func SaveKills(ocrResp *OCRResponse, cfg *Config) (*SaveKillsResponse, error) {
	if ocrResp == nil || !ocrResp.Success || ocrResp.Data.TotalKills == 0 {
		return nil, fmt.Errorf("no kills to save")
	}

	token, err := LoadToken()
	if err != nil || token == "" {
		return nil, fmt.Errorf("no API token")
	}
	token = strings.TrimSpace(token)

	// Build image hashes from OCR response
	var imageHashes []SaveImageHash
	if ocrResp.Images != nil {
		for _, h := range ocrResp.Images.Hashes {
			if h.IsValid {
				imageHashes = append(imageHashes, SaveImageHash{
					Hash:      h.Hash,
					FileName:  fmt.Sprintf("screenshot_%d.jpg", h.Index),
					ImagePath: h.ImagePath,
				})
			}
		}
	}

	// Get map from first kill's location
	mapName := "Unknown"
	if len(ocrResp.Data.Kills) > 0 && ocrResp.Data.Kills[0].Location != "" {
		mapName = ocrResp.Data.Kills[0].Location
	}

	// Build save request
	saveReq := SaveKillsRequest{
		Map:          mapName,
		RaidDate:     time.Now().UTC().Format(time.RFC3339),
		TotalKills:   ocrResp.Data.TotalKills,
		PMCKills:     ocrResp.Data.Summary.PMCKills,
		ScavKills:    ocrResp.Data.Summary.ScavKills,
		BossKills:    ocrResp.Data.Summary.BossKills,
		GuardKills:   ocrResp.Data.Summary.GuardKills,
		CultistKills: ocrResp.Data.Summary.CultistKills,
		SniperKills:  ocrResp.Data.Summary.SniperKills,
		Headshots:    ocrResp.Data.Summary.Headshots,
		Weapons:      ocrResp.Data.Summary.Weapons,
		Kills:        ocrResp.Data.Kills,
		ImageHashes:  imageHashes,
	}

	jsonBody, err := json.Marshal(saveReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal save request: %v", err)
	}

	fmt.Printf("[SAVE] Saving %d kills to database...\n", ocrResp.Data.TotalKills)

	// Build save URL (replace /api/ocr with /api/kills/save)
	saveURL := strings.Replace(cfg.API.URL, "/api/ocr", "/api/kills/save", 1)

	req, err := http.NewRequest("POST", saveURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("save request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read save response: %v", err)
	}

	fmt.Printf("[SAVE] Response status: %d\n", resp.StatusCode)
	fmt.Printf("[SAVE] Response body: %s\n", string(respBody))

	var saveResp SaveKillsResponse
	if err := json.Unmarshal(respBody, &saveResp); err != nil {
		return nil, fmt.Errorf("failed to parse save response: %v", err)
	}

	if resp.StatusCode != 200 || !saveResp.Success {
		if saveResp.Error != "" {
			return nil, fmt.Errorf("save error: %s", saveResp.Error)
		}
		return nil, fmt.Errorf("save failed: %d", resp.StatusCode)
	}

	fmt.Printf("[SAVE] Saved! RaidID: %s\n", saveResp.RaidID)
	return &saveResp, nil
}
