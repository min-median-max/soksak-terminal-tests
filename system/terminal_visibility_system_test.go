//go:build system

package system

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstalledTerminalVisibilityMatrix(t *testing.T) {
	profile := profileFromEnvironment(t)
	lifecycle, err := NewLifecycle(lifecycleConfigFromEnvironment(t, "visibility"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(lifecycle.Close)
	if err := lifecycle.Start(); err != nil {
		t.Fatal(err)
	}
	if err := InstallConfiguredTerminalFleet(profile, lifecycle.Client()); err != nil {
		t.Fatal(err)
	}
	if err := InstallPlugin("soksak-plugin-browser-wails3", lifecycle.Client()); err != nil {
		t.Fatal(err)
	}
	if _, err := lifecycle.OpenWorkspace(); err != nil {
		t.Fatal(err)
	}
	cli := lifecycle.Client()
	workspace, pane, err := activeWorkspacePane(cli)
	if err != nil {
		t.Fatal(err)
	}
	set, err := cli.Call("sections.create", map[string]any{"title": "Terminal visibility"})
	if err != nil {
		t.Fatal(err)
	}
	setID, _ := set["id"].(string)
	if setID == "" {
		t.Fatal("sections.create returned no set id")
	}
	if _, err := cli.Call("sections.arrange", map[string]any{
		"set": setID, "sections": []string{"soksak-plugin-browser-wails3.pages"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := cli.Call("sections.left", map[string]any{"set": setID}); err != nil {
		t.Fatal(err)
	}
	if _, err := cli.Call("workspace.region.toggle", map[string]any{
		"workspace": workspace, "region": "left", "open": false,
	}); err != nil {
		t.Fatal(err)
	}

	tabs := map[string]string{}
	for _, plugin := range profile.Plugins {
		program := strings.TrimPrefix(plugin.ID, "soksak-plugin-")
		opened, err := cli.Call("tab.open", map[string]any{
			"pane": pane, "program": program, "mountTimeoutMs": 12000,
		})
		if err != nil {
			t.Fatalf("open %s: %v", plugin.ID, err)
		}
		tab, _ := opened["tabId"].(string)
		if tab == "" {
			t.Fatalf("open %s returned no tab", plugin.ID)
		}
		tabs[plugin.ID] = tab
	}

	reports := []VisibilityReport{}
	for _, plugin := range profile.Plugins {
		tab := tabs[plugin.ID]
		if _, err := cli.Call("tab.activate", map[string]any{"tab": tab}); err != nil {
			t.Fatalf("activate %s: %v", plugin.ID, err)
		}
		if _, err := cli.Call("plugin."+plugin.ID+".wait", map[string]any{
			"phase": "live", "view": tab, "timeoutMs": 12000,
		}); err != nil {
			t.Fatalf("wait %s: %v", plugin.ID, err)
		}
		nodes, err := exposedNodes(cli)
		if err != nil {
			t.Fatal(err)
		}
		tabAddress := nodeAddress(nodes, "layout/tab/"+tab, "")
		screenAddress := nodeAddress(nodes, "terminal-screen", plugin.ID+"\x00"+tab)
		addAddress := nodeAddress(nodes, "tab/view/"+pane+"/add", "")
		settingsAddress := nodeAddress(nodes, "settings-open", "")
		leftAddress := nodeAddress(nodes, "titlebar/region/left", "")
		transitions := []struct{ name, trigger string }{
			{name: "picker-open", trigger: addAddress},
			{name: "settings-open", trigger: settingsAddress},
			{name: "sidebar-open", trigger: leftAddress},
		}
		for _, transition := range transitions {
			name, trigger := transition.name, transition.trigger
			if trigger == "" || tabAddress == "" || screenAddress == "" {
				t.Fatalf("%s/%s public nodes are incomplete", plugin.ID, name)
			}
			if name == "sidebar-open" {
				if _, err := cli.Call("workspace.region.toggle", map[string]any{
					"workspace": workspace, "region": "left", "open": false,
				}); err != nil {
					t.Fatal(err)
				}
			}
			report, err := recordVisibilityTransition(
				cli, lifecycle.config.EvidenceDir, plugin.ID, name,
				trigger, tabAddress, screenAddress,
			)
			if err != nil {
				t.Fatalf("%s/%s: %v", plugin.ID, name, err)
			}
			reports = append(reports, report)
			if report.BlankFrames != 0 {
				t.Errorf("%s/%s blank frames: %+v", plugin.ID, name, report.Violations)
			}
			switch name {
			case "picker-open":
				_, err = cli.Call("ui.input.key", map[string]any{"address": addAddress, "key": "Escape"})
			case "settings-open":
				next, treeErr := exposedNodes(cli)
				if treeErr != nil {
					err = treeErr
				} else {
					errAddress := nodeAddress(next, "settings/close", "")
					_, err = cli.Call("ui.input.click", map[string]any{"address": errAddress})
				}
			case "sidebar-open":
				_, err = cli.Call("workspace.region.toggle", map[string]any{
					"workspace": workspace, "region": "left", "open": false,
				})
			}
			if err != nil {
				t.Fatalf("close %s/%s: %v", plugin.ID, name, err)
			}
		}
	}
	body, err := json.MarshalIndent(map[string]any{"reports": reports}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lifecycle.config.EvidenceDir, "terminal-visibility.json"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.Finish(); err != nil {
		t.Fatal(err)
	}
}

func recordVisibilityTransition(
	cli CLI,
	evidenceDir, provider, transition, trigger, tabAddress, screenAddress string,
) (VisibilityReport, error) {
	dir := filepath.Join(evidenceDir, "visibility", provider, transition)
	receipt, err := cli.Call("ui.input.click", map[string]any{
		"address": trigger, "recordDir": dir, "recordFrames": 40, "recordIntervalMs": 16,
		"traceAddresses": []string{tabAddress, screenAddress},
		"causeTraceId":   "terminal-visibility-" + provider + "-" + transition,
	})
	if err != nil {
		return VisibilityReport{}, err
	}
	recording, _ := receipt["recording"].(map[string]any)
	if recording["status"] != "complete" {
		return VisibilityReport{}, fmt.Errorf("recording did not complete: %+v", recording)
	}
	trace, ok := receipt["trace"].(map[string]any)
	if !ok {
		return VisibilityReport{}, fmt.Errorf("click returned no finite trace")
	}
	return EvaluateVisibilityTrace(provider, transition, trace, tabAddress, screenAddress)
}

func exposedNodes(cli CLI) ([]any, error) {
	tree, err := cli.Call("ui.tree", map[string]any{"rects": true})
	if err != nil {
		return nil, err
	}
	nodes, ok := tree["nodes"].([]any)
	if !ok {
		return nil, fmt.Errorf("ui.tree returned no nodes")
	}
	return nodes, nil
}

func nodeAddress(nodes []any, nodePath, contains string) string {
	parts := strings.Split(contains, "\x00")
	for _, value := range nodes {
		node, _ := value.(map[string]any)
		dataset, _ := node["dataset"].(map[string]any)
		address, _ := node["address"].(string)
		if dataset["node"] != nodePath {
			continue
		}
		matched := true
		for _, part := range parts {
			if part != "" && !strings.Contains(address, part) {
				matched = false
			}
		}
		if matched {
			return address
		}
	}
	return ""
}

func activeWorkspacePane(cli CLI) (string, string, error) {
	tree, err := cli.Call("state.tree", map[string]any{})
	if err != nil {
		return "", "", err
	}
	workspace, _ := tree["activeProjectId"].(string)
	workspaces, _ := tree["workspaces"].([]any)
	for _, value := range workspaces {
		candidate, _ := value.(map[string]any)
		if candidate["id"] != workspace {
			continue
		}
		spaces, _ := candidate["spaces"].([]any)
		for _, spaceValue := range spaces {
			space, _ := spaceValue.(map[string]any)
			if space["active"] != true {
				continue
			}
			pane, _ := space["activePaneId"].(string)
			if workspace != "" && pane != "" {
				return workspace, pane, nil
			}
		}
	}
	return "", "", fmt.Errorf("state.tree returned no active workspace pane")
}
