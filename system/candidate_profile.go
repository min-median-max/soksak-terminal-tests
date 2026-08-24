package system

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/min-median-max/soksak-terminal-tests/fleet"
)

func applyCandidateProfile(profile fleet.Profile, plan CandidatePlan) (fleet.Profile, error) {
	configured := profile
	configured.Plugins = append([]fleet.Plugin(nil), profile.Plugins...)
	configured.Sidecars = append([]fleet.Component(nil), profile.Sidecars...)
	expected := make(map[string]string, len(profile.Plugins)+len(profile.Sidecars))
	for _, plugin := range profile.Plugins {
		expected[plugin.ID] = "plugin"
	}
	for _, sidecar := range profile.Sidecars {
		if _, duplicate := expected[sidecar.ID]; duplicate {
			return fleet.Profile{}, fmt.Errorf("candidate topology repeats component id: %s", sidecar.ID)
		}
		expected[sidecar.ID] = "sidecar"
	}
	seen := make(map[string]bool, len(expected))
	for _, component := range plan.Components {
		kind, exists := expected[component.ID]
		if !exists || kind != component.Kind || seen[component.ID] || component.Version == "" {
			return fleet.Profile{}, fmt.Errorf("candidate component does not match configured topology: %s", component.ID)
		}
		seen[component.ID] = true
		switch component.Kind {
		case "plugin":
			for index := range configured.Plugins {
				if configured.Plugins[index].ID == component.ID {
					configured.Plugins[index].Version = component.Version
					break
				}
			}
		case "sidecar":
			if component.Target != profile.Target {
				return fleet.Profile{}, fmt.Errorf("candidate sidecar target differs: %s=%s want %s", component.ID, component.Target, profile.Target)
			}
			for index := range configured.Sidecars {
				if configured.Sidecars[index].ID == component.ID {
					configured.Sidecars[index].Version = component.Version
					break
				}
			}
		}
	}
	if len(seen) != len(expected) {
		return fleet.Profile{}, fmt.Errorf("candidate component inventory is not exact: found=%d want=%d", len(seen), len(expected))
	}
	return configured, nil
}

func candidatePlanFingerprint(path string) (string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:]), nil
}
