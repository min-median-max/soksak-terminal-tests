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
		if _, err := cli.Call("plugin.install", map[string]any{
			"registryId": "official", "pluginId": plugin.ID,
		}); err != nil {
			return fmt.Errorf("install %s: %w", plugin.ID, err)
		}
		chain, err := cli.Call("plugin.consent.chain", map[string]any{"id": plugin.ID})
		if err != nil {
			return fmt.Errorf("consent chain %s: %w", plugin.ID, err)
		}
		pending, ok := chain["pending"].([]any)
		if !ok {
			return fmt.Errorf("consent chain %s returned no pending list", plugin.ID)
		}
		for _, value := range pending {
			id, ok := value.(string)
			if !ok || id == "" {
				return fmt.Errorf("consent chain %s contains an invalid id", plugin.ID)
			}
			if _, err := cli.Call("plugin.consent.summary", map[string]any{"id": id}); err != nil {
				return fmt.Errorf("consent summary %s: %w", id, err)
			}
			granted, err := cli.Call("plugin.consent.grant", map[string]any{"id": id})
			if err != nil || granted["granted"] != true {
				return fmt.Errorf("consent grant %s failed: %v", id, err)
			}
		}
		if _, err := cli.Call("plugin.enable", map[string]any{"id": plugin.ID}); err != nil {
			return fmt.Errorf("enable %s: %w", plugin.ID, err)
		}
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
