// signature.go - Screenshot Signature for Re-Capture Detection
//
// Embeds a small signature in the blue channel of 8 pixels to:
// 1. Detect re-captures (screenshot of a screenshot)
// 2. Verify the image came from our tool
//
// Signature structure: [Magic 4 bytes] + [Hash 4 bytes] = 8 bytes = 8 pixels
package main

import (
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"time"
)

// Magic bytes: "TRKV" (Tarkov)
var signatureMagic = []byte{0x54, 0x52, 0x4B, 0x56}

// getSignaturePositions calculates 8 pixel positions based on image dimensions
func getSignaturePositions(width, height int) []image.Point {
	// X positions at different percentages across the width
	xFactors := []float64{0.1, 0.3, 0.5, 0.7, 0.2, 0.4, 0.6, 0.8}
	// Y position: 3 pixels from bottom
	y := height - 3

	positions := make([]image.Point, 8)
	for i, factor := range xFactors {
		x := int(float64(width) * factor)
		positions[i] = image.Point{X: x, Y: y}
	}
	return positions
}

// generateSignatureHash creates a 4-byte hash from image metadata
func generateSignatureHash(width, height int) []byte {
	// Combine timestamp, dimensions for uniqueness
	data := make([]byte, 16)
	binary.LittleEndian.PutUint64(data[0:8], uint64(time.Now().UnixNano()))
	binary.LittleEndian.PutUint32(data[8:12], uint32(width))
	binary.LittleEndian.PutUint32(data[12:16], uint32(height))

	hash := crc32.ChecksumIEEE(data)
	result := make([]byte, 4)
	binary.LittleEndian.PutUint32(result, hash)
	return result
}

// Tolerance for JPEG compression and display rendering artifacts
const signatureTolerance = 8

// HasSignature checks if an image already has our signature (re-capture detection)
func HasSignature(img image.Image) bool {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if height < 10 || width < 100 {
		return false
	}

	positions := getSignaturePositions(width, height)

	// Read magic bytes from first 4 positions (with tolerance for JPEG artifacts)
	matches := 0
	for i := 0; i < 4; i++ {
		pos := positions[i]
		x := bounds.Min.X + pos.X
		y := bounds.Min.Y + pos.Y

		if x >= bounds.Max.X || y >= bounds.Max.Y {
			return false
		}

		_, _, b, _ := img.At(x, y).RGBA()
		blueValue := byte(b >> 8)
		expected := signatureMagic[i]

		// Check with tolerance (JPEG compression changes values slightly)
		diff := int(blueValue) - int(expected)
		if diff < 0 {
			diff = -diff
		}

		if diff <= signatureTolerance {
			matches++
			debugLog("[SIGNATURE] Pos %d: expected 0x%02X, got 0x%02X (diff=%d) ✓\n", i, expected, blueValue, diff)
		} else {
			debugLog("[SIGNATURE] Pos %d: expected 0x%02X, got 0x%02X (diff=%d) ✗\n", i, expected, blueValue, diff)
		}
	}

	// Need at least 3 of 4 magic bytes to match (allows for some corruption)
	if matches >= 3 {
		debugLog("[SIGNATURE] Re-capture detected! (%d/4 magic bytes matched)\n", matches)
		return true
	}

	return false
}

// EmbedSignature adds our signature to the image
// Returns a new image with the signature embedded
func EmbedSignature(img image.Image) *image.RGBA {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Create a copy as RGBA
	rgba := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}

	positions := getSignaturePositions(width, height)
	hash := generateSignatureHash(width, height)

	// Full signature: magic + hash
	signature := append(signatureMagic, hash...)

	// Embed each byte in the blue channel
	for i, pos := range positions {
		x := bounds.Min.X + pos.X
		y := bounds.Min.Y + pos.Y

		if x >= bounds.Max.X || y >= bounds.Max.Y {
			continue
		}

		original := rgba.At(x, y)
		r, g, _, a := original.RGBA()

		// Set blue channel to signature byte
		newColor := color.RGBA{
			R: uint8(r >> 8),
			G: uint8(g >> 8),
			B: signature[i],
			A: uint8(a >> 8),
		}
		rgba.Set(x, y, newColor)
	}

	debugLog("[SIGNATURE] Embedded signature at y=%d\n", height-3)
	return rgba
}

// VerifySignature checks if an image has our signature (for upload verification)
func VerifySignature(img image.Image) bool {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if height < 10 || width < 100 {
		return false
	}

	positions := getSignaturePositions(width, height)

	// Check magic bytes
	for i := 0; i < 4; i++ {
		pos := positions[i]
		x := bounds.Min.X + pos.X
		y := bounds.Min.Y + pos.Y

		if x >= bounds.Max.X || y >= bounds.Max.Y {
			return false
		}

		_, _, b, _ := img.At(x, y).RGBA()
		blueValue := byte(b >> 8)

		if blueValue != signatureMagic[i] {
			debugLog("[SIGNATURE] Verify failed at pos %d: expected 0x%02X, got 0x%02X\n",
				i, signatureMagic[i], blueValue)
			return false
		}
	}

	return true
}
