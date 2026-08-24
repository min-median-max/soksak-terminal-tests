package system

import "testing"

func TestVisibilityTraceCountsEveryBlankFrame(t *testing.T) {
	trace := map[string]any{
		"frames": float64(3),
		"samples": []any{
			map[string]any{"captureFrame": float64(0), "nodes": []any{
				nodeSample("tab", true, 800, 400, "block", "visible", "1", "true"),
				nodeSample("screen", true, 800, 400, "block", "visible", "1", ""),
			}},
			map[string]any{"captureFrame": float64(1), "nodes": []any{
				nodeSample("tab", true, 800, 400, "block", "hidden", "1", "true"),
				nodeSample("screen", true, 800, 400, "block", "hidden", "1", ""),
			}},
			map[string]any{"captureFrame": float64(2), "nodes": []any{
				nodeSample("tab", true, 0, 400, "block", "visible", "1", "false"),
				nodeSample("screen", false, 0, 0, "block", "visible", "1", ""),
			}},
		},
	}
	report, err := EvaluateVisibilityTrace("vt100", "settings-open", trace, "tab", "screen")
	if err != nil {
		t.Fatal(err)
	}
	if report.Frames != 3 || report.BlankFrames != 2 {
		t.Fatalf("visibility report = %+v", report)
	}
	if len(report.Violations) != 2 || report.Violations[0].Frame != 1 || report.Violations[1].Frame != 2 {
		t.Fatalf("visibility violations = %+v", report.Violations)
	}
}

func nodeSample(
	address string,
	connected bool,
	width, height float64,
	display, visibility, opacity, contentVisible string,
) map[string]any {
	dataset := map[string]any{}
	if contentVisible != "" {
		dataset["contentVisible"] = contentVisible
	}
	return map[string]any{
		"address": address, "connected": connected,
		"rect":    map[string]any{"w": width, "h": height},
		"style":   map[string]any{"display": display, "visibility": visibility, "opacity": opacity},
		"dataset": dataset,
	}
}
