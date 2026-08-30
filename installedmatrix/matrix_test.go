package installedmatrix

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"testing"
)

func TestInstalledSixEngineMatrixWaitsForLivePromptBeforeReading(t *testing.T) {
	expected := testExpectations()
	config := Config{
		Window: "w-installed", HostPane: "g-host", PromptMarker: "FRESH_PROMPT_MARKER",
		SnapshotDir: t.TempDir(), Engines: expected,
	}
	transcript := &fakePublicTranscript{t: t, config: config}
	digest := func(path string) (string, error) {
		for _, engine := range expected {
			if path == transcript.process(engine.Engine) {
				return engine.SHA256, nil
			}
		}
		return "", fmt.Errorf("unexpected process path: %s", path)
	}

	report, err := Verify(config, transcript, digest)
	if err != nil {
		t.Fatal(err)
	}
	if transcript.engineIndex != len(canonicalEngines) || transcript.stage != 0 {
		t.Fatalf("public transcript stopped at engine=%d stage=%d", transcript.engineIndex, transcript.stage)
	}
	if len(report.Engines) != 6 {
		t.Fatalf("matrix report has %d engines", len(report.Engines))
	}
	for index, result := range report.Engines {
		want := canonicalEngines[index]
		if result.Engine != want || result.View != "tab-"+want || result.Pane != "tab-"+want+".1" {
			t.Fatalf("matrix result %d = %+v", index, result)
		}
		if result.Snapshot != filepath.Join(config.SnapshotDir, want+".png") {
			t.Fatalf("snapshot %d = %q", index, result.Snapshot)
		}
	}
	for _, call := range transcript.calls {
		if call.command == "tab.activate" || call.command == "view.activate" || call.command == "window.focus" {
			t.Fatalf("matrix focused an inactive tab: %+v", call)
		}
		if slices.Contains([]string{"plugin.soksak-plugin-terminal-vision.read"}, call.command) {
			if _, scoped := call.params["lines"]; scoped {
				t.Fatalf("full prompt read was narrowed to trailing lines: %+v", call)
			}
		}
	}
}

func TestInstalledMatrixRejectsARecoveryGapAndAnExpandedSidecarStatus(t *testing.T) {
	for _, run := range []struct {
		name   string
		mutate func(*fakePublicTranscript)
	}{
		{"recovery gap", func(fake *fakePublicTranscript) { fake.recoveryGaps = 1 }},
		{"sidecar private field", func(fake *fakePublicTranscript) { fake.extraSidecarField = true }},
	} {
		t.Run(run.name, func(t *testing.T) {
			config := Config{
				Window: "w-installed", HostPane: "g-host", PromptMarker: "PROMPT",
				SnapshotDir: t.TempDir(), Engines: testExpectations(),
			}
			fake := &fakePublicTranscript{t: t, config: config}
			run.mutate(fake)
			_, err := Verify(config, fake, func(string) (string, error) {
				return config.Engines[0].SHA256, nil
			})
			if err == nil {
				t.Fatal("invalid public transcript was accepted")
			}
		})
	}
}

func testExpectations() []EngineExpectation {
	versions := map[string]string{
		"alacritty": "0.0.42", "ghostty": "0.0.40", "kitty": "0.0.36",
		"shitty": "0.0.37", "vt100": "0.0.39", "wezterm": "0.0.38",
	}
	result := make([]EngineExpectation, 0, len(canonicalEngines))
	for index, engine := range canonicalEngines {
		result = append(result, EngineExpectation{
			Engine: engine, Version: versions[engine], SHA256: fmt.Sprintf("%064x", index+1),
		})
	}
	return result
}

type transcriptCall struct {
	command string
	params  map[string]any
}

type fakePublicTranscript struct {
	t                 *testing.T
	config            Config
	engineIndex       int
	stage             int
	calls             []transcriptCall
	recoveryGaps      float64
	extraSidecarField bool
}

