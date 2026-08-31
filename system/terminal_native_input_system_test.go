//go:build system

package system

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/min-median-max/soksak-terminal-tests/fleet"
)

type nativeMatrix struct {
	profile   fleet.Profile
	lifecycle *Lifecycle
	cli       CLI
	window    string
	pane      string
}

type nativeFocusReport struct {
	Provider            string `json:"provider"`
	PointerInputRoute   string `json:"pointerInputRoute"`
	WindowFocused       bool   `json:"windowFocused"`
	ForegroundPreserved bool   `json:"foregroundPreserved"`
	FocusedInput        bool   `json:"focusedInput"`
	FocusSequence       int    `json:"focusSequence"`
}

type nativeCursorReport struct {
	Provider      string `json:"provider"`
	FocusedInput  bool   `json:"focusedInput"`
	CursorVisible bool   `json:"cursorVisible"`
	CursorActive  bool   `json:"cursorActive"`
	CursorRow     int    `json:"cursorRow"`
	CursorColumn  int    `json:"cursorColumn"`
}

type nativeKeyboardReport struct {
	Provider            string `json:"provider"`
	PointerInputRoute   string `json:"pointerInputRoute"`
	KeyboardInputRoute  string `json:"keyboardInputRoute"`
	WindowFocused       bool   `json:"windowFocused"`
	ForegroundPreserved bool   `json:"foregroundPreserved"`
	FocusedInput        bool   `json:"focusedInput"`
	ExpectedInputCount  int    `json:"expectedInputCount"`
	AcceptedInputDelta  int    `json:"acceptedInputDelta"`
	PTYWriteDelta       int    `json:"ptyWriteDelta"`
}

func TestInstalledTerminalNativeFocusMatrix(t *testing.T) {
	matrix := startNativeMatrix(t, "native-focus")
	reports := make([]nativeFocusReport, 0, len(matrix.profile.Plugins))
	for _, plugin := range matrix.profile.Plugins {
		tab, screen, _ := openNativeTerminal(t, matrix, plugin)
		pointerRoute, err := nativeClickExposedNode(matrix.cli, matrix.window, screen)
		if err != nil {
			t.Fatalf("native focus %s: %v", plugin.ID, err)
		}
		presentation := nativePresentation(t, matrix.cli, plugin.ID, tab)
		report := nativeFocusReport{
			Provider: plugin.ID, PointerInputRoute: pointerRoute,
			WindowFocused: true, ForegroundPreserved: true,
		}
		report.FocusedInput, _ = presentation["focusedInput"].(bool)
		report.FocusSequence, _ = exactInt(presentation["focusSequence"])
		if !report.FocusedInput || report.FocusSequence < 1 {
			t.Fatalf("native pointer did not retain %s input focus: %+v", plugin.ID, report)
		}
		reports = append(reports, report)
	}
	writeNativeReport(t, matrix.lifecycle, "terminal-native-focus.json", reports)
	finishNativeMatrix(t, matrix)
}

func TestInstalledTerminalNativeCursorMatrix(t *testing.T) {
	matrix := startNativeMatrix(t, "native-cursor")
	reports := make([]nativeCursorReport, 0, len(matrix.profile.Plugins))
	for _, plugin := range matrix.profile.Plugins {
		tab, screen, _ := openNativeTerminal(t, matrix, plugin)
		if _, err := nativeClickExposedNode(matrix.cli, matrix.window, screen); err != nil {
			t.Fatalf("native cursor focus %s: %v", plugin.ID, err)
		}
		if _, err := terminal(matrix.cli, plugin.ID, "wait", tab, map[string]any{
			"phase": "live", "focusedInput": true, "cursorVisible": true,
			"cursorActive": true, "timeoutMs": 12000,
		}); err != nil {
			t.Fatalf("native cursor readiness %s: %v", plugin.ID, err)
		}
		presentation := nativePresentation(t, matrix.cli, plugin.ID, tab)
		report := nativeCursorReport{Provider: plugin.ID}
		report.FocusedInput, _ = presentation["focusedInput"].(bool)
		report.CursorVisible, _ = presentation["cursorVisible"].(bool)
		report.CursorActive, _ = presentation["cursorActive"].(bool)
		var rowOK, columnOK bool
		report.CursorRow, rowOK = exactInt(presentation["cursorRow"])
		report.CursorColumn, columnOK = exactInt(presentation["cursorColumn"])
		if !report.FocusedInput || !report.CursorVisible || !report.CursorActive || !rowOK || !columnOK {
			t.Fatalf("native pointer did not activate %s cursor: %+v presentation=%+v", plugin.ID, report, presentation)
		}
		if err := captureTerminal(matrix.cli, "native-cursor-"+plugin.ID); err != nil {
			t.Fatalf("capture %s cursor: %v", plugin.ID, err)
		}
		reports = append(reports, report)
	}
	writeNativeReport(t, matrix.lifecycle, "terminal-native-cursor.json", reports)
	finishNativeMatrix(t, matrix)
}

