//go:build system

package system

import "testing"

func TestNodeAddressAcceptsDeclaredDynamicNodeSuffix(t *testing.T) {
	nodes := []any{
		map[string]any{"address": "plugin/terminal/tab/terminal-screen/1", "dataset": map[string]any{"node": "terminal-screen/1"}},
		map[string]any{"address": "plugin/terminal/tab/terminal-screen", "dataset": map[string]any{"node": "terminal-screen"}},
	}
	if got := nodeAddress(nodes, "terminal-screen", "plugin\x00tab"); got != "plugin/terminal/tab/terminal-screen" {
		t.Fatalf("exact address=%q", got)
	}
	if got := nodeAddress(nodes[:1], "terminal-screen", "plugin\x00tab"); got != "plugin/terminal/tab/terminal-screen/1" {
		t.Fatalf("dynamic address=%q", got)
	}
}
