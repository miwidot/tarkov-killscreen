// icon.go - System Tray Icon
//
// Generates a 32x32 crosshair icon for the system tray notification area.
// The icon is created programmatically using orange/dark-orange colors
// to match the Tarkov aesthetic.
package main

import (
	"image"
	"image/color"
)

// createIconImage creates a 32x32 crosshair icon
func createIconImage() image.Image {
	size := 32
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	orange := color.RGBA{255, 140, 0, 255}
	darkOrange := color.RGBA{200, 100, 0, 255}
	transparent := color.RGBA{0, 0, 0, 0}

	center := size / 2

	// Fill transparent
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, transparent)
		}
	}

	// Draw outer ring
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := x - center
			dy := y - center
			dist := dx*dx + dy*dy
			if dist <= 14*14 && dist >= 12*12 {
				img.Set(x, y, darkOrange)
			}
		}
	}

	// Draw inner ring
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := x - center
			dy := y - center
			dist := dx*dx + dy*dy
			if dist <= 10*10 && dist >= 7*7 {
				img.Set(x, y, orange)
			}
		}
	}

	// Center dot
	for y := center - 2; y <= center+2; y++ {
		for x := center - 2; x <= center+2; x++ {
			dx := x - center
			dy := y - center
			if dx*dx+dy*dy <= 4 {
				img.Set(x, y, orange)
			}
		}
	}

	// Crosshair lines
	for i := 0; i < 6; i++ {
		img.Set(center, 1+i, orange)         // Top
		img.Set(center, size-2-i, orange)    // Bottom
		img.Set(1+i, center, orange)         // Left
		img.Set(size-2-i, center, orange)    // Right
	}

	return img
}

// createUpdateIconImage creates a 32x32 icon with a green dot to indicate
// an available update. Used for blinking alternation with the normal icon.
func createUpdateIconImage() image.Image {
	size := 32
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	green := color.RGBA{0, 200, 0, 255}
	darkGreen := color.RGBA{0, 150, 0, 255}
	transparent := color.RGBA{0, 0, 0, 0}

	center := size / 2

	// Fill transparent
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, transparent)
		}
	}

	// Draw outer ring (green)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := x - center
			dy := y - center
			dist := dx*dx + dy*dy
			if dist <= 14*14 && dist >= 12*12 {
				img.Set(x, y, darkGreen)
			}
		}
	}

	// Draw filled circle (green)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := x - center
			dy := y - center
			dist := dx*dx + dy*dy
			if dist <= 10*10 {
				img.Set(x, y, green)
			}
		}
	}

	return img
}
