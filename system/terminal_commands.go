package system

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type TerminalResult struct {
	Plugin string
	View   string
}

func VerifyTerminalCommands(cli CLI) ([]TerminalResult, error) {
	if cli.EvidenceDir == "" || !filepath.IsAbs(cli.EvidenceDir) {
		return nil, fmt.Errorf("evidence directory must be absolute")
	}
	if err := os.MkdirAll(cli.EvidenceDir, 0o700); err != nil {
		return nil, err
	}
	if err := resizeWindow(cli, 1400, 900); err != nil {
		return nil, err
	}
	results := make([]TerminalResult, 0, len(TerminalPlugins))
	for index, plugin := range TerminalPlugins {
		engine := strings.TrimPrefix(plugin, "soksak-plugin-terminal-")
		opened, err := cli.Call("tab.open", map[string]any{"program": "terminal-" + engine})
		if err != nil {
			return nil, err
		}
		view, _ := opened["tabId"].(string)
		if view == "" {
			return nil, fmt.Errorf("%s opened no tab", plugin)
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
		if err := resizeWindow(cli, 900, 650); err != nil {
			return nil, err
		}
		resizeMarker := marker + "_RESIZED"
		if _, err := terminal(cli, plugin, "send", view, map[string]any{"data": "printf '%s\n' " + resizeMarker + "\r"}); err != nil {
			return nil, err
		}
		resized, err := terminal(cli, plugin, "wait", view, map[string]any{"phase": "live", "contains": resizeMarker, "timeoutMs": 8000})
		if err != nil {
			return nil, err
		}
		narrow, _ := resized["cols"].(float64)
		if narrow < 1 || narrow >= wide {
			return nil, fmt.Errorf("%s columns did not decrease: %.0f -> %.0f", plugin, wide, narrow)
		}
		if err := resizeWindow(cli, 1400, 900); err != nil {
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
			return nil, err
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

func resizeWindow(cli CLI, width, height int) error {
	receipt, err := cli.Call("window.resizeSequence", map[string]any{
		"sizes":      []map[string]int{{"w": width, "h": height}},
		"intervalMs": 0,
	})
	if err != nil {
		return err
	}
	measurement, _ := receipt["measurement"].(map[string]any)
	if measurement["passed"] != true {
		return fmt.Errorf("window resize to %dx%d has no complete observation: %+v", width, height, receipt)
	}
	return nil
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
