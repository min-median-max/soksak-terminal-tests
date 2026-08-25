package system

import (
	"image"
	"image/color"
	"testing"
)

func TestTerminalThemePixelsRejectDifferentRenderedDefaults(t *testing.T) {
	left := image.NewRGBA(image.Rect(0, 0, 40, 20))
	right := image.NewRGBA(image.Rect(0, 0, 40, 20))
	paintThemeFixture(left, color.RGBA{R: 21, G: 22, B: 30, A: 255})
	paintThemeFixture(right, color.RGBA{R: 28, G: 26, B: 22, A: 255})

	baseline, err := readTerminalThemePixels(left)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := readTerminalThemePixels(right)
	if err != nil {
		t.Fatal(err)
	}
	if err := compareTerminalThemePixels(actual, baseline); err == nil {
		t.Fatalf("different rendered defaults were accepted: left=%+v right=%+v", baseline, actual)
	}
}

func paintThemeFixture(target *image.RGBA, background color.RGBA) {
	for y := target.Bounds().Min.Y; y < target.Bounds().Max.Y; y++ {
		for x := target.Bounds().Min.X; x < target.Bounds().Max.X; x++ {
			target.Set(x, y, background)
		}
	}
	for x := 4; x < 14; x++ {
		for y := 4; y < 8; y++ {
			target.Set(x, y, color.RGBA{R: 237, G: 238, B: 245, A: 255})
		}
	}
	for x := 20; x < 26; x++ {
		for y := 10; y < 16; y++ {
			target.Set(x, y, color.RGBA{R: 144, G: 151, B: 255, A: 255})
		}
	}
}
