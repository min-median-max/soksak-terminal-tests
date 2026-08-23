package system

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/min-median-max/soksak-terminal-tests/fleet"
	platformspec "github.com/soksak-ai/soksak-spec/go/platformspec"
)

func TestTerminalInventoryRequiresSevenPluginsAndSevenSidecars(t *testing.T) {
	profile, _ := fleet.ForTarget("linux", "aarch64-unknown-linux-gnu")
	settings, installed := inventoryFixture(t, profile)
	if err := ValidateTerminalInventory(profile, settings, installed); err != nil {
		t.Fatal(err)
	}
	delete(installed.Plugins, profile.Plugins[len(profile.Plugins)-1].ID)
	if err := ValidateTerminalInventory(profile, settings, installed); err == nil {
		t.Fatal("six terminal plugins were accepted")
	}
}

func TestTerminalInventoryRejectsMissingPTYAndProviderSelection(t *testing.T) {
	profile, _ := fleet.ForTarget("linux", "aarch64-unknown-linux-gnu")
	settings, installed := inventoryFixture(t, profile)
	delete(installed.Sidecars, "soksak-sidecar-pty")
	if err := ValidateTerminalInventory(profile, settings, installed); err == nil {
		t.Fatal("missing PTY sidecar was accepted")
	}
	settings, installed = inventoryFixture(t, profile)
	plugin := settings.Plugins[profile.Plugins[0].ID]
	delete(plugin.Providers, "terminal-alacritty")
	settings.Plugins[profile.Plugins[0].ID] = plugin
	if err := ValidateTerminalInventory(profile, settings, installed); err == nil {
		t.Fatal("missing recovery provider selection was accepted")
	}
}

func TestTerminalInventoryAcceptsValidatedComponentPatchVersions(t *testing.T) {
	profile, _ := fleet.ForTarget("windows", "x86_64-pc-windows-msvc")
	settings, installed := inventoryFixture(t, profile)
	plugin := installed.Plugins["soksak-plugin-terminal-ghostty"]
	plugin.Version = "0.0.2"
	installed.Plugins["soksak-plugin-terminal-ghostty"] = plugin
	if err := ValidateTerminalInventory(profile, settings, installed); err != nil {
		t.Fatal(err)
	}
}

func inventoryFixture(t *testing.T, profile fleet.Profile) (platformspec.Settings, platformspec.Installed) {
	t.Helper()
	root := t.TempDir()
	settings, installed := platformspec.EmptySettings(), platformspec.EmptyInstalled()
	for _, plugin := range profile.Plugins {
		settings.Plugins[plugin.ID] = platformspec.PluginPreference{Enabled: true, Providers: map[string]string{"pty": "soksak-sidecar-pty", plugin.Requirement: plugin.Sidecar}}
		installed.Plugins[plugin.ID] = installedFixture(filepath.Join(root, plugin.ID), "")
	}
	for _, id := range append([]string{"soksak-sidecar-pty"}, profile.RecoverySidecars...) {
		installed.Sidecars[id] = installedFixture(filepath.Join(root, id), profile.Target)
	}
	return settings, installed
}

func installedFixture(path, target string) platformspec.InstalledComponent {
	_ = os.MkdirAll(path, 0o700)
	return platformspec.InstalledComponent{Version: "0.0.1", Path: path, RegistryID: "test", Repository: "https://github.com/example/component", SourceCommit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ManifestSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", ArtifactSHA256: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Target: target}
}
