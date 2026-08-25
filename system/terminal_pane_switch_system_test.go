//go:build system

package system

import (
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestInstalledTerminalPaneSwitchStability(t *testing.T) {
	matrix := startNativeMatrix(t, "pane-switch")
	plugins := []string{
		"soksak-plugin-terminal-xterm",
		"soksak-plugin-terminal-kitty",
		"soksak-plugin-terminal-vt100",
	}
	panes := []string{matrix.pane}
	for len(panes) < 3 {
		split, err := matrix.cli.Call("pane.split", map[string]any{
			"pane": panes[len(panes)-1], "side": "right", "mountTimeoutMs": 0,
		})
		if err != nil {
			t.Fatalf("split pane %d: %v", len(panes)+1, err)
		}
		pane, _ := split["paneId"].(string)
		if pane == "" {
			t.Fatalf("split pane %d returned no pane: %+v", len(panes)+1, split)
		}
		panes = append(panes, pane)
	}
	if _, err := matrix.cli.Call("ui.layout.wait-settled", map[string]any{"timeoutMs": 8000}); err != nil {
		t.Fatal(err)
	}

	tabs := make([]string, len(plugins))
	screens := make([]string, len(plugins))
	headers := make([]string, len(plugins))
	for index, plugin := range plugins {
		program := "terminal-" + plugin[len("soksak-plugin-terminal-"):]
		opened, err := matrix.cli.Call("tab.open", map[string]any{
			"pane": panes[index], "program": program, "mountTimeoutMs": 12000,
		})
		if err != nil {
			t.Fatalf("open %s: %v", plugin, err)
		}
		tabs[index], _ = opened["tabId"].(string)
		if tabs[index] == "" {
			t.Fatalf("open %s returned no tab", plugin)
		}
		if _, err := terminal(matrix.cli, plugin, "wait", tabs[index], map[string]any{
			"phase": "live", "timeoutMs": 12000,
		}); err != nil {
			t.Fatalf("wait %s: %v", plugin, err)
		}
	}
	nodes, err := exposedNodes(matrix.cli)
	if err != nil {
		t.Fatal(err)
	}
	for index, plugin := range plugins {
		screens[index] = nodeAddress(nodes, "terminal-screen", plugin+"\x00"+tabs[index])
		headers[index] = nodeAddress(nodes, "tab/view/"+tabs[index], "")
		if screens[index] == "" || headers[index] == "" {
			t.Fatalf("%s public pane-switch nodes are incomplete", plugin)
		}
	}
	middleBody := nodeAddress(nodes, "layout/tab/"+tabs[1], "")
	if middleBody == "" {
		t.Fatal("middle terminal has no public tab body")
	}
	measured, err := matrix.cli.Call("ui.measure", map[string]any{"address": middleBody})
	if err != nil {
		t.Fatal(err)
	}
	initialIdentity, _ := measured["nodeIdentity"].(string)
	if initialIdentity == "" {
		t.Fatal("middle terminal has no public node identity")
	}
	beforeImage := filepath.Join(matrix.lifecycle.config.EvidenceDir, "middle-before.png")
	if _, err := matrix.cli.Call("window.snapshot", map[string]any{"path": beforeImage, "node": screens[1]}); err != nil {
		t.Fatal(err)
	}
	beforePixels := readThemePixelsFile(t, beforeImage)

	reports := []PaneSwitchReport{}
	for index, target := range []int{2, 1, 2, 1} {
		dir := filepath.Join(matrix.lifecycle.config.EvidenceDir, fmt.Sprintf("switch-%d-to-pane-%d", index, target+1))
		receipt := nativeClickWithRecording(t, matrix.cli, matrix.window, headers[target], dir, []string{middleBody, screens[1]})
		trace, ok := receipt["trace"].(map[string]any)
		if !ok {
			t.Fatalf("pane switch %d returned no finite trace: %+v", index, receipt)
		}
		report, err := EvaluatePaneSwitchTrace("pane-2", trace, middleBody, screens[1], initialIdentity)
		if err != nil {
			t.Fatal(err)
		}
		if report.BlankFrames != 0 || report.RemountFrames != 0 {
			t.Fatalf("pane switch %d unstable: %+v", index, report)
		}
		reports = append(reports, report)
	}
	afterImage := filepath.Join(matrix.lifecycle.config.EvidenceDir, "middle-after.png")
	if _, err := matrix.cli.Call("window.snapshot", map[string]any{"path": afterImage, "node": screens[1]}); err != nil {
		t.Fatal(err)
	}
	if err := compareTerminalThemePixels(readThemePixelsFile(t, afterImage), beforePixels); err != nil {
		t.Fatal(err)
	}
	writePaneSwitchReport(t, matrix.lifecycle.config.EvidenceDir, reports)
	finishNativeMatrix(t, matrix)
}

func nativeClickWithRecording(
	t *testing.T,
	cli CLI,
	window, address, dir string,
	traceAddresses []string,
) map[string]any {
	t.Helper()
	measurement, err := cli.Call("ui.measure", map[string]any{"address": address})
	if err != nil {
		t.Fatal(err)
	}
	rect, _ := measurement["rect"].(map[string]any)
	x, xOK := number(rect["x"])
	y, yOK := number(rect["y"])
	w, wOK := number(rect["w"])
	h, hOK := number(rect["h"])
	if !xOK || !yOK || !wOK || !hOK || w <= 0 || h <= 0 {
		t.Fatalf("%s has no finite public rectangle: %+v", address, rect)
	}
	receipt, err := cli.Call("window.input.pointer.click", map[string]any{
		"window": window, "x": x + w/2, "y": y + h/2,
		"recordDir": dir, "recordFrames": 40, "recordIntervalMs": 16,
		"traceAddresses": traceAddresses,
	})
	if err != nil {
		t.Fatal(err)
	}
	recording, _ := receipt["recording"].(map[string]any)
	if recording["status"] != "complete" || receipt["delivered"] != true {
		t.Fatalf("native pane click did not complete: %+v", receipt)
	}
	return receipt
}

func readThemePixelsFile(t *testing.T, path string) terminalThemePixels {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	pixels, decodeErr := png.Decode(file)
	closeErr := file.Close()
	if decodeErr != nil || closeErr != nil {
		t.Fatalf("decode %s: decode=%v close=%v", path, decodeErr, closeErr)
	}
	theme, err := readTerminalThemePixels(pixels)
	if err != nil {
		t.Fatal(err)
	}
	return theme
}

func writePaneSwitchReport(t *testing.T, dir string, reports []PaneSwitchReport) {
	t.Helper()
	body, err := json.MarshalIndent(map[string]any{"reports": reports}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "terminal-pane-switch.json"), body, 0o600); err != nil {
		t.Fatal(err)
	}
}
