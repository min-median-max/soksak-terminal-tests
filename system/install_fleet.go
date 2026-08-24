package system

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/min-median-max/soksak-terminal-tests/fleet"
	platformspec "github.com/soksak-ai/soksak-spec/go/platformspec"
)

type commandCaller interface {
	Call(command string, params map[string]any) (map[string]any, error)
	CallValue(command string, params map[string]any) (any, error)
}

const installProgressIdleTimeoutMs = 30000
const pluginBootTimeoutMs = 45000

func InstallConfiguredTerminalFleet(profile fleet.Profile, cli commandCaller) error {
	plan := os.Getenv("SOKSAK_TEST_CANDIDATE_PLAN")
	if plan == "" {
		return InstallTerminalFleet(profile, cli)
	}
	if err := InstallCandidateFleet(plan, cli); err != nil {
		return err
	}
	return EnableCandidateTerminalFleet(profile, cli)
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
	return awaitTerminalFleetBoot(cli)
}

func EnableCandidateTerminalFleet(profile fleet.Profile, cli commandCaller) error {
	for _, plugin := range profile.Plugins {
		if err := enableInstalledPlugin(plugin.ID, cli); err != nil {
			return err
		}
	}
	return awaitTerminalFleetBoot(cli)
}

func awaitTerminalFleetBoot(cli commandCaller) error {
	if _, err := cli.Call("plugin.boot.wait", map[string]any{"timeoutMs": pluginBootTimeoutMs}); err != nil {
		return fmt.Errorf("wait for enabled terminal fleet: %w", err)
	}
	return nil
}

func InstallPlugin(pluginID string, cli commandCaller) error {
	if _, err := cli.Call("plugin.install", map[string]any{
		"registryId": "official", "pluginId": pluginID,
	}); err != nil {
		status, statusErr := cli.Call("plugin.install.status", map[string]any{"pluginId": pluginID})
		if statusErr != nil {
			return fmt.Errorf("install %s: %w; status unavailable: %v", pluginID, err, statusErr)
		}
		return fmt.Errorf("install %s: %w; status: %+v", pluginID, err, status)
	}
	if err := awaitArtifactInstall(pluginID, cli); err != nil {
		status, statusErr := cli.Call("plugin.install.status", map[string]any{"pluginId": pluginID})
		if statusErr == nil && installPhase(status) == "installed" {
			return enableInstalledPlugin(pluginID, cli)
		}
		return fmt.Errorf("materialize install %s: %w; plugin status: %+v; status error: %v", pluginID, err, status, statusErr)
	}
	if _, err := cli.Call("plugin.install.wait", map[string]any{
		"pluginId": pluginID, "phase": "installed", "timeoutMs": installProgressIdleTimeoutMs,
	}); err != nil {
		status, statusErr := cli.Call("plugin.install.status", map[string]any{"pluginId": pluginID})
		if statusErr != nil {
			return fmt.Errorf("observe install %s: %w; status unavailable: %v", pluginID, err, statusErr)
		}
		return fmt.Errorf("observe install %s: %w; status: %+v", pluginID, err, status)
	}
	return enableInstalledPlugin(pluginID, cli)
}

func awaitArtifactInstall(pluginID string, cli commandCaller) error {
	var sequence uint64
	for {
		progress, err := cli.Call("artifact_install_wait", map[string]any{
			"rootId": pluginID, "afterSequence": sequence, "timeoutMs": installProgressIdleTimeoutMs,
		})
		if err != nil {
			status, statusErr := cli.Call("artifact_install_status", map[string]any{"rootId": pluginID})
			if statusErr != nil {
				return fmt.Errorf("artifact progress: %w; status unavailable: %v", err, statusErr)
			}
			return fmt.Errorf("artifact progress: %w; status: %+v", err, status)
		}
		next, ok := uint64Value(progress["sequence"])
		if !ok || next <= sequence {
			return fmt.Errorf("artifact progress sequence is not monotonic: previous=%d progress=%+v", sequence, progress)
		}
		sequence = next
		switch progress["phase"] {
		case "committed":
			return nil
		case "rolled-back", "failed":
			return fmt.Errorf("artifact transaction ended at %s: %+v", progress["phase"], progress)
		}
	}
}

func uint64Value(value any) (uint64, bool) {
	switch number := value.(type) {
	case float64:
		if number < 0 || number != float64(uint64(number)) {
			return 0, false
		}
		return uint64(number), true
	case uint64:
		return number, true
	case int:
		if number < 0 {
			return 0, false
		}
		return uint64(number), true
	default:
		return 0, false
	}
}

func installPhase(status map[string]any) string {
	installs, _ := status["installs"].([]any)
	if len(installs) != 1 {
		return ""
	}
	install, _ := installs[0].(map[string]any)
	phase, _ := install["phase"].(string)
	return phase
}

func enableInstalledPlugin(pluginID string, cli commandCaller) error {
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