func TestInstalledTerminalNativeKeyboardMatrix(t *testing.T) {
	matrix := startNativeMatrix(t, "native-keyboard")
	reports := make([]nativeKeyboardReport, 0, len(matrix.profile.Plugins))
	for _, plugin := range matrix.profile.Plugins {
		tab, screen, _ := openNativeTerminal(t, matrix, plugin)
		pointerRoute, err := nativeClickExposedNode(matrix.cli, matrix.window, screen)
		if err != nil {
			t.Fatalf("native keyboard focus %s: %v", plugin.ID, err)
		}
		before := nativePresentation(t, matrix.cli, plugin.ID, tab)
		beforeAccepted, _ := exactInt(before["acceptedInputSequence"])
		beforeWrites, _ := exactInt(before["ptyWriteSequence"])

		marker := "SOKSAK_NATIVE_KEYBOARD_" + strings.ToUpper(strings.TrimPrefix(plugin.ID, "soksak-plugin-terminal-"))
		command, err := terminalPaletteCommand(matrix.profile.Platform, marker)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(command, marker) {
			t.Fatalf("%s native input contains its awaited marker", plugin.ID)
		}
		keyboardRoute := ""
		for _, key := range command {
			keyboardRoute, err = nativePressWindowKey(matrix.cli, matrix.window, string(key))
			if err != nil {
				t.Fatalf("native type %s key %q: %v", plugin.ID, key, err)
			}
		}
		keyboardRoute, err = nativePressWindowKey(matrix.cli, matrix.window, "Enter")
		if err != nil {
			t.Fatalf("native enter %s: %v", plugin.ID, err)
		}
		if _, err := terminal(matrix.cli, plugin.ID, "wait", tab, map[string]any{
			"phase": "live", "contains": marker, "timeoutMs": 12000,
		}); err != nil {
			t.Fatalf("native keyboard round trip %s: %v", plugin.ID, err)
		}
		after := nativePresentation(t, matrix.cli, plugin.ID, tab)
		afterAccepted, _ := exactInt(after["acceptedInputSequence"])
		afterWrites, _ := exactInt(after["ptyWriteSequence"])
		report := nativeKeyboardReport{
			Provider: plugin.ID, PointerInputRoute: pointerRoute, KeyboardInputRoute: keyboardRoute,
			WindowFocused: true, ForegroundPreserved: true,
			ExpectedInputCount: len([]rune(command)) + 1,
			AcceptedInputDelta: afterAccepted - beforeAccepted, PTYWriteDelta: afterWrites - beforeWrites,
		}
		report.FocusedInput, _ = after["focusedInput"].(bool)
		if !report.FocusedInput || report.AcceptedInputDelta != report.ExpectedInputCount ||
			report.PTYWriteDelta != report.ExpectedInputCount {
			t.Fatalf("native keyboard did not reach %s PTY: %+v", plugin.ID, report)
		}
		if err := captureTerminal(matrix.cli, "native-keyboard-"+plugin.ID); err != nil {
			t.Fatalf("capture %s keyboard result: %v", plugin.ID, err)
		}
		reports = append(reports, report)
	}
	writeNativeReport(t, matrix.lifecycle, "terminal-native-keyboard.json", reports)
	finishNativeMatrix(t, matrix)
}

