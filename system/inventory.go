package system

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/min-median-max/soksak-terminal-tests/fleet"
	platformspec "github.com/soksak-ai/soksak-spec/go/platformspec"
)

func ReadEnvironment(environmentPath string) (platformspec.Environment, error) {
	if !filepath.IsAbs(environmentPath) || filepath.Base(environmentPath) != platformspec.EnvironmentFile {
		return platformspec.Environment{}, fmt.Errorf("environment path must be an absolute environment.json path: %s", environmentPath)
	}
	body, err := os.ReadFile(environmentPath)
	if err != nil {
		return platformspec.Environment{}, err
	}
	return platformspec.ParseEnvironment(body)
}

func ValidateTerminalInventory(profile fleet.Profile, environment platformspec.Environment) error {
	if err := platformspec.ValidateEnvironment(environment); err != nil {
		return err
	}
	if len(environment.Plugins) != len(profile.Plugins) {
		return fmt.Errorf("plugin inventory is not exact for %s", profile.Platform)
	}
	for _, plugin := range profile.Plugins {
		component, ok := environment.Plugins[plugin.ID]
		if !ok || !component.Enabled || !filepath.IsAbs(component.Path) {
			return fmt.Errorf("plugin is not active and installed: %s", plugin.ID)
		}
		if component.Sidecars["pty"] != "soksak-sidecar-pty" {
			return fmt.Errorf("plugin has no PTY sidecar: %s", plugin.ID)
		}
		if component.Sidecars["recovery"] != plugin.Sidecar {
			return fmt.Errorf("plugin recovery sidecar selection is invalid: %s", plugin.ID)
		}
	}
	sidecars := append([]string{"soksak-sidecar-pty"}, profile.RecoverySidecars...)
	if len(environment.Sidecars) != len(sidecars) {
		return fmt.Errorf("sidecar inventory is not exact for %s", profile.Platform)
	}
	for _, id := range sidecars {
		component, ok := environment.Sidecars[id]
		if !ok || component.Target != profile.Target || !filepath.IsAbs(component.Path) {
			return fmt.Errorf("sidecar is not installed: %s found=%t target=%q want=%q path=%q", id, ok, component.Target, profile.Target, component.Path)
		}
	}
	return nil
}
