package system

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/min-median-max/soksak-terminal-tests/fleet"
)

type TerminalResult struct {
	Plugin string
	View   string
}

func VerifyTerminalCommands(profile fleet.Profile, cli CLI) ([]TerminalResult, error) {
	if cli.EvidenceDir == "" || !filepath.IsAbs(cli.EvidenceDir) {
		return nil, fmt.Errorf("evidence directory must be absolute")
	}
	if err := os.MkdirAll(cli.EvidenceDir, 0o700); err != nil {
		return nil, err
	}
	results := make([]TerminalResult, 0, len(profile.Plugins))
	terminalPane := ""
	for index, selected := range profile.Plugins {
		plugin := selected.ID
		engine := strings.TrimPrefix(plugin, "soksak-plugin-terminal-")
		openParams := map[string]any{"program": "terminal-" + engine}
		if terminalPane != "" {
			openParams["pane"] = terminalPane
		}
		opened, err := cli.Call("tab.open", openParams)
		if err != nil {
			return nil, err
		}
		view, _ := opened["tabId"].(string)
		if view == "" {
			return nil, fmt.Errorf("%s opened no tab", plugin)
		}
		if terminalPane == "" {
			terminalPane, _ = opened["paneId"].(string)
			if terminalPane == "" {
				return nil, fmt.Errorf("%s opened no pane", plugin)
			}
			if _, err := cli.Call("pane.split", map[string]any{"pane": terminalPane, "side": "right"}); err != nil {
				return nil, err
			}
		}
		if _, err := resizePane(cli, terminalPane, 0.75); err != nil {
			return nil, err
		}
		marker := fmt.Sprintf("SOKSAK_SYSTEM_%d_%s_🙂_é", index, engine)
		if _, err := terminal(cli, plugin, "wait", view, map[string]any{"phase": "live", "timeoutMs": 8000}); err != nil {
			status, statusErr := terminal(cli, plugin, "status", view, nil)
			read, readErr := terminal(cli, plugin, "read", view, nil)
			sidecars, sidecarErr := cli.Call("sidecar_status", map[string]any{})
			return nil, fmt.Errorf("%s live wait failed: %w; status=%+v statusErr=%v read=%+v readErr=%v sidecars=%+v sidecarErr=%v", plugin, err, status, statusErr, read, readErr, sidecars, sidecarErr)
		}
		if _, err := terminal(cli, plugin, "send", view, map[string]any{"data": "printf '%s\n' " + marker + "\r"}); err != nil {
			return nil, err
		}
		ready, err := terminal(cli, plugin, "wait", view, map[string]any{"phase": "live", "contains": marker, "timeoutMs": 8000})
		if err != nil {
			status, statusErr := terminal(cli, plugin, "status", view, nil)
			read, readErr := terminal(cli, plugin, "read", view, nil)
			return nil, fmt.Errorf("%s marker wait failed: %w; status=%+v statusErr=%v read=%+v readErr=%v", plugin, err, status, statusErr, read, readErr)
		}
		if ready["fidelity"] != "complete" {
			return nil, fmt.Errorf("%s reported incomplete fidelity: %+v", plugin, ready)
		}
		wide, _ := ready["cols"].(float64)
		if wide < 1 {
			return nil, fmt.Errorf("%s reported no columns: %+v", plugin, ready)
		}
		tree, err := cli.Call("ui.tree", map[string]any{})
		if err != nil {
			return nil, err
		}
		address, err := terminalNodeAddress(tree, plugin, view, "terminal-root")
		if err != nil {
			return nil, err
		}
		evidence := terminalResizeEvidence{Plugin: plugin, View: view, NodeAddress: address}
		evidence.Wide, err = measureTerminalResize(cli, plugin, view, address)
		if err != nil {
			return nil, err
		}
		evidence.ResizeReceipt, err = resizePane(cli, terminalPane, 0.45)
		if err != nil {
			return nil, err
		}
		evidence.Narrow, err = measureTerminalResize(cli, plugin, view, address)
		if err != nil {
			return nil, err
		}
		if boundary := evidence.failureBoundary(); boundary == "core-layout" {
			evidence.Failure = "terminal DOM width did not decrease"
			_ = writeTerminalResizeEvidence(cli.EvidenceDir, evidence)
			return nil, fmt.Errorf("%s resize failed at %s: %.2f -> %.2f", plugin, boundary, evidence.Wide.DOM.Width, evidence.Narrow.DOM.Width)
		}
		resizeMarker := marker + "_RESIZED"
		if _, err := terminal(cli, plugin, "send", view, map[string]any{"data": "printf '%s\n' " + resizeMarker + "\r"}); err != nil {
			return nil, err
		}
		resized, err := terminal(cli, plugin, "wait", view, map[string]any{"phase": "live", "contains": resizeMarker, "timeoutMs": 8000})
		if err != nil {
			return nil, err
		}
		resized, err = terminal(cli, plugin, "wait", view, map[string]any{"phase": "live", "colsLessThan": wide, "timeoutMs": 8000})
		if err != nil {
			evidence.Narrow, _ = measureTerminalResize(cli, plugin, view, address)
			evidence.Failure = err.Error()
			_ = writeTerminalResizeEvidence(cli.EvidenceDir, evidence)
			return nil, fmt.Errorf("%s resize wait failed at %s: %w; evidence=%+v", plugin, evidence.failureBoundary(), err, evidence)
		}
		narrow, _ := resized["cols"].(float64)
		evidence.Narrow, err = measureTerminalResize(cli, plugin, view, address)
		if err != nil {
			return nil, err
		}
		if narrow < 1 || narrow >= wide {
			evidence.Failure = fmt.Sprintf("columns did not decrease: %.0f -> %.0f", wide, narrow)
			_ = writeTerminalResizeEvidence(cli.EvidenceDir, evidence)
			return nil, fmt.Errorf("%s columns did not decrease: %.0f -> %.0f", plugin, wide, narrow)
		}
		if err := writeTerminalResizeEvidence(cli.EvidenceDir, evidence); err != nil {
			return nil, err
		}
		if _, err := resizePane(cli, terminalPane, 0.75); err != nil {
			return nil, err
		}
		read, err := terminal(cli, plugin, "read", view, nil)
		if err != nil || !strings.Contains(fmt.Sprint(read["text"]), marker) {
			return nil, fmt.Errorf("%s did not read marker: %v %+v", plugin, err, read)
		}
		focus, err := terminal(cli, plugin, "focus", view, nil)
		if err != nil || focus["focused"] != true {
			return nil, fmt.Errorf("%s did not focus: %v %+v", plugin, err, focus)
		}
		if _, err := terminal(cli, plugin, "status", view, nil); err != nil {
			return nil, err
		}
		tail := fmt.Sprintf("SOKSAK_HIGH_OUTPUT_TAIL_%d", index)
		if _, err := terminal(cli, plugin, "send", view, map[string]any{"data": "yes X | head -c 262144; printf '\\n%s\\n' " + tail + "\r"}); err != nil {
			return nil, err
		}
		if _, err := terminal(cli, plugin, "wait", view, map[string]any{"phase": "live", "contains": tail, "timeoutMs": 20000}); err != nil {
			status, statusErr := terminal(cli, plugin, "status", view, nil)
			read, readErr := terminal(cli, plugin, "read", view, nil)
			sidecars, sidecarErr := cli.Call("sidecar_status", map[string]any{})
			return nil, fmt.Errorf("%s high-output wait failed: %w; status=%+v statusErr=%v read=%+v readErr=%v sidecars=%+v sidecarErr=%v", plugin, err, status, statusErr, read, readErr, sidecars, sidecarErr)
		}
		if err := verifyTerminalNodes(cli, plugin, view); err != nil {
			return nil, err
		}
		if err := captureTerminal(cli, plugin); err != nil {
			return nil, err
		}
		results = append(results, TerminalResult{Plugin: plugin, View: view})
	}
	return results, nil
}

