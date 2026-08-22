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
