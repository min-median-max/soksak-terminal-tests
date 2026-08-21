package system

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTerminalInventoryRequiresSevenPluginsAndSixSidecars(t *testing.T) {
	root := t.TempDir()
	settings := Settings{Spec: SettingsSpec}
	for _, id := range TerminalPlugins {
		settings.Plugins = append(settings.Plugins, componentFixture(t, root, id, "plugin.json"))
	}
	for _, id := range RecoverySidecars {
		settings.Sidecars = append(settings.Sidecars, componentFixture(t, root, id, "sidecar.json"))
	}
	if err := ValidateTerminalInventory(settings); err != nil {
		t.Fatal(err)
	}
	settings.Plugins = settings.Plugins[:len(settings.Plugins)-1]
	if err := ValidateTerminalInventory(settings); err == nil {
		t.Fatal("six terminal plugins were accepted")
	}
}

func TestTerminalInventoryRejectsAChangedVersion(t *testing.T) {
	root := t.TempDir()
	settings := Settings{Spec: SettingsSpec}
	for _, id := range TerminalPlugins {
		settings.Plugins = append(settings.Plugins, componentFixture(t, root, id, "plugin.json"))
	}
	for _, id := range RecoverySidecars {
		settings.Sidecars = append(settings.Sidecars, componentFixture(t, root, id, "sidecar.json"))
	}
	settings.Sidecars[0].Version = "0.0.0"
	if err := ValidateTerminalInventory(settings); err == nil {
		t.Fatal("a sidecar below 0.0.1 was accepted")
	}
}

func componentFixture(t *testing.T, root, id, manifest string) Component {
	t.Helper()
	installPath := filepath.Join(root, id)
	if err := os.MkdirAll(installPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installPath, manifest), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	return Component{ID: id, Version: "0.0.1", Enabled: true, InstallPath: installPath, Manifest: manifest}
}
