package system

import (
	"fmt"
	"strings"
)

type TerminalResult struct {
	Plugin string
	View   string
}

func VerifyTerminalCommands(cli CLI) ([]TerminalResult, error) {
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
			return nil, err
		}
		if _, err := terminal(cli, plugin, "send", view, map[string]any{"data": "printf '%s\n' " + marker + "\r"}); err != nil {
			return nil, err
		}
		ready, err := terminal(cli, plugin, "wait", view, map[string]any{"phase": "live", "contains": marker, "timeoutMs": 8000})
		if err != nil {
			return nil, err
		}
		if ready["fidelity"] != "complete" {
			return nil, fmt.Errorf("%s reported incomplete fidelity: %+v", plugin, ready)
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
		if err := verifyTerminalNodes(cli, plugin, view); err != nil {
			return nil, err
		}
		results = append(results, TerminalResult{Plugin: plugin, View: view})
	}
	return results, nil
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
