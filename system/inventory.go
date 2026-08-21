package system

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const SettingsSpec = "soksak-spec-composition@0.0.1"

var TerminalPlugins = []string{
	"soksak-plugin-terminal-alacritty",
	"soksak-plugin-terminal-ghostty",
	"soksak-plugin-terminal-kitty",
	"soksak-plugin-terminal-shitty",
	"soksak-plugin-terminal-vt100",
	"soksak-plugin-terminal-wezterm",
	"soksak-plugin-terminal-xterm",
}

var RecoverySidecars = []string{
	"soksak-sidecar-terminal-alacritty",
	"soksak-sidecar-terminal-ghostty",
	"soksak-sidecar-terminal-kitty",
	"soksak-sidecar-terminal-shitty",
	"soksak-sidecar-terminal-vt100",
	"soksak-sidecar-terminal-wezterm",
}

type Component struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	Enabled     bool   `json:"enabled"`
	InstallPath string `json:"installPath"`
	Manifest    string `json:"manifest"`
}

type Settings struct {
	Spec     string      `json:"spec"`
	Plugins  []Component `json:"plugins"`
	Sidecars []Component `json:"sidecars"`
}

func ReadSettings(path string) (Settings, error) {
	if !filepath.IsAbs(path) {
		return Settings{}, fmt.Errorf("settings path must be absolute: %s", path)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return Settings{}, err
	}
	var settings Settings
	if err := json.Unmarshal(body, &settings); err != nil {
		return Settings{}, err
	}
	if settings.Spec != SettingsSpec {
		return Settings{}, fmt.Errorf("settings spec is %q, expected %q", settings.Spec, SettingsSpec)
	}
	return settings, nil
}

func ValidateTerminalInventory(settings Settings) error {
	if err := validateComponents("plugin", settings.Plugins, TerminalPlugins); err != nil {
		return err
	}
	return validateComponents("sidecar", settings.Sidecars, RecoverySidecars)
}

func validateComponents(kind string, components []Component, expected []string) error {
	index := make(map[string]Component, len(components))
	for _, component := range components {
		if _, exists := index[component.ID]; exists {
			return fmt.Errorf("duplicate %s: %s", kind, component.ID)
		}
		index[component.ID] = component
	}
	for _, id := range expected {
		component, exists := index[id]
		if !exists {
			return fmt.Errorf("missing %s: %s", kind, id)
		}
		if component.Version != "0.0.1" {
			return fmt.Errorf("%s %s has version %s", kind, id, component.Version)
		}
		if !component.Enabled {
			return fmt.Errorf("%s %s is disabled", kind, id)
		}
		if !filepath.IsAbs(component.InstallPath) {
			return fmt.Errorf("%s %s install path is not absolute", kind, id)
		}
		info, err := os.Lstat(filepath.Join(component.InstallPath, filepath.FromSlash(component.Manifest)))
		if err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("%s %s manifest is not a regular file: %v", kind, id, err)
		}
	}
	return nil
}
