package system

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/min-median-max/soksak-terminal-tests/fleet"
	platformspec "github.com/soksak-ai/soksak-spec/go/platformspec"
)

type runtimeEnvironmentFixture struct{}

func (runtimeEnvironmentFixture) Call(string, map[string]any) (map[string]any, error) {
	return nil, nil
}

func (runtimeEnvironmentFixture) CallValue(string, map[string]any) (any, error) {
	return map[string]any{
		"revision": float64(1),
		"plugins":  map[string]any{},
		"sidecars": map[string]any{"terminal-runtime": map[string]any{
			"version": "1.0.0", "path": "/runtime/terminal-runtime", "process": "/runtime/terminal-runtime/process",
			"artifactSha256": strings.Repeat("a", 64), "source": "local", "target": "aarch64-apple-darwin",
		}},
	}, nil
}

func TestRuntimeEnvironmentAcceptsTheContractProcessField(t *testing.T) {
	environment, err := ReadRuntimeEnvironment(runtimeEnvironmentFixture{})
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := environment.Sidecars["terminal-runtime"]; !exists {
		t.Fatal("runtime sidecar was not decoded")
	}
}

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
		environment.Sidecars[expected.ID] = platformspec.Component{Version: expected.Version, Path: path, Process: filepath.Join(path, "process"), ArtifactSHA256: strings.Repeat("a", 64), Source: platformspec.RegistrySource, Registry: "test", Target: profile.Target}
	}
	return environment
}
