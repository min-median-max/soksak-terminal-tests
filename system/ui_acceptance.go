package system

import (
	"fmt"
	"path/filepath"
)

func VerifyInstalledUI(cli CLI) error {
	plugins, err := cli.Call("plugin.list", map[string]any{})
	if err != nil {
		return err
	}
	rejected, _ := plugins["rejected"].([]any)
	if len(rejected) != 0 {
		return fmt.Errorf("plugin loader rejected %d plugins: %+v", len(rejected), rejected)
	}
	verified, err := cli.Call("ui.verify", map[string]any{})
	if err != nil {
		return err
	}
	failed, _ := verified["failed"].(float64)
	unanswered, _ := verified["unanswered"].(float64)
	if verified["passed"] != true || failed != 0 || unanswered != 0 {
		return fmt.Errorf("ui.verify failed: %+v", verified)
	}
	composition, err := cli.Call("surface.composition", map[string]any{})
	if err != nil {
		return err
	}
	for _, field := range []string{"displaced", "unapplied", "undeclared", "misparented"} {
		values, _ := composition[field].([]any)
		if len(values) != 0 {
			return fmt.Errorf("surface.composition %s=%+v", field, values)
		}
	}
	surfaces, _ := composition["surfaces"].([]any)
	for _, value := range surfaces {
		surface, _ := value.(map[string]any)
		covered, _ := surface["coveredFraction"].(float64)
		if covered != 0 {
			return fmt.Errorf("surface %v is %.2f%% covered", surface["id"], covered*100)
		}
	}
	alignment, err := cli.Call("layout.alignment", map[string]any{})
	if err != nil {
		return err
	}
	off, _ := alignment["worstOff"].(float64)
	if off > 2 {
		return fmt.Errorf("surface alignment is off by %.2fpx", off)
	}
	titleSnapshot, err := cli.Call("ui.snapshot.dom", map[string]any{"selector": workspaceTitleSelector})
	if err != nil {
		return err
	}
	titleEvidence, titleErr := readWorkspaceTitleEvidence(titleSnapshot)
	if titleErr != nil {
		titleEvidence.Failure = titleErr.Error()
		_ = writeWorkspaceTitleEvidence(cli.EvidenceDir, titleEvidence)
		return titleErr
	}
	titleNode := titleEvidence.Nodes[0]
	capturePath := filepath.Join(cli.EvidenceDir, "workspace-title.png")
	_, captureErr := cli.Call("window.snapshot", map[string]any{
		"path": capturePath,
		"rect": map[string]any{
			"x": titleNode.X, "y": titleNode.Y, "w": titleNode.Width, "h": titleNode.Height,
		},
	})
	if captureErr == nil {
		var pixels workspaceTitlePixelEvidence
		pixels, captureErr = readWorkspaceTitleCapture(capturePath)
		if captureErr == nil {
			titleEvidence.Capture = &pixels
		}
	}
	if captureErr != nil {
		titleEvidence.Failure = captureErr.Error()
	}
	if err := writeWorkspaceTitleEvidence(cli.EvidenceDir, titleEvidence); err != nil {
		return err
	}
	if captureErr != nil {
		return captureErr
	}
	return nil
}
