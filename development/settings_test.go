package development

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteSettingsCreatesExactDevelopmentComposition(t *testing.T) {
	input := fixtureInput(t)
	path, err := WriteSettings(input)
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(input.Home, "settings.json") {
		t.Fatalf("path=%s", path)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var settings Settings
	if err := json.Unmarshal(body, &settings); err != nil {
		t.Fatal(err)
	}
	if settings.Spec != settingsSpec || settings.Generation != 1 {
		t.Fatalf("identity=%s generation=%d", settings.Spec, settings.Generation)
	}
	if len(settings.Plugins) != 7 || len(settings.Sidecars) != 7 || len(settings.Kits) != 0 || len(settings.Bindings) != 14 {
		t.Fatalf("plugins=%d sidecars=%d kits=%d bindings=%d", len(settings.Plugins), len(settings.Sidecars), len(settings.Kits), len(settings.Bindings))
	}
	for _, plugin := range settings.Plugins {
		if plugin.Version != version || !plugin.Enabled || !plugin.Development || plugin.Source.Type != "path" || plugin.Source.Path != plugin.InstallPath {
			t.Fatalf("invalid plugin: %+v", plugin)
		}
	}
	for _, sidecar := range settings.Sidecars {
		if sidecar.Version != version || !sidecar.Enabled || !sidecar.Development || sidecar.Source.Type != "path" || sidecar.Source.Path != sidecar.InstallPath {
			t.Fatalf("invalid sidecar: %+v", sidecar)
		}
	}
}

func TestWriteSettingsIsDeterministic(t *testing.T) {
	input := fixtureInput(t)
	path, err := WriteSettings(input)
	if err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(path)
	if _, err := WriteSettings(input); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(path)
	if string(first) != string(second) {
		t.Fatal("identical input changed settings bytes")
	}
}

func TestWriteSettingsRejectsMissingSidecarProcess(t *testing.T) {
	input := fixtureInput(t)
	root := input.Sidecars["soksak-sidecar-terminal-kitty"]
	if err := os.Remove(filepath.Join(root, "dist", "soksak-sidecar-terminal-kitty")); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteSettings(input); err == nil {
		t.Fatal("missing sidecar process was accepted")
	}
}

func fixtureInput(t *testing.T) Input {
	t.Helper()
	root := t.TempDir()
	input := Input{Home: filepath.Join(root, "home"), Plugins: map[string]string{}, Sidecars: map[string]string{}}
	for _, plugin := range pluginProviders {
		path := filepath.Join(root, plugin.ID)
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatal(err)
		}
		body := map[string]any{"id": plugin.ID, "version": version}
		writeJSON(t, filepath.Join(path, "plugin.json"), body)
		input.Plugins[plugin.ID] = path
	}
	for _, sidecar := range sidecarIDs {
		path := filepath.Join(root, sidecar)
		process := filepath.Join("dist", sidecar)
		if err := os.MkdirAll(filepath.Join(path, "dist"), 0o700); err != nil {
			t.Fatal(err)
		}
		body := map[string]any{"spec": "soksak-spec-sidecar@0.0.1", "id": sidecar, "version": version, "interface": map[string]any{"id": "interface", "version": version}, "process": process}
		writeJSON(t, filepath.Join(path, "sidecar.json"), body)
		if err := os.WriteFile(filepath.Join(path, process), []byte("process"), 0o700); err != nil {
			t.Fatal(err)
		}
		input.Sidecars[sidecar] = path
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
