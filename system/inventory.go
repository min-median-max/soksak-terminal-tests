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
		if !ok || !component.Enabled || component.Version != plugin.Version || !filepath.IsAbs(component.Path) {
			return fmt.Errorf("plugin is not active and installed: %s", plugin.ID)
		}
	}
	if len(environment.Sidecars) != len(profile.Sidecars) {
		return fmt.Errorf("sidecar inventory is not exact for %s", profile.Platform)
	}
	for _, expected := range profile.Sidecars {
		component, ok := environment.Sidecars[expected.ID]
		if !ok || component.Version != expected.Version || component.Target != profile.Target || !filepath.IsAbs(component.Path) {
			return fmt.Errorf("sidecar is not installed: %s@%s found=%t version=%q target=%q want=%q path=%q", expected.ID, expected.Version, ok, component.Version, component.Target, profile.Target, component.Path)
		}
	}
	return nil
}
