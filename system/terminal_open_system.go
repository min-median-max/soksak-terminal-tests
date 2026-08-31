//go:build system

package system

import (
	"fmt"

	"github.com/min-median-max/soksak-terminal-tests/fleet"
)

func openTerminalForScenario(cli CLI, plugin fleet.Plugin) (TerminalResult, error) {
	_, pane, err := activeWorkspacePane(cli)
	if err != nil {
		return TerminalResult{}, err
	}
	program := plugin.Program
	opened, err := cli.Call("tab.open", map[string]any{
		"pane": pane, "program": program, "mountTimeoutMs": 12000,
	})
	if err != nil {
		return TerminalResult{}, err
	}
	tab, _ := opened["tabId"].(string)
	if tab == "" {
		return TerminalResult{}, fmt.Errorf("%s opened no tab", plugin.ID)
	}
	if _, err := cli.Call("tab.activate", map[string]any{"tab": tab}); err != nil {
		return TerminalResult{}, err
	}
	ready, err := terminal(cli, plugin.ID, "wait", tab, map[string]any{
		"phase": "live", "timeoutMs": 12000,
	})
	if err != nil {
		return TerminalResult{}, err
	}
	terminalPane, _ := ready["pane"].(string)
	if terminalPane == "" {
		return TerminalResult{}, fmt.Errorf("%s live status returned no pane key: %+v", plugin.ID, ready)
	}
	return TerminalResult{Plugin: plugin.ID, View: tab, Pane: terminalPane}, nil
}
