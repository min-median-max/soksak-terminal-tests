package system

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	platformspec "github.com/soksak-ai/soksak-spec/go/platformspec"
)

var TerminalPlugins = []string{
	"soksak-plugin-terminal-alacritty", "soksak-plugin-terminal-ghostty",
	"soksak-plugin-terminal-kitty", "soksak-plugin-terminal-shitty",
	"soksak-plugin-terminal-vt100", "soksak-plugin-terminal-wezterm",
	"soksak-plugin-terminal-xterm",
}
var RecoverySidecars = []string{
	"soksak-sidecar-terminal-alacritty", "soksak-sidecar-terminal-ghostty",
	"soksak-sidecar-terminal-kitty", "soksak-sidecar-terminal-shitty",
	"soksak-sidecar-terminal-vt100", "soksak-sidecar-terminal-wezterm",
}

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

func ValidateTerminalInventory(settings platformspec.Settings, installed platformspec.Installed) error {
	for _, id := range TerminalPlugins {
		preference, ok := settings.Plugins[id]
		component, installedOK := installed.Plugins[id]
		if !ok || !preference.Enabled || !installedOK || component.Version != "0.0.1" || !filepath.IsAbs(component.Path) {
			return fmt.Errorf("plugin is not active and installed: %s", id)
		}
		if preference.Providers["pty"] != "soksak-sidecar-pty" {
			return fmt.Errorf("plugin has no PTY provider: %s", id)
		}
	}
	for _, id := range append([]string{"soksak-sidecar-pty"}, RecoverySidecars...) {
		component, ok := installed.Sidecars[id]
		if !ok || component.Version != "0.0.1" || component.Target == "" || !filepath.IsAbs(component.Path) {
			return fmt.Errorf("sidecar is not installed: %s", id)
		}
	}
	for _, id := range TerminalPlugins {
		engine := strings.TrimPrefix(id, "soksak-plugin-terminal-")
		provider, requirement := "soksak-sidecar-terminal-"+engine, "terminal-"+engine
		if engine == "xterm" {
			provider, requirement = "soksak-sidecar-terminal-vt100", "terminal-vt100"
		}
		if settings.Plugins[id].Providers[requirement] != provider {
			return fmt.Errorf("plugin provider selection is invalid: %s", id)
		}
	}
	return nil
}
