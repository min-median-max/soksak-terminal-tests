package system

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

const workspaceTitleSelector = ".workspace-tab-title"

type workspaceTitleNodeEvidence struct {
	Text   string  `json:"text"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type workspaceTitleEvidence struct {
	Selector string                       `json:"selector"`
	Count    int                          `json:"count"`
	Nodes    []workspaceTitleNodeEvidence `json:"nodes"`
	Capture  *workspaceTitlePixelEvidence `json:"capture,omitempty"`
	Failure  string                       `json:"failure,omitempty"`
}

type workspaceTitlePixelEvidence struct {
	Width          int `json:"width"`
	Height         int `json:"height"`
	OpaquePixels   int `json:"opaquePixels"`
	DistinctPixels int `json:"distinctPixels"`
	LuminanceRange int `json:"luminanceRange"`
}

func measureWorkspaceTitlePixels(pixels image.Image) (workspaceTitlePixelEvidence, error) {
	bounds := pixels.Bounds()
	evidence := workspaceTitlePixelEvidence{Width: bounds.Dx(), Height: bounds.Dy()}
	if evidence.Width <= 0 || evidence.Height <= 0 {
		return evidence, fmt.Errorf("workspace title capture is empty")
	}
	colors := map[uint32]struct{}{}
	minimumLuminance := 255
	maximumLuminance := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r16, g16, b16, a16 := pixels.At(x, y).RGBA()
			r, g, b, a := uint32(r16>>8), uint32(g16>>8), uint32(b16>>8), uint32(a16>>8)
			colors[r<<24|g<<16|b<<8|a] = struct{}{}
			if a == 255 {
				evidence.OpaquePixels++
			}
			luminance := int((299*r + 587*g + 114*b) / 1000)
			if luminance < minimumLuminance {
				minimumLuminance = luminance
			}
			if luminance > maximumLuminance {
				maximumLuminance = luminance
			}
		}
	}
	evidence.DistinctPixels = len(colors)
	evidence.LuminanceRange = maximumLuminance - minimumLuminance
	total := evidence.Width * evidence.Height
	if evidence.OpaquePixels != total {
		return evidence, fmt.Errorf("workspace title capture has %d opaque pixels, expected %d", evidence.OpaquePixels, total)
	}
	if evidence.DistinctPixels < 2 || evidence.LuminanceRange < 32 {
		return evidence, fmt.Errorf("workspace title capture has no rendered text contrast: %+v", evidence)
	}
	return evidence, nil
}

func readWorkspaceTitleCapture(path string) (workspaceTitlePixelEvidence, error) {
	file, err := os.Open(path)
	if err != nil {
		return workspaceTitlePixelEvidence{}, fmt.Errorf("open workspace title capture: %w", err)
	}
	defer file.Close()
	pixels, err := png.Decode(file)
	if err != nil {
		return workspaceTitlePixelEvidence{}, fmt.Errorf("decode workspace title capture: %w", err)
	}
	return measureWorkspaceTitlePixels(pixels)
}

func readWorkspaceTitleEvidence(snapshot map[string]any) (workspaceTitleEvidence, error) {
	evidence := workspaceTitleEvidence{Selector: workspaceTitleSelector}
	count, ok := snapshot["count"].(float64)
	if !ok || count < 1 || count != float64(int(count)) {
		return evidence, fmt.Errorf("workspace title count is invalid: %v", snapshot["count"])
	}
	evidence.Count = int(count)
	nodes, ok := snapshot["nodes"].([]any)
	if !ok || len(nodes) != evidence.Count {
		return evidence, fmt.Errorf("workspace title node count is %d, expected %d", len(nodes), evidence.Count)
	}
	for index, value := range nodes {
		node, ok := value.(map[string]any)
		if !ok || node["selector"] != workspaceTitleSelector {
			return evidence, fmt.Errorf("workspace title node %d has invalid identity", index)
		}
		text, ok := node["text"].(string)
		if !ok || strings.TrimSpace(text) == "" {
			return evidence, fmt.Errorf("workspace title node %d has empty text", index)
		}
		rect, ok := node["rect"].(map[string]any)
		if !ok {
			return evidence, fmt.Errorf("workspace title node %d has no rectangle", index)
		}
		x, xOK := rect["x"].(float64)
		y, yOK := rect["y"].(float64)
		width, widthOK := rect["w"].(float64)
		height, heightOK := rect["h"].(float64)
		if !xOK || !yOK || !widthOK || !heightOK || width <= 0 || height <= 0 {
			return evidence, fmt.Errorf("workspace title node %d has invalid rectangle: %+v", index, rect)
		}
		evidence.Nodes = append(evidence.Nodes, workspaceTitleNodeEvidence{
			Text: text, X: x, Y: y, Width: width, Height: height,
		})
	}
	return evidence, nil
}

func writeWorkspaceTitleEvidence(directory string, evidence workspaceTitleEvidence) error {
	body, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return fmt.Errorf("encode workspace title evidence: %w", err)
	}
	return os.WriteFile(filepath.Join(directory, "workspace-title.json"), append(body, '\n'), 0o600)
}