func startNativeMatrix(t *testing.T, scenario string) nativeMatrix {
	t.Helper()
	profile := profileFromEnvironment(t)
	if profile.Platform != "darwin" {
		t.Fatalf("native AppKit matrix requires darwin, got %s", profile.Platform)
	}
	lifecycle, err := NewLifecycle(nativeInputLifecycleConfigFromEnvironment(t, scenario))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(lifecycle.Close)
	if err := lifecycle.Start(); err != nil {
		t.Fatal(err)
	}
	cli := lifecycle.Client()
	if err := InstallConfiguredTerminalFleet(profile, cli); err != nil {
		t.Fatal(err)
	}
	window, err := lifecycle.OpenWorkspace()
	if err != nil {
		t.Fatal(err)
	}
	cli = lifecycle.Client()
	if _, err := cli.Call("window.focus", map[string]any{"label": window}); err != nil {
		t.Fatalf("activate isolated %s window: %v", scenario, err)
	}
	_, pane, err := activeWorkspacePane(cli)
	if err != nil {
		t.Fatal(err)
	}
	return nativeMatrix{profile: profile, lifecycle: lifecycle, cli: cli, window: window, pane: pane}
}

func nativeInputLifecycleConfigFromEnvironment(t *testing.T, scenario string) LifecycleConfig {
	t.Helper()
	config := lifecycleConfigFromEnvironment(t, scenario)
	config.Presentation = "interactive"
	config.Focus = true
	return config
}

func openNativeTerminal(t *testing.T, matrix nativeMatrix, plugin fleet.Plugin) (string, string, string) {
	t.Helper()
	program := plugin.Program
	opened, err := matrix.cli.Call("tab.open", map[string]any{
		"pane": matrix.pane, "program": program, "mountTimeoutMs": 12000,
	})
	if err != nil {
		t.Fatalf("open %s: %v", plugin.ID, err)
	}
	tab, _ := opened["tabId"].(string)
	if tab == "" {
		t.Fatalf("open %s returned no tab", plugin.ID)
	}
	if _, err := matrix.cli.Call("tab.activate", map[string]any{"tab": tab}); err != nil {
		t.Fatal(err)
	}
	if _, err := terminal(matrix.cli, plugin.ID, "wait", tab, map[string]any{
		"phase": "live", "timeoutMs": 12000,
	}); err != nil {
		t.Fatalf("wait %s: %v", plugin.ID, err)
	}
	nodes, err := exposedNodes(matrix.cli)
	if err != nil {
		t.Fatal(err)
	}
	screen := nodeAddress(nodes, "terminal-screen", plugin.ID+"\x00"+tab)
	input := nodeAddress(nodes, "terminal-input", plugin.ID+"\x00"+tab)
	if screen == "" || input == "" {
		t.Fatalf("%s has no public screen/input address", plugin.ID)
	}
	return tab, screen, input
}

func nativePresentation(t *testing.T, cli CLI, plugin, tab string) map[string]any {
	t.Helper()
	status, err := terminal(cli, plugin, "status", tab, nil)
	if err != nil {
		t.Fatal(err)
	}
	presentation, ok := status["presentation"].(map[string]any)
	if !ok {
		t.Fatalf("%s has no presentation status: %+v", plugin, status)
	}
	return presentation
}

func writeNativeReport(t *testing.T, lifecycle *Lifecycle, name string, reports any) {
	t.Helper()
	body, err := json.MarshalIndent(map[string]any{"reports": reports}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lifecycle.config.EvidenceDir, name), body, 0o600); err != nil {
		t.Fatal(err)
	}
}

func finishNativeMatrix(t *testing.T, matrix nativeMatrix) {
	t.Helper()
	if err := matrix.lifecycle.Finish(); err != nil {
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