func resizePane(cli CLI, pane string, ratio float64) (map[string]any, error) {
	receipt, err := cli.Call("pane.resize", map[string]any{"pane": pane, "edge": "right", "ratio": ratio})
	if err != nil {
		return nil, err
	}
	if receipt["paneId"] != pane {
		return nil, fmt.Errorf("pane resize targeted %s and returned %+v", pane, receipt)
	}
	_, err = cli.Call("ui.layout.wait-settled", map[string]any{"timeoutMs": 8000})
	if err != nil {
		return nil, err
	}
	return receipt, nil
}

func captureTerminal(cli CLI, plugin string) error {
	image := filepath.Join(cli.EvidenceDir, plugin+".png")
	recording := filepath.Join(cli.EvidenceDir, plugin+"-recording")
	if err := os.RemoveAll(recording); err != nil {
		return err
	}
	if _, err := cli.Call("window.snapshot", map[string]any{"path": image}); err != nil {
		return err
	}
	if _, err := cli.Call("window.record", map[string]any{"dir": recording, "frames": 6, "intervalMs": 16}); err != nil {
		return err
	}
	info, err := os.Stat(image)
	if err != nil || info.Size() == 0 {
		return fmt.Errorf("%s capture was not written: %v", plugin, err)
	}
	frames, err := filepath.Glob(filepath.Join(recording, "f*.png"))
	if err != nil || len(frames) != 6 {
		return fmt.Errorf("%s recording has %d frames: %v", plugin, len(frames), err)
	}
	return nil
}

func terminal(cli CLI, plugin, command, view string, params map[string]any) (map[string]any, error) {
	if params == nil {
		params = map[string]any{}
	}
	params = cloneMap(params)
	params["view"] = view
	return cli.Call("plugin."+plugin+"."+command, params)
}

func verifyTerminalNodes(cli CLI, plugin, view string) error {
	data, err := cli.Call("ui.tree", map[string]any{})
	if err != nil {
		return err
	}
	nodes, _ := data["nodes"].([]any)
	wanted := map[string]bool{
		"terminal-root": false, "terminal-screen": false,
		"terminal-input": false, "terminal-restore-status": false,
	}
	for _, value := range nodes {
		node, _ := value.(map[string]any)
		address, _ := node["address"].(string)
		path, _ := node["nodePath"].(string)
		if !strings.Contains(address, plugin) || !strings.Contains(address, view) {
			continue
		}
		if _, exists := wanted[path]; exists {
			wanted[path] = true
		}
		if path == "terminal-screen" && (node["role"] != "log" || node["ariaLive"] != "polite") {
			return fmt.Errorf("%s terminal screen accessibility is incomplete: %+v", plugin, node)
		}
		if path == "terminal-input" {
			label, _ := node["ariaLabel"].(string)
			if label == "" {
				return fmt.Errorf("%s terminal input has no accessible label", plugin)
			}
		}
	}
	for path, found := range wanted {
		if !found {
			return fmt.Errorf("%s view %s exposes no %s", plugin, view, path)
		}
	}
	return nil
}
