package system

import (
	"fmt"
	"strconv"
)

type VisibilityViolation struct {
	Frame   int      `json:"frame"`
	Reasons []string `json:"reasons"`
}

type VisibilityReport struct {
	Provider    string                `json:"provider"`
	Transition  string                `json:"transition"`
	Frames      int                   `json:"frames"`
	BlankFrames int                   `json:"blankFrames"`
	Violations  []VisibilityViolation `json:"violations"`
}

func EvaluateVisibilityTrace(
	provider, transition string,
	trace map[string]any,
	tabAddress, screenAddress string,
) (VisibilityReport, error) {
	frames, ok := exactInt(trace["frames"])
	if !ok || frames < 1 {
		return VisibilityReport{}, fmt.Errorf("%s/%s trace has invalid frame count", provider, transition)
	}
	samples, ok := trace["samples"].([]any)
	if !ok || len(samples) != frames {
		return VisibilityReport{}, fmt.Errorf("%s/%s trace has %d samples for %d frames", provider, transition, len(samples), frames)
	}
	report := VisibilityReport{Provider: provider, Transition: transition, Frames: frames, Violations: []VisibilityViolation{}}
	for _, value := range samples {
		sample, ok := value.(map[string]any)
		if !ok {
			return VisibilityReport{}, fmt.Errorf("%s/%s trace contains a non-object sample", provider, transition)
		}
		frame, ok := exactInt(sample["captureFrame"])
		if !ok {
			return VisibilityReport{}, fmt.Errorf("%s/%s trace sample has no frame", provider, transition)
		}
		nodes, ok := sample["nodes"].([]any)
		if !ok {
			return VisibilityReport{}, fmt.Errorf("%s/%s frame %d has no nodes", provider, transition, frame)
		}
		tab := findTraceNode(nodes, tabAddress)
		screen := findTraceNode(nodes, screenAddress)
		reasons := append(nodeBlankReasons("tab", tab), nodeBlankReasons("screen", screen)...)
		if tab != nil {
			dataset, _ := tab["dataset"].(map[string]any)
			if dataset["contentVisible"] != "true" {
				reasons = append(reasons, "tab.contentVisible")
			}
		}
		if len(reasons) > 0 {
			report.BlankFrames++
			report.Violations = append(report.Violations, VisibilityViolation{Frame: frame, Reasons: reasons})
		}
	}
	return report, nil
}

func findTraceNode(nodes []any, address string) map[string]any {
	for _, value := range nodes {
		node, _ := value.(map[string]any)
		if node["address"] == address {
			return node
		}
	}
	return nil
}

func nodeBlankReasons(name string, node map[string]any) []string {
	if node == nil {
		return []string{name + ".missing"}
	}
	reasons := []string{}
	if node["connected"] != true {
		reasons = append(reasons, name+".disconnected")
	}
	rect, _ := node["rect"].(map[string]any)
	width, widthOK := number(rect["w"])
	height, heightOK := number(rect["h"])
	if !widthOK || !heightOK || width <= 0 || height <= 0 {
		reasons = append(reasons, name+".zeroArea")
	}
	style, _ := node["style"].(map[string]any)
	if style["display"] == "none" {
		reasons = append(reasons, name+".displayNone")
	}
	if style["visibility"] == "hidden" {
		reasons = append(reasons, name+".hidden")
	}
	opacity, opacityErr := strconv.ParseFloat(fmt.Sprint(style["opacity"]), 64)
	if opacityErr != nil || opacity == 0 {
		reasons = append(reasons, name+".transparent")
	}
	return reasons
}

func exactInt(value any) (int, bool) {
	number, ok := number(value)
	integer := int(number)
	return integer, ok && number == float64(integer)
}

func number(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case int:
		return float64(typed), true
	default:
		return 0, false
	}
}
