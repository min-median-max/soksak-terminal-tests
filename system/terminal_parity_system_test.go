//go:build system

package system

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type PresentationParityReport struct {
	Provider              string                `json:"provider"`
	Delivery              string                `json:"delivery"`
	RenderSequence        int                   `json:"renderSequence"`
	AcceptedInputSequence int                   `json:"acceptedInputSequence"`
	PtyWriteSequence      int                   `json:"ptyWriteSequence"`
	FocusedInput          bool                  `json:"focusedInput"`
	CursorVisible         bool                  `json:"cursorVisible"`
	CursorActive          bool                  `json:"cursorActive"`
	LastRenderDurationMs  float64               `json:"lastRenderDurationMs"`
	MaxRenderDurationMs   float64               `json:"maxRenderDurationMs"`
	LastInputToPtyWriteMs float64               `json:"lastInputToPtyWriteMs"`
	WindowFocused         bool                  `json:"windowFocused"`
	Theme                 terminalThemeEvidence `json:"theme"`
}

func TestInstalledTerminalPresentationParity(t *testing.T) {
	planPath := os.Getenv("SOKSAK_TEST_CANDIDATE_PLAN")
	if planPath == "" {
		t.Fatal("SOKSAK_TEST_CANDIDATE_PLAN is required")
	}
	plan, _, err := readCandidatePlan(planPath)
	if err != nil {
		t.Fatal(err)
	}
	profile := profileFromEnvironment(t)
	lifecycle, err := NewLifecycle(lifecycleConfigFromEnvironment(t, "parity"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(lifecycle.Close)
	if err := lifecycle.Start(); err != nil {
		t.Fatal(err)
	}
	cli := lifecycle.Client()
	if err := InstallConfiguredTerminalFleet(profile, cli); err != nil {
		t.Fatal(err)
	}
	budgets := plan.PresentationContract.Data.Budgets
	if _, err := lifecycle.OpenWorkspace(); err != nil {
		t.Fatal(err)
	}
	cli = lifecycle.Client()
	_, pane, err := activeWorkspacePane(cli)
	if err != nil {
		t.Fatal(err)
	}

	reports := make([]PresentationParityReport, 0, len(profile.Plugins))
	var baselineTheme *terminalThemeEvidence
	for _, plugin := range profile.Plugins {
		program := strings.TrimPrefix(plugin.ID, "soksak-plugin-")
		opened, err := cli.Call("tab.open", map[string]any{"pane": pane, "program": program, "mountTimeoutMs": 12000})
		if err != nil {
			t.Fatalf("open %s: %v", plugin.ID, err)
		}
		tab, _ := opened["tabId"].(string)
		if tab == "" {
			t.Fatalf("open %s returned no tab", plugin.ID)
		}
		if _, err := cli.Call("tab.activate", map[string]any{"tab": tab}); err != nil {
			t.Fatal(err)
		}
		if _, err := cli.Call("plugin."+plugin.ID+".wait", map[string]any{"phase": "live", "view": tab, "timeoutMs": 12000}); err != nil {
			t.Fatalf("wait %s: %v", plugin.ID, err)
		}
		nodes, err := exposedNodes(cli)
		if err != nil {
			t.Fatal(err)
		}
		screen := nodeAddress(nodes, "terminal-screen", plugin.ID+"\x00"+tab)
		input := nodeAddress(nodes, "terminal-input", plugin.ID+"\x00"+tab)
		if screen == "" || input == "" {
			t.Fatalf("%s has no public screen/input address", plugin.ID)
		}
		if _, err := cli.Call("ui.input.click", map[string]any{"address": screen}); err != nil {
			t.Fatalf("focus %s: %v", plugin.ID, err)
		}
		marker := "SOKSAK_PARITY_" + strings.ToUpper(strings.TrimPrefix(plugin.ID, "soksak-plugin-terminal-"))
		command, err := terminalPaletteCommand(profile.Platform, marker)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(command, marker) {
			t.Fatalf("%s palette input contains its awaited output marker", plugin.ID)
		}
		for _, key := range command {
			if _, err := cli.Call("ui.input.key", map[string]any{"address": input, "key": string(key)}); err != nil {
				t.Fatalf("type %s key %q: %v", plugin.ID, key, err)
			}
		}
		if _, err := cli.Call("ui.input.key", map[string]any{"address": input, "key": "Enter"}); err != nil {
			t.Fatalf("enter %s: %v", plugin.ID, err)
		}
		if _, err := cli.Call("plugin."+plugin.ID+".wait", map[string]any{
			"phase": "live", "view": tab, "contains": marker, "timeoutMs": 12000,
		}); err != nil {
			status, statusErr := cli.Call("plugin."+plugin.ID+".status", map[string]any{"view": tab})
			read, readErr := cli.Call("plugin."+plugin.ID+".read", map[string]any{"view": tab, "lines": 8})
			t.Fatalf("keyboard round trip %s: %v; status=%+v statusErr=%v; read=%+v readErr=%v",
				plugin.ID, err, status, statusErr, read, readErr)
		}
		if err := captureTerminal(cli, "parity-"+plugin.ID); err != nil {
			t.Fatalf("capture %s: %v", plugin.ID, err)
		}
		terminalImage := filepath.Join(lifecycle.config.EvidenceDir, "parity-"+plugin.ID+"-screen.png")
		if _, err := cli.Call("window.snapshot", map[string]any{"path": terminalImage, "node": screen}); err != nil {
			t.Fatalf("capture %s terminal screen: %v", plugin.ID, err)
		}
		status, err := cli.Call("plugin."+plugin.ID+".status", map[string]any{"view": tab})
		if err != nil {
			t.Fatalf("status %s: %v", plugin.ID, err)
		}
		presentation, _ := status["presentation"].(map[string]any)
		report, err := presentationReport(plugin.ID, presentation)
		if err != nil {
			t.Errorf("%s presentation report: %v; status=%+v", plugin.ID, err, presentation)
			report = PresentationParityReport{Provider: plugin.ID}
			report.Delivery, _ = presentation["delivery"].(string)
		}
		measurement, err := cli.Call("ui.measure", map[string]any{
			"address": screen, "props": terminalThemeMeasureProperties(plan.PresentationContract.Data),
		})
		if err != nil {
			t.Errorf("%s theme measurement: %v", plugin.ID, err)
		} else {
			report.Theme, err = readTerminalThemeEvidence(measurement, presentation, plan.PresentationContract.Data)
			if err != nil {
				t.Errorf("%s theme evidence: %v", plugin.ID, err)
			} else if baselineTheme == nil {
				copy := report.Theme
				baselineTheme = &copy
			} else if !reflect.DeepEqual(*baselineTheme, report.Theme) {
				t.Errorf("%s theme differs from fleet baseline: got=%+v want=%+v", plugin.ID, report.Theme, *baselineTheme)
			}
		}
		inputState, err := cli.Call("window.input.state", map[string]any{})
		if err != nil {
			t.Fatal(err)
		}
		report.WindowFocused, _ = inputState["windowFocused"].(bool)
		if report.WindowFocused {
			t.Errorf("%s capture-only window became focused", plugin.ID)
		}
		if !report.FocusedInput || !report.CursorVisible || !report.CursorActive ||
			report.RenderSequence < 1 || report.AcceptedInputSequence < 2 || report.PtyWriteSequence < 2 {
			t.Errorf("%s presentation state=%+v", plugin.ID, report)
		}
		if report.MaxRenderDurationMs > budgets.RenderMs {
			t.Errorf("%s render %.3fms exceeds %.3fms", plugin.ID, report.MaxRenderDurationMs, budgets.RenderMs)
		}
		if report.LastInputToPtyWriteMs > budgets.InputToPtyWriteMs {
			t.Errorf("%s input-to-PTY %.3fms exceeds %.3fms", plugin.ID, report.LastInputToPtyWriteMs, budgets.InputToPtyWriteMs)
		}
		reports = append(reports, report)
	}
	body, err := json.MarshalIndent(map[string]any{
		"presentationContract": map[string]any{
			"id": plan.PresentationContract.ID, "version": plan.PresentationContract.Version,
			"sourceRepository": plan.PresentationContract.SourceRepository,
			"sourceCommit":     plan.PresentationContract.SourceCommit,
		},
		"budgets": budgets, "ansiBase": plan.PresentationContract.Data.ANSI.Base, "reports": reports,
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lifecycle.config.EvidenceDir, "terminal-parity.json"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.Finish(); err != nil {
		t.Fatal(err)
	}
}

func presentationReport(provider string, value map[string]any) (PresentationParityReport, error) {
	if value == nil {
		return PresentationParityReport{}, fmt.Errorf("%s status has no presentation", provider)
	}
	report := PresentationParityReport{Provider: provider}
	report.Delivery, _ = value["delivery"].(string)
	report.RenderSequence, _ = exactInt(value["renderSequence"])
	report.AcceptedInputSequence, _ = exactInt(value["acceptedInputSequence"])
	report.PtyWriteSequence, _ = exactInt(value["ptyWriteSequence"])
	report.FocusedInput, _ = value["focusedInput"].(bool)
	report.CursorVisible, _ = value["cursorVisible"].(bool)
	report.CursorActive, _ = value["cursorActive"].(bool)
	var ok bool
	if report.LastRenderDurationMs, ok = number(value["lastRenderDurationMs"]); !ok {
		return PresentationParityReport{}, fmt.Errorf("%s has no last render duration", provider)
	}
	if report.MaxRenderDurationMs, ok = number(value["maxRenderDurationMs"]); !ok {
		return PresentationParityReport{}, fmt.Errorf("%s has no max render duration", provider)
	}
	if report.LastInputToPtyWriteMs, ok = number(value["lastInputToPtyWriteMs"]); !ok {
		return PresentationParityReport{}, fmt.Errorf("%s has no input-to-PTY duration", provider)
	}
	return report, nil
}
