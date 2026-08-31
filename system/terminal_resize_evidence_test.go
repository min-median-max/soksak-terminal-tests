package system

import "testing"

func TestTerminalNodeAddressRequiresTheSelectedPluginViewAndNode(t *testing.T) {
	tree := map[string]any{"nodes": []any{
		map[string]any{"address": "win/w/view/plugin-a.content/tab/view-a/node/terminal-root", "nodePath": "terminal-root"},
		map[string]any{"address": "win/w/view/plugin-a.content/tab/view-b/node/terminal-root", "nodePath": "terminal-root"},
	}}
	address, err := terminalNodeAddress(tree, "plugin-a", "view-b", "terminal-root")
	if err != nil || address != "win/w/view/plugin-a.content/tab/view-b/node/terminal-root" {
		t.Fatalf("address=%q err=%v", address, err)
	}
	if _, err := terminalNodeAddress(tree, "plugin-a", "missing", "terminal-root"); err == nil {
		t.Fatal("missing terminal node was accepted")
	}
}

func TestTerminalNodeAddressAcceptsPaneQualifiedDynamicNodes(t *testing.T) {
	tree := map[string]any{"nodes": []any{
		map[string]any{"address": "win/w/view/plugin-a.content/tab/view-a/node/terminal-screen/1", "nodePath": "terminal-screen/1"},
	}}
	address, err := terminalNodeAddress(tree, "plugin-a", "view-a", "terminal-screen")
	if err != nil || address != "win/w/view/plugin-a.content/tab/view-a/node/terminal-screen/1" {
		t.Fatalf("address=%q err=%v", address, err)
	}
}

func TestResizeEvidenceIdentifiesTheFirstFailedBoundary(t *testing.T) {
	evidence := terminalResizeEvidence{
		Wide:   terminalResizeSample{DOM: terminalDOMSize{Width: 720, Height: 400}, ReportedCols: 90, ReportedRows: 25},
		Narrow: terminalResizeSample{DOM: terminalDOMSize{Width: 432, Height: 400}, ReportedCols: 90, ReportedRows: 25},
	}
	if got := evidence.failureBoundary(); got != "plugin-size" {
		t.Fatalf("failure boundary = %q", got)
	}
	evidence.Narrow.ReportedCols = 54
	if got := evidence.failureBoundary(); got != "plugin-request" {
		t.Fatalf("failure boundary = %q", got)
	}
	evidence.Narrow.Requested = &terminalSize{Cols: 54, Rows: 25}
	if got := evidence.failureBoundary(); got != "source-observation" {
		t.Fatalf("failure boundary = %q", got)
	}
	evidence.Narrow.PTY = &terminalSequencedSize{terminalSize: terminalSize{Cols: 54, Rows: 25}, EventSequence: 7, OutputSequence: 100}
	if got := evidence.failureBoundary(); got != "recovery-observation" {
		t.Fatalf("failure boundary = %q", got)
	}
	evidence.Narrow.Recovery = &terminalRecoverySize{terminalSequencedSize: terminalSequencedSize{terminalSize: terminalSize{Cols: 54, Rows: 25}, EventSequence: 7, OutputSequence: 100}}
	evidence.Narrow.Recovery.Gaps = 1
	if got := evidence.failureBoundary(); got != "recovery-observation" {
		t.Fatalf("gap failure boundary = %q", got)
	}
	evidence.Narrow.Recovery.Gaps = 0
	evidence.Narrow.Recovery.OutputSequence = 99
	if got := evidence.failureBoundary(); got != "recovery-observation" {
		t.Fatalf("recovery output boundary = %q", got)
	}
	evidence.Narrow.Recovery.OutputSequence = 100
	evidence.Narrow.Rendered = &terminalRenderedSize{terminalSize: terminalSize{Cols: 90, Rows: 25}, OutputSequence: 100}
	if got := evidence.failureBoundary(); got != "rendered-frame" {
		t.Fatalf("failure boundary = %q", got)
	}
	evidence.Narrow.Rendered = &terminalRenderedSize{terminalSize: terminalSize{Cols: 54, Rows: 25}, OutputSequence: 100}
	evidence.Narrow.Rendered.OutputSequence = 101
	if got := evidence.failureBoundary(); got != "rendered-frame" {
		t.Fatalf("rendered output boundary = %q", got)
	}
	evidence.Narrow.Rendered.OutputSequence = 99
	evidence.Restored = &terminalResizeSample{DOM: evidence.Wide.DOM, ReportedCols: evidence.Wide.ReportedCols}
	if got := evidence.failureBoundary(); got != "" {
		t.Fatalf("failure boundary = %q", got)
	}
	evidence.Narrow.DOM.Width = 720
	if got := evidence.failureBoundary(); got != "core-layout" {
		t.Fatalf("failure boundary = %q", got)
	}
}

func TestResizeEvidenceUsesRecoveryForNativeSurfaceObservation(t *testing.T) {
	requested := &terminalSize{Cols: 54, Rows: 25}
	recovery := &terminalRecoverySize{terminalSequencedSize: terminalSequencedSize{
		terminalSize: *requested, EventSequence: 7, OutputSequence: 100,
	}}
	evidence := terminalResizeEvidence{
		Wide: terminalResizeSample{DOM: terminalDOMSize{Width: 720}, ReportedCols: 90},
		Narrow: terminalResizeSample{
			DOM: terminalDOMSize{Width: 432}, ReportedCols: 54, Requested: requested,
			Recovery: recovery,
			Rendered: &terminalRenderedSize{terminalSize: *requested, OutputSequence: 100},
		},
		Restored: &terminalResizeSample{DOM: terminalDOMSize{Width: 720}, ReportedCols: 90},
	}
	if got := evidence.failureBoundary(); got != "" {
		t.Fatalf("native surface boundary = %q", got)
	}
}
