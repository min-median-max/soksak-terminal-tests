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

type nativeInputReport struct {
	Provider            string `json:"provider"`
	PointerInputRoute   string `json:"pointerInputRoute"`
	KeyboardInputRoute  string `json:"keyboardInputRoute"`
	WindowFocused       bool   `json:"windowFocused"`
	ForegroundPreserved bool   `json:"foregroundPreserved"`
	FocusedInput        bool   `json:"focusedInput"`
	CursorVisible       bool   `json:"cursorVisible"`
	CursorActive        bool   `json:"cursorActive"`
	AcceptedInput       int    `json:"acceptedInputSequence"`
	PTYWrites           int    `json:"ptyWriteSequence"`
}

func TestInstalledTerminalNativeInputMatrix(t *testing.T) {
	planPath := os.Getenv("SOKSAK_TEST_CANDIDATE_PLAN")
	profile := profileFromEnvironment(t)
	if profile.Platform != "darwin" {
		t.Fatalf("native AppKit input matrix requires darwin, got %s", profile.Platform)
	}
	lifecycle, err := NewLifecycle(nativeInputLifecycleConfigFromEnvironment(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(lifecycle.Close)
	if err := lifecycle.Start(); err != nil {
		t.Fatal(err)
	}
	cli := lifecycle.Client()
	if planPath == "" {
		if err := InstallTerminalFleet(profile, cli); err != nil {
			t.Fatal(err)
		}
	} else {
		if err := InstallCandidateFleet(planPath, cli); err != nil {
			t.Fatal(err)
		}
		if _, err := cli.Call("plugin.boot.wait", map[string]any{"timeoutMs": 45000}); err != nil {
			t.Fatal(err)
		}
		if err := EnableCandidateTerminalFleet(profile, cli); err != nil {
			t.Fatal(err)
		}
	}
	window, err := lifecycle.OpenWorkspace()
	if err != nil {
		t.Fatal(err)
	}
	cli = lifecycle.Client()
	if _, err := cli.Call("window.focus", map[string]any{"label": window}); err != nil {
		t.Fatalf("activate isolated native-input window: %v", err)
	}
	_, pane, err := activeWorkspacePane(cli)
	if err != nil {
		t.Fatal(err)
	}

	reports := make([]nativeInputReport, 0, len(profile.Plugins))
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
		if _, err := cli.Call("tab.activate", map[string]any{"tab": tab}); err != nil {
			t.Fatal(err)
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
		screen := nodeAddress(nodes, "terminal-screen", plugin.ID+"\x00"+tab)
		input := nodeAddress(nodes, "terminal-input", plugin.ID+"\x00"+tab)
		if screen == "" || input == "" {
			t.Fatalf("%s has no public screen/input address", plugin.ID)
		}
		pointerRoute, err := nativeClickExposedNode(cli, window, screen)
		if err != nil {
			t.Fatalf("native focus %s: %v", plugin.ID, err)
		}
		status, err := cli.Call("plugin."+plugin.ID+".status", map[string]any{"view": tab})
		if err != nil {
			t.Fatal(err)
		}
		presentation, _ := status["presentation"].(map[string]any)
		if focused, _ := presentation["focusedInput"].(bool); !focused {
			t.Fatalf("native pointer did not focus %s input: %+v", plugin.ID, presentation)
		}

		marker := "SOKSAK_NATIVE_INPUT_" + strings.ToUpper(strings.TrimPrefix(plugin.ID, "soksak-plugin-terminal-"))
		command, err := terminalPaletteCommand(profile.Platform, marker)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(command, marker) {
			t.Fatalf("%s native input contains its awaited marker", plugin.ID)
		}
		keyboardRoute := ""
		for _, key := range command {
			keyboardRoute, err = nativePressWindowKey(cli, window, string(key))
			if err != nil {
				t.Fatalf("native type %s key %q: %v", plugin.ID, key, err)
			}
		}
		keyboardRoute, err = nativePressWindowKey(cli, window, "Enter")
		if err != nil {
			t.Fatalf("native enter %s: %v", plugin.ID, err)
		}
		if _, err := cli.Call("plugin."+plugin.ID+".wait", map[string]any{
			"phase": "live", "view": tab, "contains": marker, "timeoutMs": 12000,
		}); err != nil {
			t.Fatalf("native keyboard round trip %s: %v", plugin.ID, err)
		}
		if err := captureTerminal(cli, "native-input-"+plugin.ID); err != nil {
			t.Fatalf("capture %s: %v", plugin.ID, err)
		}
		status, err = cli.Call("plugin."+plugin.ID+".status", map[string]any{"view": tab})
		if err != nil {
			t.Fatal(err)
		}
		presentation, _ = status["presentation"].(map[string]any)
		report := nativeInputReport{
			Provider: plugin.ID, PointerInputRoute: pointerRoute, KeyboardInputRoute: keyboardRoute,
			WindowFocused: true, ForegroundPreserved: true,
		}
		report.FocusedInput, _ = presentation["focusedInput"].(bool)
		report.CursorVisible, _ = presentation["cursorVisible"].(bool)
		report.CursorActive, _ = presentation["cursorActive"].(bool)
		report.AcceptedInput, _ = exactInt(presentation["acceptedInputSequence"])
		report.PTYWrites, _ = exactInt(presentation["ptyWriteSequence"])
		if !report.FocusedInput || !report.CursorVisible || !report.CursorActive ||
			report.AcceptedInput < 2 || report.PTYWrites < 2 {
			t.Errorf("%s native input state=%+v", plugin.ID, report)
		}
		reports = append(reports, report)
	}
	body, err := json.MarshalIndent(map[string]any{"reports": reports}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lifecycle.config.EvidenceDir, "terminal-native-input.json"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.Finish(); err != nil {
		t.Fatal(err)
	}
}

func nativeClickExposedNode(cli CLI, window, address string) (string, error) {
	measured, err := cli.Call("ui.measure", map[string]any{"address": address})
	if err != nil {
		return "", err
	}
	rect, _ := measured["rect"].(map[string]any)
	x, xOK := number(rect["x"])
	y, yOK := number(rect["y"])
	w, wOK := number(rect["w"])
	h, hOK := number(rect["h"])
	if !xOK || !yOK || !wOK || !hOK || w <= 0 || h <= 0 {
		return "", fmt.Errorf("%s has no finite nonempty public rectangle: %+v", address, rect)
	}
	receipt, err := cli.Call("window.input.pointer.click", map[string]any{
		"window": window, "x": x + w/2, "y": y + h/2,
	})
	if err != nil {
		return "", err
	}
	return nativeInputRoute(receipt, true)
}

func nativePressWindowKey(cli CLI, window, key string) (string, error) {
	receipt, err := cli.Call("window.input.key.press", map[string]any{
		"window": window, "key": key,
	})
	if err != nil {
		return "", err
	}
	return nativeInputRoute(receipt, true)
}

func nativeInputRoute(receipt map[string]any, wantFocused bool) (string, error) {
	route, _ := receipt["inputRoute"].(string)
	if route == "" || receipt["delivered"] != true {
		return "", fmt.Errorf("native input returned no delivery route: %+v", receipt)
	}
	focused, _ := receipt["windowFocused"].(bool)
	if focused != wantFocused {
		return "", fmt.Errorf("native input window focus=%t want=%t: %+v", focused, wantFocused, receipt)
	}
	if preserved, _ := receipt["foregroundPreserved"].(bool); !preserved {
		return "", fmt.Errorf("native input changed the foreground application: %+v", receipt)
	}
	return route, nil
}
