package development

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/min-median-max/soksak-terminal-tests/fleet"
	platformspec "github.com/soksak-ai/soksak-spec/go/platformspec"
)

const testCommit = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const testDigest = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

func TestPrepareStateSeparatesSettingsFromInstalledFacts(t *testing.T) {
	input := fixtureInput(t)
	paths, err := PrepareState(input)
	if err != nil {
		t.Fatal(err)
	}
	settingsBody, err := os.ReadFile(paths.Settings)
	if err != nil {
		t.Fatal(err)
	}
	settings, err := platformspec.ParseSettings(settingsBody)
	if err != nil {
		t.Fatal(err)
	}
	installedBody, err := os.ReadFile(paths.Installed)
	if err != nil {
		t.Fatal(err)
	}
	installed, err := platformspec.ParseInstalled(installedBody)
	if err != nil {
		t.Fatal(err)
	}
	if len(settings.Plugins) != 7 || len(settings.Sidecars) != 7 || len(installed.Plugins) != 7 || len(installed.Sidecars) != 7 {
		t.Fatalf("settings=%d/%d installed=%d/%d", len(settings.Plugins), len(settings.Sidecars), len(installed.Plugins), len(installed.Sidecars))
	}
	for id, plugin := range settings.Plugins {
		if !plugin.Enabled || plugin.Development == nil || plugin.Providers["pty"] != "soksak-sidecar-pty" {
			t.Fatalf("invalid plugin preference %s: %+v", id, plugin)
		}
		if installed.Plugins[id].Path != input.Plugins[id].Path {
			t.Fatalf("installed plugin path differs: %s", id)
		}
	}
	if strings.Contains(string(settingsBody), "artifactSha256") || strings.Contains(string(installedBody), "enabled") {
		t.Fatal("settings and installed facts were copied into each other")
	}
}

func TestPrepareStateUsesTheExactWindowsFleet(t *testing.T) {
	input := fixtureInputForPlatform(t, "windows")
	paths, err := PrepareState(input)
	if err != nil {
		t.Fatal(err)
	}
	settingsBody, _ := os.ReadFile(paths.Settings)
	settings, err := platformspec.ParseSettings(settingsBody)
	if err != nil {
		t.Fatal(err)
	}
	if len(settings.Plugins) != 5 || len(settings.Sidecars) != 5 {
		t.Fatalf("Windows settings plugins=%d sidecars=%d", len(settings.Plugins), len(settings.Sidecars))
	}
	for _, id := range []string{"soksak-plugin-terminal-kitty", "soksak-plugin-terminal-shitty"} {
		if _, exists := settings.Plugins[id]; exists {
			t.Fatalf("Windows settings contain unsupported plugin %s", id)
		}
	}
}

func TestPrepareStateIsDeterministic(t *testing.T) {
	input := fixtureInput(t)
	paths, err := PrepareState(input)
	if err != nil {
		t.Fatal(err)
	}
	firstSettings, _ := os.ReadFile(paths.Settings)
	firstInstalled, _ := os.ReadFile(paths.Installed)
	if _, err := PrepareState(input); err != nil {
		t.Fatal(err)
	}
	secondSettings, _ := os.ReadFile(paths.Settings)
	secondInstalled, _ := os.ReadFile(paths.Installed)
	if string(firstSettings) != string(secondSettings) || string(firstInstalled) != string(secondInstalled) {
		t.Fatal("identical input changed platform state bytes")
	}
}

func TestPrepareStateRejectsMissingProcessAndUnverifiedProvenance(t *testing.T) {
	input := fixtureInput(t)
	kitty := input.Sidecars["soksak-sidecar-terminal-kitty"]
	if err := os.Remove(filepath.Join(kitty.Path, "dist", "soksak-sidecar-terminal-kitty")); err != nil {
		t.Fatal(err)
	}
	if _, err := PrepareState(input); err == nil {
		t.Fatal("missing sidecar process was accepted")
	}
	input = fixtureInput(t)
	profile, _ := fleet.ForPlatform(input.Platform)
	plugin := input.Plugins[profile.Plugins[0].ID]
	plugin.Commit = "main"
	input.Plugins[profile.Plugins[0].ID] = plugin
	if _, err := PrepareState(input); err == nil {
		t.Fatal("non-exact source provenance was accepted")
	}
}

func fixtureInput(t *testing.T) Input {
	return fixtureInputForPlatform(t, "linux")
}

func fixtureInputForPlatform(t *testing.T, platform string) Input {
	t.Helper()
	root := t.TempDir()
	profile, err := fleet.ForPlatform(platform)
	if err != nil {
		t.Fatal(err)
	}
	input := Input{Platform: platform, Home: filepath.Join(root, "home"), Plugins: map[string]ArtifactInput{}, Sidecars: map[string]ArtifactInput{}}
	for _, plugin := range profile.Plugins {
		path := filepath.Join(root, plugin.ID)
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatal(err)
		}
		writeJSON(t, filepath.Join(path, "plugin.json"), map[string]any{"id": plugin.ID, "version": version})
		input.Plugins[plugin.ID] = ArtifactInput{Path: path, Repository: "https://github.com/soksak-ai/" + plugin.ID, Commit: testCommit, ArtifactSHA256: testDigest}
	}
	for _, id := range append([]string{"soksak-sidecar-pty"}, profile.RecoverySidecars...) {
		path := filepath.Join(root, id)
		process := filepath.Join("dist", id)
		if err := os.MkdirAll(filepath.Join(path, "dist"), 0o700); err != nil {
			t.Fatal(err)
		}
		writeJSON(t, filepath.Join(path, "sidecar.json"), map[string]any{"id": id, "version": version, "interface": map[string]any{"id": "soksak-spec-sidecar-terminal", "version": version}, "process": filepath.ToSlash(process)})
		if err := os.WriteFile(filepath.Join(path, process), []byte("process"), 0o700); err != nil {
			t.Fatal(err)
		}
		input.Sidecars[id] = ArtifactInput{Path: path, Repository: "https://github.com/soksak-ai/" + id, Commit: testCommit, ArtifactSHA256: testDigest, Target: profile.Target}
	}
	return input
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
}
