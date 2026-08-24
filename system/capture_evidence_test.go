package system

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestCaptureEvidenceRejectsUniformAndAcceptsDrawnPixels(t *testing.T) {
	uniform := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			uniform.Set(x, y, color.Black)
		}
	}
	uniformPath := writeCaptureFixture(t, "uniform.png", uniform)
	if err := validateCaptureEvidence(uniformPath); err == nil {
		t.Fatal("uniform capture was accepted")
	}

	drawn := image.NewRGBA(image.Rect(0, 0, 4, 4))
	drawn.Set(1, 1, color.White)
	if err := validateCaptureEvidence(writeCaptureFixture(t, "drawn.png", drawn)); err != nil {
		t.Fatal(err)
	}
}

func TestTerminalPaletteEvidenceRequiresAllThreePaintedSwatches(t *testing.T) {
	expected, err := terminalPaletteExpectationFromBase([]string{
		"#2e3436", "#cc0000", "#4e9a06", "#c4a000", "#3465a4", "#75507b", "#06989a", "#d3d7cf",
		"#555753", "#ef2929", "#8ae234", "#fce94f", "#729fcf", "#ad7fa8", "#34e2e2", "#eeeeec",
	})
	if err != nil {
		t.Fatal(err)
	}
	complete := image.NewRGBA(image.Rect(0, 0, 36, 12))
	paintBlock(complete, image.Rect(0, 0, 12, 12), color.RGBA{R: 0xcc, G: 0x00, B: 0x00, A: 255})
	paintBlock(complete, image.Rect(12, 0, 24, 12), color.RGBA{R: 0x4e, G: 0x9a, B: 0x06, A: 255})
	paintBlock(complete, image.Rect(24, 0, 36, 12), color.RGBA{R: 0x34, G: 0x65, B: 0xa4, A: 255})
	if _, err := validateTerminalPaletteEvidence(writeCaptureFixture(t, "palette.png", complete), expected); err != nil {
		t.Fatal(err)
	}

	blank := image.NewRGBA(image.Rect(0, 0, 36, 12))
	paintBlock(blank, blank.Bounds(), color.RGBA{R: 21, G: 22, B: 29, A: 255})
	blank.Set(1, 1, color.RGBA{R: 164, G: 169, B: 222, A: 255})
	if _, err := validateTerminalPaletteEvidence(writeCaptureFixture(t, "blank-terminal.png", blank), expected); err == nil {
		t.Fatal("blank terminal surrounded by drawn pixels was accepted")
	}
}

func TestTerminalPaletteEvidenceAcceptsOneSharedFocusLightingTransform(t *testing.T) {
	expected, err := terminalPaletteExpectationFromBase([]string{
		"#2e3436", "#cc0000", "#4e9a06", "#c4a000", "#3465a4", "#75507b", "#06989a", "#d3d7cf",
		"#555753", "#ef2929", "#8ae234", "#fce94f", "#729fcf", "#ad7fa8", "#34e2e2", "#eeeeec",
	})
	if err != nil {
		t.Fatal(err)
	}
	painted := image.NewRGBA(image.Rect(0, 0, 36, 12))
	paintBlock(painted, image.Rect(0, 0, 12, 12), color.RGBA{R: 0xbb, G: 0x27, B: 0x1a, A: 255})
	paintBlock(painted, image.Rect(12, 0, 24, 12), color.RGBA{R: 0x61, G: 0x98, B: 0x2d, A: 255})
	paintBlock(painted, image.Rect(24, 0, 36, 12), color.RGBA{R: 0x40, G: 0x64, B: 0x9f, A: 255})
	if _, err := validateTerminalPaletteEvidence(writeCaptureFixture(t, "focus-lit-palette.png", painted), expected); err != nil {
		t.Fatal(err)
	}
}

func paintBlock(target *image.RGBA, bounds image.Rectangle, value color.Color) {
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			target.Set(x, y, value)
		}
	}
}

func writeCaptureFixture(t *testing.T, name string, value image.Image) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, value); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}
