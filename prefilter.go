// prefilter.go - Local Kill Screen Detection
//
// Pre-filters screenshots before upload to save API costs.
// Uses two fast checks:
// 1. Edge Density - Kill screens have lots of text = many edges
// 2. Stripe Pattern - Kill list has alternating row colors
package main

import (
	"fmt"
	"image"
)

// PrefilterResult contains the analysis results
type PrefilterResult struct {
	IsLikelyKillScreen bool
	EdgeDensity        float64
	StripeScore        float64
	Reason             string
}

// PrefilterScreenshot checks if an image is likely a Tarkov kill screen
func PrefilterScreenshot(img image.Image) PrefilterResult {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Focus on the kill list area (right 60% of screen, middle 70% vertically)
	// This is where the kill list typically appears
	startX := bounds.Min.X + int(float64(width)*0.4)
	startY := bounds.Min.Y + int(float64(height)*0.15)
	endX := bounds.Max.X - int(float64(width)*0.05)
	endY := bounds.Max.Y - int(float64(height)*0.15)

	// 1. Edge Density Check
	edgeDensity := calculateEdgeDensity(img, startX, startY, endX, endY)

	// 2. Stripe Pattern Check
	stripeScore := calculateStripeScore(img, startX, startY, endX, endY)

	// Thresholds (tuned for Tarkov kill screens)
	edgeThreshold := 0.03      // Kill screens typically have 3-8% edge density
	stripeThreshold := 0.15    // Stripe alternation score

	isLikely := edgeDensity >= edgeThreshold && stripeScore >= stripeThreshold

	reason := ""
	if !isLikely {
		if edgeDensity < edgeThreshold {
			reason = fmt.Sprintf("Low edge density (%.1f%% < %.1f%%)", edgeDensity*100, edgeThreshold*100)
		} else {
			reason = fmt.Sprintf("No stripe pattern (%.2f < %.2f)", stripeScore, stripeThreshold)
		}
	}

	result := PrefilterResult{
		IsLikelyKillScreen: isLikely,
		EdgeDensity:        edgeDensity,
		StripeScore:        stripeScore,
		Reason:             reason,
	}

	fmt.Printf("[PREFILTER] Edge: %.2f%%, Stripes: %.2f, Likely: %v\n",
		edgeDensity*100, stripeScore, isLikely)

	return result
}

// calculateEdgeDensity counts horizontal edges (text creates many horizontal transitions)
func calculateEdgeDensity(img image.Image, x1, y1, x2, y2 int) float64 {
	edgeCount := 0
	totalPixels := 0
	threshold := uint32(20 << 8) // Brightness difference threshold

	for y := y1; y < y2-1; y++ {
		for x := x1; x < x2; x++ {
			// Get grayscale values
			r1, g1, b1, _ := img.At(x, y).RGBA()
			r2, g2, b2, _ := img.At(x, y+1).RGBA()

			gray1 := (r1 + g1 + b1) / 3
			gray2 := (r2 + g2 + b2) / 3

			// Check for edge (brightness change)
			var diff uint32
			if gray1 > gray2 {
				diff = gray1 - gray2
			} else {
				diff = gray2 - gray1
			}

			if diff > threshold {
				edgeCount++
			}
			totalPixels++
		}
	}

	if totalPixels == 0 {
		return 0
	}
	return float64(edgeCount) / float64(totalPixels)
}

// calculateStripeScore detects alternating brightness bands (kill list rows)
func calculateStripeScore(img image.Image, x1, y1, x2, y2 int) float64 {
	// Sample vertical brightness profile (average brightness per row)
	height := y2 - y1
	if height < 20 {
		return 0
	}

	// Calculate average brightness for each row
	rowBrightness := make([]float64, height)
	sampleWidth := (x2 - x1) / 10 // Sample 10% of width for speed

	for y := y1; y < y2; y++ {
		var sum uint64
		count := 0
		for x := x1; x < x2; x += sampleWidth {
			r, g, b, _ := img.At(x, y).RGBA()
			sum += uint64(r + g + b)
			count++
		}
		if count > 0 {
			rowBrightness[y-y1] = float64(sum) / float64(count*3)
		}
	}

	// Count alternations (sign changes in brightness delta)
	alternations := 0
	expectedRowHeight := 25 // Approximate pixel height of kill list rows
	windowSize := expectedRowHeight / 2

	for i := windowSize; i < height-windowSize; i += windowSize {
		// Compare average brightness of adjacent windows
		var avg1, avg2 float64
		for j := 0; j < windowSize; j++ {
			avg1 += rowBrightness[i-windowSize+j]
			avg2 += rowBrightness[i+j]
		}
		avg1 /= float64(windowSize)
		avg2 /= float64(windowSize)

		// Check for significant brightness change
		diff := avg1 - avg2
		if diff < 0 {
			diff = -diff
		}
		if diff > 500 { // Threshold for "significant" change
			alternations++
		}
	}

	// Normalize by expected number of alternations
	expectedAlternations := float64(height) / float64(expectedRowHeight)
	if expectedAlternations == 0 {
		return 0
	}

	return float64(alternations) / expectedAlternations
}
