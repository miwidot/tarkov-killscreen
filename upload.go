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
	"net/textproto"
	"strings"
	"time"

)

// createImagePart creates a multipart form file part with Content-Type: image/jpeg
// (Go's default CreateFormFile uses application/octet-stream).
func createImagePart(writer *multipart.Writer, fieldName, filename string) (io.Writer, error) {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name=%q; filename=%q`, fieldName, filename))
	h.Set("Content-Type", "image/jpeg")
	return writer.CreatePart(h)
}

// Shared HTTP client with connection pooling for all API requests.
var apiClient = &http.Client{
	Timeout:   120 * time.Second,
	Transport: &http.Transport{MaxIdleConnsPerHost: 2},
}

// KillData represents a single kill entry returned by the OCR API.
type KillData struct {
	Number   int     `json:"number"`
	Location string  `json:"location"`
	Time     string  `json:"time"`
	Player   string  `json:"player"`
	Level    int     `json:"level"`
	Faction  string  `json:"faction"`
	Weapon   string  `json:"weapon"`
	BodyPart     string  `json:"bodyPart"`
	BodyPartSide string  `json:"bodyPartSide"`
	Distance     float64 `json:"distance"`
	Status   string  `json:"status"`
}

// KillSummary contains aggregated kill statistics for a raid.
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

// OCRData holds the parsed kill data from OCR analysis.
type OCRData struct {
	TotalKills int         `json:"totalKills"`
	Kills      []KillData  `json:"kills"`
	Summary    KillSummary `json:"summary"`
}

// InvalidDetail describes why a specific image was rejected by the server.
type InvalidDetail struct {
	Index  int    `json:"index"`
	Hash   string `json:"hash"`
	Reason string `json:"reason"`
}

// Validation contains server-side image validation results.
type Validation struct {
	ValidImages    int             `json:"validImages"`
	InvalidImages  int             `json:"invalidImages"`
	AllValid       bool            `json:"allValid"`
	InvalidDetails []InvalidDetail `json:"invalidDetails"`
}

// ImageHash identifies a processed image by its server-computed hash.
type ImageHash struct {
	Index     int    `json:"index"`
	Hash      string `json:"hash"`
	KillCount int    `json:"killCount"`
	IsValid   bool   `json:"isValid"`
	ImagePath string `json:"imagePath,omitempty"`
}

// ImagesInfo summarizes how the server processed the uploaded images.
type ImagesInfo struct {
	Total            int         `json:"total"`
	Processed        int         `json:"processed"`
	DuplicateImages  int         `json:"duplicateImages"`
	AlreadyProcessed int         `json:"alreadyProcessed"`
	Hashes           []ImageHash `json:"hashes"`
}

// OCRResponse is the top-level response from the /api/ocr endpoint.
type OCRResponse struct {
	Success    bool        `json:"success"`
	Status     string      `json:"status,omitempty"`
	Message    string      `json:"message,omitempty"`
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

// SaveImageHash links an image hash to its filename for the save request.
type SaveImageHash struct {
	Hash      string `json:"hash"`
	FileName  string `json:"fileName"`
	ImagePath string `json:"imagePath,omitempty"`
}

// SaveKillsRequest is the payload sent to /api/kills/save to persist a raid.
type SaveKillsRequest struct {
	Map            string            `json:"map"`
	RaidDate       string            `json:"raidDate"`
	TotalKills     int               `json:"totalKills"`
	PMCKills       int               `json:"pmcKills"`
	ScavKills      int               `json:"scavKills"`
	BossKills      int               `json:"bossKills"`
	GuardKills     int               `json:"guardKills"`
	CultistKills   int               `json:"cultistKills"`
	SniperKills    int               `json:"sniperKills"`
	Headshots      int               `json:"headshots"`
	Weapons        map[string]int    `json:"weapons"`
	Kills          []KillData        `json:"kills"`
	ImageHashes    []SaveImageHash   `json:"imageHashes"`
	ClientVersion  string            `json:"clientVersion"`
	KillEventID    string            `json:"killEventId,omitempty"`
}

// SaveKillsResponse is the server's reply after saving a raid.
type SaveKillsResponse struct {
	Success bool   `json:"success"`
	RaidID  string `json:"raidId"`
	Error   string `json:"error,omitempty"`
}

func compressImage(img image.Image, cfg *Config) ([]byte, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Scale down if wider than MaxWidth (bilinear, no intermediate float64 buffer)
	if width > cfg.API.MaxWidth {
		ratio := float64(cfg.API.MaxWidth) / float64(width)
		newWidth := cfg.API.MaxWidth
		newHeight := int(float64(height) * ratio)
		if rgbaImg, ok := img.(*image.RGBA); ok {
			img = scaleDownBilinear(rgbaImg, newWidth, newHeight)
		}
	}

	// Encode as JPEG with pre-allocated buffer
	bounds = img.Bounds()
	var buf bytes.Buffer
	buf.Grow(bounds.Dx() * bounds.Dy() / 4) // ~25% of raw pixels as heuristic
	opts := &jpeg.Options{Quality: cfg.API.JPEGQuality}
	if err := jpeg.Encode(&buf, img, opts); err != nil {
		return nil, err
	}

	// Exact-fit copy so the oversized buffer backing array can be GC'd
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

// UploadScreenshotData uploads pre-compressed JPEG bytes to the OCR API.
func UploadScreenshotData(jpegData []byte, cfg *Config) (*OCRResponse, error) {
	if !cfg.API.Enabled {
		return nil, fmt.Errorf("API upload disabled")
	}

	token, err := LoadToken()
	if err != nil || token == "" {
		return nil, fmt.Errorf("no API token configured - please set token first")
	}
	token = strings.TrimSpace(token)

	debugLog("[UPLOAD] Token length: %d, first 4: %s, last 4: %s\n", len(token), token[:4], token[len(token)-4:])
	debugLog("[UPLOAD] URL: %s, Mode: %s\n", APIURL, cfg.API.Mode)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("mode", cfg.API.Mode); err != nil {
		return nil, err
	}
	if err := writer.WriteField("client_version", CurrentVersion); err != nil {
		return nil, err
	}

	part, err := createImagePart(writer, "image", "screenshot.jpg")
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(jpegData); err != nil {
		return nil, err
	}

	writer.Close()

	req, err := http.NewRequest("POST", APIURL, &body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	setDeviceHeader(req)

	resp, err := apiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	debugLog("[UPLOAD] Response status: %d\n", resp.StatusCode)
	debugLog("[UPLOAD] Response body: %s\n", string(respBody))

	if resp.StatusCode == 403 {
		return nil, checkDeviceLockError(respBody)
	}

	if resp.StatusCode != 200 {
		return nil, parseAPIError(resp.StatusCode, respBody)
	}

	var ocrResp OCRResponse
	if err := json.Unmarshal(respBody, &ocrResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &ocrResp, nil
}

// UploadMultipleScreenshotData uploads multiple pre-compressed JPEG images in one request.
func UploadMultipleScreenshotData(jpegDatas [][]byte, cfg *Config) (*OCRResponse, error) {
	if !cfg.API.Enabled {
		return nil, fmt.Errorf("API upload disabled")
	}

	if len(jpegDatas) == 0 {
		return nil, fmt.Errorf("no images to upload")
	}

	token, err := LoadToken()
	if err != nil || token == "" {
		return nil, fmt.Errorf("no API token configured")
	}
	token = strings.TrimSpace(token)

	fmt.Printf("[UPLOAD] Uploading %d images...\n", len(jpegDatas))

	// Pre-allocate buffer based on known JPEG sizes + multipart overhead
	totalSize := 0
	for _, d := range jpegDatas {
		totalSize += len(d) + 256 // 256 bytes overhead per part header
	}
	var body bytes.Buffer
	body.Grow(totalSize)
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("mode", cfg.API.Mode); err != nil {
		return nil, err
	}
	if err := writer.WriteField("client_version", CurrentVersion); err != nil {
		return nil, err
	}

	for i, jpegData := range jpegDatas {
		part, err := createImagePart(writer, "images", fmt.Sprintf("screenshot_%d.jpg", i))
		if err != nil {
			return nil, err
		}
		if _, err := part.Write(jpegData); err != nil {
			return nil, err
		}
		jpegDatas[i] = nil // Free original bytes early
	}

	writer.Close()

	req, err := http.NewRequest("POST", APIURL, &body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	setDeviceHeader(req)

	resp, err := apiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	debugLog("[UPLOAD] Response status: %d\n", resp.StatusCode)
	debugLog("[UPLOAD] Response body: %s\n", string(respBody))

	if resp.StatusCode == 403 {
		return nil, checkDeviceLockError(respBody)
	}

	if resp.StatusCode != 200 {
		return nil, parseAPIError(resp.StatusCode, respBody)
	}

	var ocrResp OCRResponse
	if err := json.Unmarshal(respBody, &ocrResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
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
		ImageHashes:   imageHashes,
		ClientVersion: CurrentVersion,
		KillEventID:   GetSelectedEventID(),
	}

	jsonBody, err := json.Marshal(saveReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal save request: %v", err)
	}

	fmt.Printf("[SAVE] Saving %d kills...\n", ocrResp.Data.TotalKills) // User-facing status

	// Build save URL (replace /api/ocr with /api/kills/save)
	saveURL := strings.Replace(APIURL, "/api/ocr", "/api/kills/save", 1)

	req, err := http.NewRequest("POST", saveURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	setDeviceHeader(req)

	resp, err := apiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("save request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read save response: %v", err)
	}

	debugLog("[SAVE] Response status: %d\n", resp.StatusCode)
	debugLog("[SAVE] Response body: %s\n", string(respBody))

	if resp.StatusCode == 403 {
		return nil, checkDeviceLockError(respBody)
	}

	var saveResp SaveKillsResponse
	if err := json.Unmarshal(respBody, &saveResp); err != nil {
		return nil, fmt.Errorf("failed to parse save response: %v", err)
	}

	if resp.StatusCode != 200 || !saveResp.Success {
		if saveResp.Error != "" {
			// Check for event-related errors
			if IsEventError(saveResp.Error) {
				return nil, fmt.Errorf("event error: %s", saveResp.Error)
			}
			return nil, fmt.Errorf("save error: %s", saveResp.Error)
		}
		return nil, fmt.Errorf("save failed: %d", resp.StatusCode)
	}

	fmt.Printf("[SAVE] Saved! RaidID: %s\n", saveResp.RaidID) // User-facing success
	return &saveResp, nil
}

// parseAPIError extracts an error message from a non-200 API response.
// Handles mixed schemas where "status" can be string or number.
func parseAPIError(statusCode int, body []byte) error {
	var errResp struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
		return fmt.Errorf("API error (%d): %s", statusCode, errResp.Error)
	}
	return fmt.Errorf("API error: %d", statusCode)
}

// setDeviceHeader adds the X-Device-ID header to an outgoing request.
func setDeviceHeader(req *http.Request) {
	if id := GetDeviceID(); id != "" {
		req.Header.Set("X-Device-ID", id)
	}
}

// ErrDeviceLocked is returned when the server rejects the request because
// the token is bound to a different device.
var ErrDeviceLocked = fmt.Errorf("device_locked")

// checkDeviceLockError checks if a 403 response is a device lock error.
func checkDeviceLockError(respBody []byte) error {
	var errResp struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
		if strings.Contains(errResp.Error, "Gerät") || strings.Contains(errResp.Error, "device") {
			return ErrDeviceLocked
		}
		return fmt.Errorf("%s", errResp.Error)
	}
	return fmt.Errorf("access denied (403)")
}

// scaleDownBilinear scales an RGBA image down using bilinear interpolation.
// Unlike draw.CatmullRom, this uses no intermediate float64 buffer (~84MB savings at 2K).
func scaleDownBilinear(src *image.RGBA, newWidth, newHeight int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	srcW := src.Bounds().Dx()
	srcH := src.Bounds().Dy()

	for y := 0; y < newHeight; y++ {
		srcY := float64(y) * float64(srcH) / float64(newHeight)
		y0 := int(srcY)
		y1 := y0 + 1
		if y1 >= srcH {
			y1 = srcH - 1
		}
		fy := srcY - float64(y0)

		for x := 0; x < newWidth; x++ {
			srcX := float64(x) * float64(srcW) / float64(newWidth)
			x0 := int(srcX)
			x1 := x0 + 1
			if x1 >= srcW {
				x1 = srcW - 1
			}
			fx := srcX - float64(x0)

			si00 := y0*src.Stride + x0*4
			si10 := y0*src.Stride + x1*4
			si01 := y1*src.Stride + x0*4
			si11 := y1*src.Stride + x1*4
			di := y*dst.Stride + x*4

			for c := 0; c < 4; c++ {
				v := float64(src.Pix[si00+c])*(1-fx)*(1-fy) +
					float64(src.Pix[si10+c])*fx*(1-fy) +
					float64(src.Pix[si01+c])*(1-fx)*fy +
					float64(src.Pix[si11+c])*fx*fy
				dst.Pix[di+c] = byte(v + 0.5)
			}
		}
	}
	return dst
}
