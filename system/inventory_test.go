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
	environment := inventoryFixture(t, profile)
	if err := ValidateTerminalInventory(profile, environment); err != nil {
		t.Fatal(err)
	}
	delete(environment.Plugins, profile.Plugins[len(profile.Plugins)-1].ID)
	if err := ValidateTerminalInventory(profile, environment); err == nil {
		t.Fatal("six terminal plugins were accepted")
	}
}

func TestTerminalInventoryRejectsMissingPTYAndRecoveryBindings(t *testing.T) {
	profile, _ := fleet.ForTarget("linux", "aarch64-unknown-linux-gnu")
	environment := inventoryFixture(t, profile)
	delete(environment.Sidecars, "soksak-sidecar-pty")
	if err := ValidateTerminalInventory(profile, environment); err == nil {
		t.Fatal("missing PTY sidecar was accepted")
	}
	environment = inventoryFixture(t, profile)
	plugin := environment.Plugins[profile.Plugins[0].ID]
	delete(plugin.Sidecars, "recovery")
	environment.Plugins[profile.Plugins[0].ID] = plugin
	if err := ValidateTerminalInventory(profile, environment); err == nil {
		t.Fatal("missing recovery sidecar selection was accepted")
	}
}

func inventoryFixture(t *testing.T, profile fleet.Profile) platformspec.Environment {
	t.Helper()
	root := t.TempDir()
	environment := platformspec.EmptyEnvironment()
	for _, plugin := range profile.Plugins {
		path := filepath.Join(root, plugin.ID)
		_ = os.MkdirAll(path, 0o700)
		environment.Plugins[plugin.ID] = platformspec.Plugin{
			Component: platformspec.Component{Version: "0.0.1", Path: path, Source: platformspec.RegistrySource, Registry: "test"},
			Enabled:   true, Sidecars: map[string]string{"pty": "soksak-sidecar-pty", "recovery": plugin.Sidecar},
		}
	}
	for _, id := range append([]string{"soksak-sidecar-pty"}, profile.RecoverySidecars...) {
		path := filepath.Join(root, id)
		_ = os.MkdirAll(path, 0o700)
		environment.Sidecars[id] = platformspec.Component{Version: "0.0.1", Path: path, Source: platformspec.RegistrySource, Registry: "test", Target: profile.Target}
	}
	return environment
}
