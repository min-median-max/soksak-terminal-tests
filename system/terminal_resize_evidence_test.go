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

func TestResizeEvidenceIdentifiesTheFirstFailedBoundary(t *testing.T) {
	evidence := terminalResizeEvidence{
		Wide:   terminalResizeSample{DOM: terminalDOMSize{Width: 720, Height: 400}, ReportedCols: 90, ReportedRows: 25},
		Narrow: terminalResizeSample{DOM: terminalDOMSize{Width: 432, Height: 400}, ReportedCols: 90, ReportedRows: 25},
	}
	if got := evidence.failureBoundary(); got != "plugin-size" {
		t.Fatalf("failure boundary = %q", got)
	}
	evidence.Narrow.DOM.Width = 720
	if got := evidence.failureBoundary(); got != "core-layout" {
		t.Fatalf("failure boundary = %q", got)
	}
}
