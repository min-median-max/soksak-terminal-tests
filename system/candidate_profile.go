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
	configured.Plugins = nil
	configured.Sidecars = nil
	seen := make(map[string]bool, len(plan.Components))
	for _, component := range plan.Components {
		if seen[component.ID] || component.Version == "" {
			return fleet.Profile{}, fmt.Errorf("candidate topology repeats component id: %s", component.ID)
		}
		seen[component.ID] = true
		switch component.Kind {
		case "plugin":
			if component.Program == "" {
				return fleet.Profile{}, fmt.Errorf("candidate plugin has no program: %s", component.ID)
			}
			configured.Plugins = append(configured.Plugins, fleet.Plugin{
				Component: fleet.Component{ID: component.ID, Version: component.Version}, Program: component.Program,
			})
		case "sidecar":
			if component.Target != profile.Target {
				return fleet.Profile{}, fmt.Errorf("candidate sidecar target differs: %s=%s want %s", component.ID, component.Target, profile.Target)
			}
			configured.Sidecars = append(configured.Sidecars, fleet.Component{ID: component.ID, Version: component.Version})
		default:
			return fleet.Profile{}, fmt.Errorf("candidate component kind is invalid: %s=%s", component.ID, component.Kind)
		}
	}
	if len(configured.Plugins) == 0 || len(configured.Sidecars) == 0 {
		return fleet.Profile{}, fmt.Errorf("candidate topology is incomplete")
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
