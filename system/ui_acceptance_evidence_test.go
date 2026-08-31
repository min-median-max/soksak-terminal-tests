package system

import (
	"image"
	"image/color"
	"testing"
)

func TestReadWorkspaceTitleEvidenceRequiresTextAndVisibleRect(t *testing.T) {
	snapshot := map[string]any{
		"count": float64(1),
		"nodes": []any{map[string]any{
			"selector": ".workspace-tab-title",
			"text":     "workspace 1",
			"rect": map[string]any{
				"x": float64(20), "y": float64(12), "w": float64(88), "h": float64(24),
			},
		}},
	}

	evidence, err := readWorkspaceTitleEvidence(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Selector != workspaceTitleSelector || evidence.Count != 1 {
		t.Fatalf("workspace title identity = %+v", evidence)
	}
	if len(evidence.Nodes) != 1 || evidence.Nodes[0].Text != "workspace 1" {
		t.Fatalf("workspace title text = %+v", evidence.Nodes)
	}
	if evidence.Nodes[0].X != 20 || evidence.Nodes[0].Y != 12 || evidence.Nodes[0].Width != 88 || evidence.Nodes[0].Height != 24 {
		t.Fatalf("workspace title rectangle = %+v", evidence.Nodes[0])
	}
}

func TestMeasureWorkspaceTitlePixelsRequiresOpaqueContrast(t *testing.T) {
	pixels := image.NewRGBA(image.Rect(0, 0, 20, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 20; x++ {
			pixels.Set(x, y, color.RGBA{R: 240, G: 240, B: 240, A: 255})
		}
	}
	for y := 2; y < 8; y++ {
		for x := 6; x < 14; x++ {
			pixels.Set(x, y, color.RGBA{R: 20, G: 20, B: 20, A: 255})
		}
	}

	evidence, err := measureWorkspaceTitlePixels(pixels)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Width != 20 || evidence.Height != 10 || evidence.OpaquePixels != 200 || evidence.DistinctPixels != 2 || evidence.LuminanceRange != 220 {
		t.Fatalf("workspace title pixels = %+v", evidence)
	}

	uniform := image.NewRGBA(image.Rect(0, 0, 20, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 20; x++ {
			uniform.Set(x, y, color.RGBA{R: 240, G: 240, B: 240, A: 255})
		}
	}
	if _, err := measureWorkspaceTitlePixels(uniform); err == nil {
		t.Fatal("uniform workspace title capture passed")
	}
}

func TestReadWorkspaceTitleEvidenceRejectsEmptyTextAndZeroSize(t *testing.T) {
	cases := map[string]struct {
		text  string
		width float64
	}{
		"empty-text": {text: "", width: 88},
		"zero-width": {text: "workspace 1", width: 0},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := readWorkspaceTitleEvidence(map[string]any{
				"count": float64(1),
				"nodes": []any{map[string]any{
					"selector": workspaceTitleSelector,
					"text":     testCase.text,
					"rect": map[string]any{
						"x": float64(20), "y": float64(12), "w": testCase.width, "h": float64(24),
					},
				}},
			})
			if err == nil {
				t.Fatal("invalid workspace title evidence passed")
			}
		})
	}
}
