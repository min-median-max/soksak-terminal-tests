package system

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/min-median-max/soksak-terminal-tests/fleet"
	platformspec "github.com/soksak-ai/soksak-spec/go/platformspec"
)

func ReadState(settingsPath string) (platformspec.Settings, platformspec.Installed, error) {
	if !filepath.IsAbs(settingsPath) || filepath.Base(settingsPath) != platformspec.SettingsFile {
		return platformspec.Settings{}, platformspec.Installed{}, fmt.Errorf("settings path must be an absolute settings.json path: %s", settingsPath)
	}
	settingsBody, err := os.ReadFile(settingsPath)
	if err != nil {
		return platformspec.Settings{}, platformspec.Installed{}, err
	}
	settings, err := platformspec.ParseSettings(settingsBody)
	if err != nil {
		return platformspec.Settings{}, platformspec.Installed{}, err
	}
	installedBody, err := os.ReadFile(filepath.Join(filepath.Dir(settingsPath), platformspec.InstalledFile))
	if err != nil {
		return platformspec.Settings{}, platformspec.Installed{}, err
	}
	installed, err := platformspec.ParseInstalled(installedBody)
	return settings, installed, err
}

func ValidateTerminalInventory(profile fleet.Profile, settings platformspec.Settings, installed platformspec.Installed) error {
	if len(settings.Plugins) != len(profile.Plugins) || len(installed.Plugins) != len(profile.Plugins) {
		return fmt.Errorf("plugin inventory is not exact for %s", profile.Platform)
	}
	for _, plugin := range profile.Plugins {
		preference, ok := settings.Plugins[plugin.ID]
		component, installedOK := installed.Plugins[plugin.ID]
		if !ok || !preference.Enabled || !installedOK || component.Version != "0.0.1" || !filepath.IsAbs(component.Path) {
			return fmt.Errorf("plugin is not active and installed: %s", plugin.ID)
		}
		if preference.Providers["pty"] != "soksak-sidecar-pty" {
			return fmt.Errorf("plugin has no PTY provider: %s", plugin.ID)
		}
	}
	sidecars := append([]string{"soksak-sidecar-pty"}, profile.RecoverySidecars...)
	if len(installed.Sidecars) != len(sidecars) {
		return fmt.Errorf("sidecar inventory is not exact for %s", profile.Platform)
	}
	for _, id := range sidecars {
		component, ok := installed.Sidecars[id]
		if !ok || component.Version != "0.0.1" || component.Target != profile.Target || !filepath.IsAbs(component.Path) {
			return fmt.Errorf("sidecar is not installed: %s", id)
		}
	}
	for _, plugin := range profile.Plugins {
		if settings.Plugins[plugin.ID].Providers[plugin.Requirement] != plugin.Sidecar {
			return fmt.Errorf("plugin provider selection is invalid: %s", plugin.ID)
		}
	}
	return nil
}
