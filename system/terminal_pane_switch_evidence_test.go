package system

import "testing"

func TestPaneSwitchTraceRejectsAnyBlankOrRemountedFrame(t *testing.T) {
	trace := map[string]any{
		"frames": float64(3),
		"samples": []any{
			paneSwitchSample(0, "pane-2", "screen-2", "node-2", true, 600, 400, "visible", "1"),
			paneSwitchSample(1, "pane-2", "screen-2", "node-2-next", true, 600, 400, "hidden", "1"),
			paneSwitchSample(2, "pane-2", "screen-2", "node-2-next", false, 0, 0, "visible", "0"),
		},
	}
	report, err := EvaluatePaneSwitchTrace("pane-2", trace, "pane-2", "screen-2", "node-2")
	if err != nil {
		t.Fatal(err)
	}
	if report.BlankFrames != 2 || report.RemountFrames != 2 {
		t.Fatalf("pane switch report=%+v", report)
	}
}

func paneSwitchSample(
	frame int,
	tabAddress, screenAddress, identity string,
	connected bool,
	width, height float64,
	visibility, opacity string,
) map[string]any {
	return map[string]any{
		"captureFrame": float64(frame),
		"nodes": []any{
			map[string]any{
				"address": tabAddress, "connected": connected, "nodeIdentity": identity,
				"rect":    map[string]any{"w": width, "h": height},
				"style":   map[string]any{"display": "block", "visibility": visibility, "opacity": opacity},
				"dataset": map[string]any{"contentVisible": "true"},
			},
			map[string]any{
				"address": screenAddress, "connected": connected, "nodeIdentity": identity + "-screen",
				"rect":    map[string]any{"w": width, "h": height},
				"style":   map[string]any{"display": "block", "visibility": visibility, "opacity": opacity},
				"dataset": map[string]any{},
			},
		},
	}
}
