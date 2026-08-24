package system

import (
	"encoding/json"
	"fmt"

	"github.com/min-median-max/soksak-terminal-tests/fleet"
	platformspec "github.com/soksak-ai/soksak-spec/go/platformspec"
)

type commandCaller interface {
	Call(command string, params map[string]any) (map[string]any, error)
	CallValue(command string, params map[string]any) (any, error)
}

func InstallTerminalFleet(profile fleet.Profile, cli commandCaller) error {
	if _, err := cli.Call("plugin.catalog", map[string]any{"refresh": true}); err != nil {
		return fmt.Errorf("refresh plugin catalogue: %w", err)
	}
	for _, plugin := range profile.Plugins {
		if err := InstallPlugin(plugin.ID, cli); err != nil {
			return err
		}
	}
	return nil
}

func InstallPlugin(pluginID string, cli commandCaller) error {
	if _, err := cli.Call("plugin.install", map[string]any{
		"registryId": "official", "pluginId": pluginID, "timeoutMs": 60000,
	}); err != nil {
		status, statusErr := cli.Call("plugin.install.status", map[string]any{"pluginId": pluginID})
		if statusErr != nil {
			return fmt.Errorf("install %s: %w; status unavailable: %v", pluginID, err, statusErr)
		}
		return fmt.Errorf("install %s: %w; status: %+v", pluginID, err, status)
	}
	chain, err := cli.Call("plugin.consent.chain", map[string]any{"id": pluginID})
	if err != nil {
		return fmt.Errorf("consent chain %s: %w", pluginID, err)
	}
	pending, ok := chain["pending"].([]any)
	if !ok {
		return fmt.Errorf("consent chain %s returned no pending list", pluginID)
	}
	for _, value := range pending {
		id, ok := value.(string)
		if !ok || id == "" {
			return fmt.Errorf("consent chain %s contains an invalid id", pluginID)
		}
		if _, err := cli.Call("plugin.consent.summary", map[string]any{"id": id}); err != nil {
			return fmt.Errorf("consent summary %s: %w", id, err)
		}
		granted, err := cli.Call("plugin.consent.grant", map[string]any{"id": id})
		if err != nil || granted["granted"] != true {
			return fmt.Errorf("consent grant %s failed: %v", id, err)
		}
	}
	if _, err := cli.Call("plugin.enable", map[string]any{"id": pluginID}); err != nil {
		return fmt.Errorf("enable %s: %w", pluginID, err)
	}
	return nil
}

func ReadRuntimeEnvironment(cli commandCaller) (platformspec.Environment, error) {
	raw, err := cli.CallValue("environment_get", map[string]any{})
	if err != nil {
		return platformspec.Environment{}, err
	}
	body, err := json.Marshal(raw)
	if err != nil {
		return platformspec.Environment{}, err
	}
	return platformspec.ParseEnvironment(body)
}
