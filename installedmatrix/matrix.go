// Package installedmatrix verifies the six real terminal providers through Vision's public sok
// commands. It does not start, install, or inspect product implementations.
package installedmatrix

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var canonicalEngines = []string{"alacritty", "ghostty", "kitty", "shitty", "vt100", "wezterm"}

var exactVersion = regexp.MustCompile(`^[0-9]+[.][0-9]+[.][0-9]+$`)

// Caller is the public sok request boundary used by the matrix and by transcript fakes.
type Caller interface {
	Call(command string, params map[string]any) (map[string]any, error)
}

// ArtifactDigester returns the SHA-256 of the executable path published by sidecar.status.
type ArtifactDigester func(path string) (string, error)

type EngineExpectation struct {
	Engine  string `json:"engine"`
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
}

type Config struct {
	Window       string
	HostPane     string
	PromptMarker string
	SnapshotDir  string
	Engines      []EngineExpectation
}

type SidecarIdentity struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Process string `json:"process"`
	PID     int    `json:"pid"`
}

type EngineResult struct {
	Engine         string          `json:"engine"`
	View           string          `json:"view"`
	Pane           string          `json:"pane"`
	Phase          string          `json:"phase"`
	Failure        *string         `json:"failure"`
	RecoveryGaps   int             `json:"recoveryGaps"`
	Sidecar        SidecarIdentity `json:"sidecar"`
	ArtifactSHA256 string          `json:"artifactSha256"`
	Snapshot       string          `json:"snapshot"`
}

type Report struct {
	Window  string         `json:"window"`
	Engines []EngineResult `json:"engines"`
}

// Verify runs one event-backed fresh-tab transaction per engine. It never focuses a tab, narrows
// the read to trailing rows, polls state, or sleeps.
func Verify(config Config, caller Caller, digest ArtifactDigester) (Report, error) {
	expectations, err := validateConfig(config, caller, digest)
	if err != nil {
		return Report{}, err
	}
	report := Report{Window: config.Window, Engines: make([]EngineResult, 0, len(canonicalEngines))}
	for _, engine := range canonicalEngines {
		result, err := verifyEngine(config, caller, digest, expectations[engine])
		if err != nil {
			return Report{}, fmt.Errorf("%s installed matrix: %w", engine, err)
		}
		report.Engines = append(report.Engines, result)
	}
	return report, nil
}

