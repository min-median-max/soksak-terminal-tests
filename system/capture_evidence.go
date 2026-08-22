package system

import (
	"fmt"
	"image/png"
	"os"
)

func validateCaptureEvidence(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	image, err := png.Decode(file)
	if err != nil {
		return fmt.Errorf("capture is not a PNG: %w", err)
	}
	bounds := image.Bounds()
	if bounds.Dx() < 2 || bounds.Dy() < 2 {
		return fmt.Errorf("capture extent is %dx%d", bounds.Dx(), bounds.Dy())
	}
	first := image.At(bounds.Min.X, bounds.Min.Y)
	r0, g0, b0, a0 := first.RGBA()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := image.At(x, y).RGBA()
			if r != r0 || g != g0 || b != b0 || a != a0 {
				return nil
			}
		}
	}
	return fmt.Errorf("capture contains one uniform color")
}