func (fake *fakePublicTranscript) Call(command string, params map[string]any) (map[string]any, error) {
	fake.t.Helper()
	fake.calls = append(fake.calls, transcriptCall{command: command, params: maps.Clone(params)})
	if fake.engineIndex >= len(canonicalEngines) {
		fake.t.Fatalf("unexpected call after matrix completion: %s %+v", command, params)
	}
	engine := canonicalEngines[fake.engineIndex]
	expectation := expectationByEngine(fake.config.Engines, engine)
	view := "tab-" + engine
	pane := view + ".1"
	window := fake.config.Window
	snapshot := filepath.Join(fake.config.SnapshotDir, engine+".png")

	var wantCommand string
	var wantParams map[string]any
	var answer map[string]any
	switch fake.stage {
	case 0:
		wantCommand = "plugin.settings.set"
		wantParams = map[string]any{
			"id": "soksak-plugin-terminal-vision", "key": "engine", "value": engine,
			"scope": "global", "window": window,
		}
		answer = map[string]any{"id": "soksak-plugin-terminal-vision", "key": "engine", "scope": "global", "value": engine}
	case 1:
		wantCommand = "tab.open"
		wantParams = map[string]any{
			"program": "terminal-vision", "pane": fake.config.HostPane,
			"mountTimeoutMs": float64(10000), "window": window,
		}
		answer = map[string]any{"mounted": true, "paneId": fake.config.HostPane, "tabId": view}
	case 2:
		wantCommand = "plugin.soksak-plugin-terminal-vision.wait"
		wantParams = map[string]any{
			"view": view, "pane": pane, "phase": "live", "contains": fake.config.PromptMarker,
			"timeoutMs": float64(30000), "window": window,
		}
		answer = map[string]any{
			"phase": "live", "engineId": engine, "pane": pane, "failure": nil,
			"fidelity": "complete", "cols": float64(100), "rows": float64(31),
		}
	case 3:
		wantCommand = "plugin.soksak-plugin-terminal-vision.status"
		wantParams = map[string]any{"view": view, "pane": pane, "window": window}
		answer = map[string]any{
			"phase": "live", "engineId": engine, "rendererId": "vision",
			"rendererProfile": "native-surface", "view": view, "pane": pane, "failure": nil,
			"rendered": map[string]any{"outputSequence": float64(315)},
			"recovery": map[string]any{"gaps": fake.recoveryGaps, "outputSequence": float64(315)},
		}
	case 4:
		wantCommand = "plugin.soksak-plugin-terminal-vision.read"
		wantParams = map[string]any{"view": view, "pane": pane, "window": window}
		answer = map[string]any{"text": fake.config.PromptMarker + " $"}
	case 5:
		wantCommand = "sidecar.status"
		wantParams = map[string]any{"window": window}
		unit := map[string]any{
			"name": "soksak-sidecar-terminal-" + engine, "version": expectation.Version,
			"process": fake.process(engine), "pid": float64(4000 + fake.engineIndex),
		}
		if fake.extraSidecarField {
			unit["address"] = "private"
		}
		answer = map[string]any{"units": []any{unit}}
	case 6:
		wantCommand = "ui.layout.wait-settled"
		wantParams = map[string]any{"timeoutMs": float64(10000), "window": window}
		answer = map[string]any{"settled": true, "syncPending": false}
	case 7:
		wantCommand = "surface.composition"
		wantParams = map[string]any{"window": window}
		answer = map[string]any{
			"worst": float64(0), "unapplied": []any{}, "undeclared": []any{}, "misparented": []any{},
		}
	case 8:
		wantCommand = "window.snapshot"
		wantParams = map[string]any{"tab": view, "path": snapshot, "window": window}
		answer = map[string]any{"saved": snapshot, "tabId": view, "bytes": float64(4096)}
	default:
		fake.t.Fatalf("invalid transcript stage %d", fake.stage)
	}
	if command != wantCommand || !maps.EqualFunc(params, wantParams, equalTranscriptValue) {
		fake.t.Fatalf("call engine=%s stage=%d = %s %+v, want %s %+v", engine, fake.stage, command, params, wantCommand, wantParams)
	}
	fake.stage++
	if fake.stage == 9 {
		fake.engineIndex++
		fake.stage = 0
	}
	return answer, nil
}

func (fake *fakePublicTranscript) process(engine string) string {
	return filepath.Join("/installed", "soksak-sidecar-terminal-"+engine)
}

func equalTranscriptValue(left, right any) bool { return fmt.Sprint(left) == fmt.Sprint(right) }

func expectationByEngine(values []EngineExpectation, engine string) EngineExpectation {
	for _, value := range values {
		if value.Engine == engine {
			return value
		}
	}
	return EngineExpectation{}
}
