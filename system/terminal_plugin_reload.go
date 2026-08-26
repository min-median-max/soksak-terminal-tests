package system

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type pluginRealmLifetime struct {
	Open     int `json:"open"`
	Created  int `json:"created"`
	Disposed int `json:"disposed"`
}

type pluginReloadSample struct {
	Attempt   int                 `json:"attempt"`
	Unchanged bool                `json:"unchanged"`
	Realms    pluginRealmLifetime `json:"realms"`
	Streams   int                 `json:"streams"`
}

func VerifyPluginReloadLifetime(platform string, cli CLI, view TerminalResult, attempts int) error {
	if attempts < 1 {
		return fmt.Errorf("plugin reload attempts must be positive")
	}
	baselineHealth, err := cli.Call("state.health", map[string]any{})
	if err != nil {
		return err
	}
	baselineRealms, baselineStreams, err := readPluginLifetime(baselineHealth)
	if err != nil {
		return err
	}
	baselinePTY, err := ptyStatus(cli)
	if err != nil {
		return err
	}
	baselinePID, err := shellPID(baselinePTY, view.Pane)
	if err != nil {
		return err
	}
	samples := make([]pluginReloadSample, 0, attempts)
	for attempt := 1; attempt <= attempts; attempt++ {
		reloaded, err := cli.Call("plugin.reload", map[string]any{"id": view.Plugin})
		if err != nil || reloaded["status"] != "enabled" {
			return fmt.Errorf("reload %d did not enable %s: err=%v response=%+v", attempt, view.Plugin, err, reloaded)
		}
		unchanged, _ := reloaded["unchanged"].(bool)
		if !unchanged {
			return fmt.Errorf("reload %d replaced unchanged %s runtime: %+v", attempt, view.Plugin, reloaded)
		}
		if _, err := cli.Call("plugin.boot.wait", map[string]any{"timeoutMs": 30000}); err != nil {
			return err
		}
		if _, err := cli.Call("tab.mount.wait", map[string]any{"tab": view.View, "timeoutMs": 20000}); err != nil {
			return err
		}
		if _, err := terminal(cli, view.Plugin, "wait", view.View, map[string]any{
			"phase": "live", "timeoutMs": 30000,
		}); err != nil {
			return err
		}
		health, err := cli.Call("state.health", map[string]any{})
		if err != nil {
			return err
		}
		realms, streams, err := readPluginLifetime(health)
		if err != nil {
			return err
		}
		if realms.Open != baselineRealms.Open || realms.Created-realms.Disposed != realms.Open {
			return fmt.Errorf("plugin realm lifetime leaked at reload %d: baseline=%+v current=%+v", attempt, baselineRealms, realms)
		}
		if streams != baselineStreams {
			return fmt.Errorf("stream receivers leaked at reload %d: %d -> %d", attempt, baselineStreams, streams)
		}
		status, err := ptyStatus(cli)
		if err != nil {
			return err
		}
		pid, err := shellPID(status, view.Pane)
		if err != nil || pid != baselinePID {
			return fmt.Errorf("plugin reload replaced PTY shell %d with %d: %v", baselinePID, pid, err)
		}
		samples = append(samples, pluginReloadSample{Attempt: attempt, Unchanged: true, Realms: realms, Streams: streams})
	}
	marker := "SOKSAK_PLUGIN_RELOAD_LIFETIME"
	command, err := terminalPrintCommand(platform, marker)
	if err != nil {
		return err
	}
	if err := typeTerminalCommand(cli, view.Plugin, view.View, command); err != nil {
		return err
	}
	if _, err := terminal(cli, view.Plugin, "wait", view.View, map[string]any{
		"phase": "live", "contains": marker, "timeoutMs": 12000,
	}); err != nil {
		return err
	}
	if err := captureTerminal(cli, view.Plugin+"-reload-lifetime"); err != nil {
		return err
	}
	body, err := json.MarshalIndent(map[string]any{"plugin": view.Plugin, "samples": samples}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(cli.EvidenceDir, "terminal-plugin-reload-lifetime.json"), body, 0o600)
}

func readPluginLifetime(health map[string]any) (pluginRealmLifetime, int, error) {
	plugins, _ := health["plugins"].(map[string]any)
	realms, _ := plugins["realms"].(map[string]any)
	streams, _ := health["streams"].(map[string]any)
	encoded, err := json.Marshal(realms)
	if err != nil {
		return pluginRealmLifetime{}, 0, err
	}
	var lifetime pluginRealmLifetime
	if err := json.Unmarshal(encoded, &lifetime); err != nil {
		return pluginRealmLifetime{}, 0, err
	}
	streamCount, ok := exactInt(streams["open"])
	if realms == nil || !ok {
		return pluginRealmLifetime{}, 0, fmt.Errorf("state.health has no plugin realm/stream lifetime: %+v", health)
	}
	return lifetime, streamCount, nil
}
