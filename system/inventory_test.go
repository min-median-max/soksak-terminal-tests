package system

import (
	"os"
	"path/filepath"
	"strings"
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

func TestTerminalInventoryRejectsMissingOrWrongSidecarComponents(t *testing.T) {
	profile, _ := fleet.ForTarget("linux", "aarch64-unknown-linux-gnu")
	environment := inventoryFixture(t, profile)
	delete(environment.Sidecars, "soksak-sidecar-pty")
	if err := ValidateTerminalInventory(profile, environment); err == nil {
		t.Fatal("missing PTY sidecar was accepted")
	}
	environment = inventoryFixture(t, profile)
	sidecar := environment.Sidecars[profile.Sidecars[1].ID]
	sidecar.Version = "9.9.9"
	environment.Sidecars[profile.Sidecars[1].ID] = sidecar
	if err := ValidateTerminalInventory(profile, environment); err == nil {
		t.Fatal("wrong sidecar version was accepted")
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
			Component: platformspec.Component{Version: plugin.Version, Path: path, ArtifactSHA256: strings.Repeat("a", 64), Source: platformspec.RegistrySource, Registry: "test"},
			Enabled:   true,
		}
	}
	for _, expected := range profile.Sidecars {
		path := filepath.Join(root, expected.ID)
		_ = os.MkdirAll(path, 0o700)
		environment.Sidecars[expected.ID] = platformspec.Component{Version: expected.Version, Path: path, ArtifactSHA256: strings.Repeat("a", 64), Source: platformspec.RegistrySource, Registry: "test", Target: profile.Target}
	}
	return environment
}
