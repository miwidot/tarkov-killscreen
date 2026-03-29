//go:build ignore

package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

func main() {
	sizes := []int{256, 48, 32, 16}
	names := []string{"winres/icon256.png", "winres/icon48.png", "winres/icon.png", "winres/icon16.png"}

	for i, size := range sizes {
		img := createIcon(size)
		f, _ := os.Create(names[i])
		png.Encode(f, img)
		f.Close()
	}
}

func createIcon(size int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	orange := color.RGBA{255, 140, 0, 255}
	darkOrange := color.RGBA{200, 100, 0, 255}

	center := float64(size) / 2
	scale := float64(size) / 32.0

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - center
			dy := float64(y) - center
			dist := dx*dx + dy*dy

			outerMax := 14 * scale
			outerMin := 12 * scale
			innerMax := 10 * scale
			innerMin := 7 * scale
			dotR := 2 * scale

			if dist <= outerMax*outerMax && dist >= outerMin*outerMin {
				img.Set(x, y, darkOrange)
			} else if dist <= innerMax*innerMax && dist >= innerMin*innerMin {
				img.Set(x, y, orange)
			} else if dist <= dotR*dotR {
				img.Set(x, y, orange)
			}
		}
	}

	// Crosshair lines
	lineLen := int(6 * scale)
	c := int(center)
	lineW := int(scale + 0.5)
	if lineW < 1 {
		lineW = 1
	}
	for w := -lineW/2; w <= lineW/2; w++ {
		for i := 0; i < lineLen; i++ {
			img.Set(c+w, int(1*scale)+i, orange)
			img.Set(c+w, size-int(2*scale)-i, orange)
			img.Set(int(1*scale)+i, c+w, orange)
			img.Set(size-int(2*scale)-i, c+w, orange)
		}
	}

	return img
}
