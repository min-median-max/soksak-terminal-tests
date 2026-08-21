package development

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	settingsSpec = "soksak-spec-composition@0.0.1"
	version      = "0.0.1"
)

type Input struct {
	Home     string            `json:"home"`
	Plugins  map[string]string `json:"plugins"`
	Sidecars map[string]string `json:"sidecars"`
}

type Source struct {
	Type string `json:"type"`
	Path string `json:"path"`
}

type Component struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	Enabled     bool   `json:"enabled"`
	Development bool   `json:"development"`
	InstallPath string `json:"installPath"`
	Manifest    string `json:"manifest"`
	Source      Source `json:"source"`
}

type Reference struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type Endpoint struct {
	Plugin  *Reference `json:"plugin,omitempty"`
	Sidecar *Reference `json:"sidecar,omitempty"`
}

type Binding struct {
	Consumer    Endpoint `json:"consumer"`
	Requirement string   `json:"requirement"`
	Provider    Endpoint `json:"provider"`
}

type Settings struct {
	Spec       string      `json:"spec"`
	Generation uint64      `json:"generation"`
	Plugins    []Component `json:"plugins"`
	Sidecars   []Component `json:"sidecars"`
	Kits       []any       `json:"kits"`
	Bindings   []Binding   `json:"bindings"`
}

type provider struct {
	ID          string
	Requirement string
	Sidecar     string
}

var pluginProviders = []provider{
	{"soksak-plugin-terminal-alacritty", "terminal-alacritty", "soksak-sidecar-terminal-alacritty"},
	{"soksak-plugin-terminal-ghostty", "terminal-ghostty", "soksak-sidecar-terminal-ghostty"},
	{"soksak-plugin-terminal-kitty", "terminal-kitty", "soksak-sidecar-terminal-kitty"},
	{"soksak-plugin-terminal-shitty", "terminal-shitty", "soksak-sidecar-terminal-shitty"},
	{"soksak-plugin-terminal-vt100", "terminal-vt100", "soksak-sidecar-terminal-vt100"},
	{"soksak-plugin-terminal-wezterm", "terminal-wezterm", "soksak-sidecar-terminal-wezterm"},
	{"soksak-plugin-terminal-xterm", "terminal-vt100", "soksak-sidecar-terminal-vt100"},
}

var sidecarIDs = []string{
	"soksak-sidecar-pty",
	"soksak-sidecar-terminal-alacritty",
	"soksak-sidecar-terminal-ghostty",
	"soksak-sidecar-terminal-kitty",
	"soksak-sidecar-terminal-shitty",
	"soksak-sidecar-terminal-vt100",
	"soksak-sidecar-terminal-wezterm",
}

func DecodeInput(reader io.Reader) (Input, error) {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	var input Input
	if err := decoder.Decode(&input); err != nil {
		return Input{}, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return Input{}, fmt.Errorf("input has trailing data")
	}
	return input, nil
}

func WriteSettings(input Input) (string, error) {
	if err := validateAbsoluteDirectory(input.Home, false); err != nil {
		return "", fmt.Errorf("home: %w", err)
	}
	settings := Settings{Spec: settingsSpec, Generation: 1, Plugins: []Component{}, Sidecars: []Component{}, Kits: []any{}, Bindings: []Binding{}}
	for _, provider := range pluginProviders {
		root, exists := input.Plugins[provider.ID]
		if !exists {
			return "", fmt.Errorf("missing plugin path: %s", provider.ID)
		}
		if err := validateManifestIdentity(root, "plugin.json", provider.ID, false); err != nil {
			return "", err
		}
		settings.Plugins = append(settings.Plugins, component(provider.ID, root, "plugin.json"))
		settings.Bindings = append(settings.Bindings, binding(provider.ID, "pty", "soksak-sidecar-pty"), binding(provider.ID, provider.Requirement, provider.Sidecar))
	}
	if len(input.Plugins) != len(pluginProviders) {
		return "", fmt.Errorf("plugins must contain exactly %d entries", len(pluginProviders))
	}
	for _, id := range sidecarIDs {
		root, exists := input.Sidecars[id]
		if !exists {
			return "", fmt.Errorf("missing sidecar path: %s", id)
		}
		if err := validateManifestIdentity(root, "sidecar.json", id, true); err != nil {
			return "", err
		}
		settings.Sidecars = append(settings.Sidecars, component(id, root, "sidecar.json"))
	}
	if len(input.Sidecars) != len(sidecarIDs) {
		return "", fmt.Errorf("sidecars must contain exactly %d entries", len(sidecarIDs))
	}
	body, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return "", err
	}
	body = append(body, '\n')
	if err := os.MkdirAll(input.Home, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(input.Home, "settings.json")
	temporary, err := os.CreateTemp(input.Home, "settings-*.json")
	if err != nil {
		return "", err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err == nil {
		_, err = temporary.Write(body)
	}
	if err == nil {
		err = temporary.Sync()
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return "", err
	}
	if err := replaceFile(temporaryPath, path); err != nil {
		return "", err
	}
	return path, nil
}

func component(id, root, manifest string) Component {
	return Component{ID: id, Version: version, Enabled: true, Development: true, InstallPath: root, Manifest: manifest, Source: Source{Type: "path", Path: root}}
}

func binding(plugin, requirement, sidecar string) Binding {
	return Binding{Consumer: Endpoint{Plugin: &Reference{ID: plugin, Version: version}}, Requirement: requirement, Provider: Endpoint{Sidecar: &Reference{ID: sidecar, Version: version}}}
}

func validateManifestIdentity(root, name, id string, processRequired bool) error {
	if err := validateAbsoluteDirectory(root, true); err != nil {
		return fmt.Errorf("%s: %w", id, err)
	}
	path := filepath.Join(root, name)
	body, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s: %w", id, err)
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("%s manifest is not a regular file", id)
	}
	var identity struct {
		ID      string `json:"id"`
		Version string `json:"version"`
		Process string `json:"process"`
	}
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&identity); err != nil {
		return fmt.Errorf("%s: %w", id, err)
	}
	if identity.ID != id || identity.Version != version {
		return fmt.Errorf("%s manifest identity is %s@%s", id, identity.ID, identity.Version)
	}
	if processRequired {
		if identity.Process == "" || filepath.IsAbs(identity.Process) || filepath.Clean(identity.Process) != identity.Process {
			return fmt.Errorf("%s has an invalid process path", id)
		}
		process, err := os.Lstat(filepath.Join(root, filepath.FromSlash(identity.Process)))
		if err != nil || !process.Mode().IsRegular() {
			return fmt.Errorf("%s process is not a regular file", id)
		}
	}
	return nil
}

func validateAbsoluteDirectory(path string, mustExist bool) error {
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return fmt.Errorf("path must be absolute and clean")
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) && !mustExist {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("path must be a directory without a symbolic link")
	}
	return nil
}
