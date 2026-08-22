package system

import (
	"os"
	"path/filepath"
	"testing"

	platformspec "github.com/soksak-ai/soksak-spec/go/platformspec"
)

func TestTerminalInventoryRequiresSevenPluginsAndSevenSidecars(t *testing.T) {
	settings, installed := inventoryFixture(t)
	if err := ValidateTerminalInventory(settings, installed); err != nil {
		t.Fatal(err)
	}
	delete(installed.Plugins, TerminalPlugins[len(TerminalPlugins)-1])
	if err := ValidateTerminalInventory(settings, installed); err == nil {
		t.Fatal("six terminal plugins were accepted")
	}
}

func TestTerminalInventoryRejectsMissingPTYAndProviderSelection(t *testing.T) {
	settings, installed := inventoryFixture(t)
	delete(installed.Sidecars, "soksak-sidecar-pty")
	if err := ValidateTerminalInventory(settings, installed); err == nil {
		t.Fatal("missing PTY sidecar was accepted")
	}
	settings, installed = inventoryFixture(t)
	plugin := settings.Plugins[TerminalPlugins[0]]
	delete(plugin.Providers, "terminal-alacritty")
	settings.Plugins[TerminalPlugins[0]] = plugin
	if err := ValidateTerminalInventory(settings, installed); err == nil {
		t.Fatal("missing recovery provider selection was accepted")
	}
}

func inventoryFixture(t *testing.T) (platformspec.Settings, platformspec.Installed) {
	t.Helper()
	root := t.TempDir()
	settings, installed := platformspec.EmptySettings(), platformspec.EmptyInstalled()
	for _, id := range TerminalPlugins {
		engine := id[len("soksak-plugin-terminal-"):]
		provider, requirement := "soksak-sidecar-terminal-"+engine, "terminal-"+engine
		if engine == "xterm" {
			provider, requirement = "soksak-sidecar-terminal-vt100", "terminal-vt100"
		}
		settings.Plugins[id] = platformspec.PluginPreference{Enabled: true, Providers: map[string]string{"pty": "soksak-sidecar-pty", requirement: provider}}
		installed.Plugins[id] = installedFixture(filepath.Join(root, id), false)
	}
	for _, id := range append([]string{"soksak-sidecar-pty"}, RecoverySidecars...) {
		installed.Sidecars[id] = installedFixture(filepath.Join(root, id), true)
	}
	return settings, installed
}

func installedFixture(path string, sidecar bool) platformspec.InstalledComponent {
	_ = os.MkdirAll(path, 0o700)
	target := ""
	if sidecar {
		target = "x86_64-unknown-linux-gnu"
	}
	return platformspec.InstalledComponent{Version: "0.0.1", Path: path, RegistryID: "test", Repository: "https://github.com/example/component", SourceCommit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ManifestSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", ArtifactSHA256: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Target: target}
}
