package system

import (
	"encoding/json"
	"fmt"
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
	Failure  string                       `json:"failure,omitempty"`
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
