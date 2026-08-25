package system

import "fmt"

type PaneSwitchReport struct {
	Pane          string
	Frames        int
	BlankFrames   int
	RemountFrames int
	Violations    []VisibilityViolation
}

func EvaluatePaneSwitchTrace(
	pane string,
	trace map[string]any,
	tabAddress, screenAddress, initialIdentity string,
) (PaneSwitchReport, error) {
	frames, ok := exactInt(trace["frames"])
	if !ok || frames < 1 {
		return PaneSwitchReport{}, fmt.Errorf("%s pane-switch trace has invalid frame count", pane)
	}
	samples, ok := trace["samples"].([]any)
	if !ok || len(samples) != frames {
		return PaneSwitchReport{}, fmt.Errorf("%s pane-switch trace has %d samples for %d frames", pane, len(samples), frames)
	}
	report := PaneSwitchReport{Pane: pane, Frames: frames, Violations: []VisibilityViolation{}}
	for _, value := range samples {
		sample, ok := value.(map[string]any)
		if !ok {
			return PaneSwitchReport{}, fmt.Errorf("%s pane-switch trace contains a non-object sample", pane)
		}
		frame, ok := exactInt(sample["captureFrame"])
		if !ok {
			return PaneSwitchReport{}, fmt.Errorf("%s pane-switch trace sample has no frame", pane)
		}
		nodes, _ := sample["nodes"].([]any)
		tab := findTraceNode(nodes, tabAddress)
		screen := findTraceNode(nodes, screenAddress)
		blankReasons := append(nodeBlankReasons("tab", tab), nodeBlankReasons("screen", screen)...)
		reasons := append([]string(nil), blankReasons...)
		identity, _ := tab["nodeIdentity"].(string)
		remounted := identity == "" || identity != initialIdentity
		if remounted {
			reasons = append(reasons, "tab.remounted")
		}
		if len(reasons) > 0 {
			report.Violations = append(report.Violations, VisibilityViolation{Frame: frame, Reasons: reasons})
		}
		if len(blankReasons) > 0 {
			report.BlankFrames++
		}
		if remounted {
			report.RemountFrames++
		}
	}
	return report, nil
}