func validateConfig(
	config Config, caller Caller, digest ArtifactDigester,
) (map[string]EngineExpectation, error) {
	if caller == nil {
		return nil, fmt.Errorf("public sok caller is required")
	}
	if digest == nil {
		return nil, fmt.Errorf("sidecar artifact digester is required")
	}
	for name, value := range map[string]string{
		"window": config.Window, "host pane": config.HostPane, "prompt marker": config.PromptMarker,
	} {
		if value == "" {
			return nil, fmt.Errorf("%s is required", name)
		}
	}
	if !filepath.IsAbs(config.SnapshotDir) {
		return nil, fmt.Errorf("snapshot directory must be absolute: %s", config.SnapshotDir)
	}
	info, err := os.Lstat(config.SnapshotDir)
	if err != nil {
		return nil, fmt.Errorf("snapshot directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("snapshot output is not a regular directory: %s", config.SnapshotDir)
	}
	if len(config.Engines) != len(canonicalEngines) {
		return nil, fmt.Errorf("matrix requires exactly six engine expectations")
	}
	expectations := make(map[string]EngineExpectation, len(config.Engines))
	for _, value := range config.Engines {
		if !slices.Contains(canonicalEngines, value.Engine) {
			return nil, fmt.Errorf("unsupported matrix engine: %s", value.Engine)
		}
		if _, exists := expectations[value.Engine]; exists {
			return nil, fmt.Errorf("duplicate matrix engine: %s", value.Engine)
		}
		if !exactVersion.MatchString(value.Version) {
			return nil, fmt.Errorf("%s version is not exact: %s", value.Engine, value.Version)
		}
		decoded, err := hex.DecodeString(value.SHA256)
		if err != nil || len(decoded) != 32 || value.SHA256 != strings.ToLower(value.SHA256) {
			return nil, fmt.Errorf("%s SHA-256 is not exact", value.Engine)
		}
		expectations[value.Engine] = value
	}
	return expectations, nil
}

func verifyEngine(
	config Config, caller Caller, digest ArtifactDigester, expected EngineExpectation,
) (EngineResult, error) {
	window := config.Window
	settings, err := caller.Call("plugin.settings.set", map[string]any{
		"id": "soksak-plugin-terminal-vision", "key": "engine", "value": expected.Engine,
		"scope": "global", "window": window,
	})
	if err != nil {
		return EngineResult{}, err
	}
	if !hasExactKeys(settings, "id", "key", "scope", "value") ||
		settings["id"] != "soksak-plugin-terminal-vision" || settings["key"] != "engine" ||
		settings["scope"] != "global" || settings["value"] != expected.Engine {
		return EngineResult{}, fmt.Errorf("plugin.settings.set returned %+v", settings)
	}

	opened, err := caller.Call("tab.open", map[string]any{
		"program": "terminal-vision", "pane": config.HostPane,
		"mountTimeoutMs": float64(10000), "window": window,
	})
	if err != nil {
		return EngineResult{}, err
	}
	view, _ := opened["tabId"].(string)
	if opened["mounted"] != true || opened["paneId"] != config.HostPane || view == "" {
		return EngineResult{}, fmt.Errorf("tab.open returned %+v", opened)
	}
	pane := view + ".1"
	waited, err := caller.Call("plugin.soksak-plugin-terminal-vision.wait", map[string]any{
		"view": view, "pane": pane, "phase": "live", "contains": config.PromptMarker,
		"timeoutMs": float64(30000), "window": window,
	})
	if err != nil {
		return EngineResult{}, err
	}
	if err := validateLiveWait(waited, expected.Engine, pane); err != nil {
		return EngineResult{}, err
	}
	status, err := caller.Call("plugin.soksak-plugin-terminal-vision.status", map[string]any{
		"view": view, "pane": pane, "window": window,
	})
	if err != nil {
		return EngineResult{}, err
	}
	if err := validateStatus(status, expected.Engine, view, pane); err != nil {
		return EngineResult{}, err
	}
	read, err := caller.Call("plugin.soksak-plugin-terminal-vision.read", map[string]any{
		"view": view, "pane": pane, "window": window,
	})
	if err != nil {
		return EngineResult{}, err
	}
	text, _ := read["text"].(string)
	if !strings.Contains(text, config.PromptMarker) {
		return EngineResult{}, fmt.Errorf("full read omitted prompt marker %q", config.PromptMarker)
	}

	sidecars, err := caller.Call("sidecar.status", map[string]any{"window": window})
	if err != nil {
		return EngineResult{}, err
	}
	identity, actualDigest, err := validateSidecars(sidecars, expected, digest)
	if err != nil {
		return EngineResult{}, err
	}
	settled, err := caller.Call("ui.layout.wait-settled", map[string]any{
		"timeoutMs": float64(10000), "window": window,
	})
	if err != nil {
		return EngineResult{}, err
	}
	if settled["settled"] != true || settled["syncPending"] != false {
		return EngineResult{}, fmt.Errorf("layout did not settle: %+v", settled)
	}
	composition, err := caller.Call("surface.composition", map[string]any{"window": window})
	if err != nil {
		return EngineResult{}, err
	}
	if err := validateComposition(composition); err != nil {
		return EngineResult{}, err
	}
	snapshotPath := filepath.Join(config.SnapshotDir, expected.Engine+".png")
	if info, err := os.Lstat(snapshotPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return EngineResult{}, fmt.Errorf("snapshot path is a symbolic link: %s", snapshotPath)
	} else if err != nil && !os.IsNotExist(err) {
		return EngineResult{}, err
	}
	snapshot, err := caller.Call("window.snapshot", map[string]any{
		"tab": view, "path": snapshotPath, "window": window,
	})
	if err != nil {
		return EngineResult{}, err
	}
	bytes, bytesOK := positiveInt(snapshot["bytes"])
	if snapshot["saved"] != snapshotPath || snapshot["tabId"] != view || !bytesOK || bytes < 1 {
		return EngineResult{}, fmt.Errorf("window.snapshot returned %+v", snapshot)
	}

	return EngineResult{
		Engine: expected.Engine, View: view, Pane: pane, Phase: "live", Failure: nil,
		RecoveryGaps: 0, Sidecar: identity, ArtifactSHA256: actualDigest, Snapshot: snapshotPath,
	}, nil
}

func validateLiveWait(value map[string]any, engine, pane string) error {
	_, failurePresent := value["failure"]
	cols, colsOK := positiveInt(value["cols"])
	rows, rowsOK := positiveInt(value["rows"])
	if value["phase"] != "live" || value["engineId"] != engine || value["pane"] != pane ||
		!failurePresent || value["failure"] != nil || value["fidelity"] != "complete" ||
		!colsOK || !rowsOK || cols < 1 || rows < 1 {
		return fmt.Errorf("Vision wait returned %+v", value)
	}
	return nil
}

func validateStatus(value map[string]any, engine, view, pane string) error {
	_, failurePresent := value["failure"]
	rendered, _ := value["rendered"].(map[string]any)
	output, outputOK := positiveInt(rendered["outputSequence"])
	recovery, _ := value["recovery"].(map[string]any)
	gaps, gapsOK := nonNegativeInt(recovery["gaps"])
	if value["phase"] != "live" || value["engineId"] != engine ||
		value["rendererId"] != "vision" || value["rendererProfile"] != "native-surface" ||
		value["view"] != view || value["pane"] != pane || !failurePresent || value["failure"] != nil ||
		!outputOK || output < 1 || !gapsOK || gaps != 0 {
		return fmt.Errorf("Vision status returned %+v", value)
	}
	return nil
}

func validateSidecars(
	value map[string]any, expected EngineExpectation, digest ArtifactDigester,
) (SidecarIdentity, string, error) {
	units, ok := value["units"].([]any)
	if !ok {
		return SidecarIdentity{}, "", fmt.Errorf("sidecar.status returned %+v", value)
	}
	wanted := "soksak-sidecar-terminal-" + expected.Engine
	var found SidecarIdentity
	for _, raw := range units {
		unit, ok := raw.(map[string]any)
		if !ok || !hasExactKeys(unit, "name", "version", "process", "pid") {
			return SidecarIdentity{}, "", fmt.Errorf("sidecar.status unit is not the exact public identity: %+v", raw)
		}
		name, _ := unit["name"].(string)
		version, _ := unit["version"].(string)
		process, _ := unit["process"].(string)
		pid, pidOK := positiveInt(unit["pid"])
		if name == "" || version == "" || !filepath.IsAbs(process) || !pidOK || pid < 1 {
			return SidecarIdentity{}, "", fmt.Errorf("sidecar.status unit is invalid: %+v", unit)
		}
		if name == wanted {
			found = SidecarIdentity{Name: name, Version: version, Process: process, PID: pid}
		}
	}
	if found.Name == "" {
		return SidecarIdentity{}, "", fmt.Errorf("sidecar.status has no %s", wanted)
	}
	if found.Version != expected.Version {
		return SidecarIdentity{}, "", fmt.Errorf("%s version=%s want=%s", wanted, found.Version, expected.Version)
	}
	actual, err := digest(found.Process)
	if err != nil {
		return SidecarIdentity{}, "", fmt.Errorf("digest %s: %w", found.Process, err)
	}
	actual = strings.ToLower(actual)
	if actual != expected.SHA256 {
		return SidecarIdentity{}, "", fmt.Errorf("%s sha256=%s want=%s", wanted, actual, expected.SHA256)
	}
	return found, actual, nil
}

func validateComposition(value map[string]any) error {
	worst, ok := nonNegativeInt(value["worst"])
	if !ok || worst != 0 {
		return fmt.Errorf("surface.composition worst=%v", value["worst"])
	}
	for _, key := range []string{"unapplied", "undeclared", "misparented"} {
		items, ok := value[key].([]any)
		if !ok || len(items) != 0 {
			return fmt.Errorf("surface.composition %s=%+v", key, value[key])
		}
	}
	return nil
}

func hasExactKeys(value map[string]any, keys ...string) bool {
	if len(value) != len(keys) {
		return false
	}
	for _, key := range keys {
		if _, ok := value[key]; !ok {
			return false
		}
	}
	return true
}

func positiveInt(value any) (int, bool) {
	integer, ok := nonNegativeInt(value)
	return integer, ok && integer > 0
}

func nonNegativeInt(value any) (int, bool) {
	var number float64
	switch typed := value.(type) {
	case float64:
		number = typed
	case int:
		number = float64(typed)
	default:
		return 0, false
	}
	integer := int(number)
	return integer, number >= 0 && number == float64(integer)
}
