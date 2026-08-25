//go:build system

package system

import (
	"encoding/json"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type terminalThemeReport struct {
	Provider      string                `json:"provider"`
	Delivery      string                `json:"delivery"`
	WindowFocused bool                  `json:"windowFocused"`
	Theme         terminalThemeEvidence `json:"theme"`
}

func TestInstalledTerminalThemeParity(t *testing.T) {
	planPath := os.Getenv("SOKSAK_TEST_CANDIDATE_PLAN")
	if planPath == "" {
		t.Fatal("SOKSAK_TEST_CANDIDATE_PLAN is required")
	}
	plan, _, err := readCandidatePlan(planPath)
	if err != nil {
		t.Fatal(err)
	}
	profile := profileFromEnvironment(t)
	lifecycle, err := NewLifecycle(lifecycleConfigFromEnvironment(t, "theme"))
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
	if _, err := lifecycle.OpenWorkspace(); err != nil {
		t.Fatal(err)
	}
	cli = lifecycle.Client()
	_, pane, err := activeWorkspacePane(cli)
	if err != nil {
		t.Fatal(err)
	}

	reports := make([]terminalThemeReport, 0, len(profile.Plugins))
	pixelReports := make([]map[string]any, 0, len(profile.Plugins))
	var baseline *terminalThemeEvidence
	var pixelBaseline *terminalThemePixels
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
		if screen == "" {
			t.Fatalf("%s has no public screen address", plugin.ID)
		}
		marker := "SOKSAK_THEME_" + strings.ToUpper(strings.TrimPrefix(plugin.ID, "soksak-plugin-terminal-"))
		command, err := terminalPaletteCommand(profile.Platform, marker)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(command, marker) {
			t.Fatalf("%s theme command contains its awaited output marker", plugin.ID)
		}
		if _, err := terminal(cli, plugin.ID, "send", tab, map[string]any{"data": command + "\r"}); err != nil {
			t.Fatalf("send theme fixture %s: %v", plugin.ID, err)
		}
		if _, err := terminal(cli, plugin.ID, "wait", tab, map[string]any{
			"phase": "live", "contains": marker, "timeoutMs": 12000,
		}); err != nil {
			t.Fatalf("theme output %s: %v", plugin.ID, err)
		}
		if err := captureTerminal(cli, "theme-"+plugin.ID); err != nil {
			t.Fatalf("capture %s: %v", plugin.ID, err)
		}
		screenImage := filepath.Join(lifecycle.config.EvidenceDir, "theme-"+plugin.ID+"-screen.png")
		if _, err := cli.Call("window.snapshot", map[string]any{"path": screenImage, "node": screen}); err != nil {
			t.Fatalf("capture %s terminal screen: %v", plugin.ID, err)
		}
		file, err := os.Open(screenImage)
		if err != nil {
			t.Fatalf("open %s terminal screen: %v", plugin.ID, err)
		}
		pixels, decodeErr := png.Decode(file)
		closeErr := file.Close()
		if decodeErr != nil || closeErr != nil {
			t.Fatalf("decode %s terminal screen: decode=%v close=%v", plugin.ID, decodeErr, closeErr)
		}
		pixelTheme, err := readTerminalThemePixels(pixels)
		if err != nil {
			t.Fatalf("%s rendered theme pixels: %v", plugin.ID, err)
		}
		if pixelBaseline == nil {
			copy := pixelTheme
			pixelBaseline = &copy
		} else if err := compareTerminalThemePixels(pixelTheme, *pixelBaseline); err != nil {
			t.Fatalf("%s %v", plugin.ID, err)
		}
		pixelReports = append(pixelReports, map[string]any{
			"provider": plugin.ID, "background": pixelTheme.Background,
			"foreground": pixelTheme.Foreground, "cursor": pixelTheme.Cursor,
		})
		status, err := terminal(cli, plugin.ID, "status", tab, nil)
		if err != nil {
			t.Fatalf("status %s: %v", plugin.ID, err)
		}
		presentation, _ := status["presentation"].(map[string]any)
		measurement, err := cli.Call("ui.measure", map[string]any{
			"address": screen, "props": terminalThemeMeasureProperties(plan.PresentationContract.Data),
		})
		if err != nil {
			t.Fatalf("%s theme measurement: %v", plugin.ID, err)
		}
		theme, err := readTerminalThemeEvidence(measurement, presentation, plan.PresentationContract.Data)
		if err != nil {
			t.Fatalf("%s theme evidence: %v", plugin.ID, err)
		}
		if baseline == nil {
			copy := theme
			baseline = &copy
		} else if !reflect.DeepEqual(*baseline, theme) {
			t.Fatalf("%s theme differs from fleet baseline: got=%+v want=%+v", plugin.ID, theme, *baseline)
		}
		inputState, err := cli.Call("window.input.state", map[string]any{})
		if err != nil {
			t.Fatal(err)
		}
		windowFocused, _ := inputState["windowFocused"].(bool)
		if windowFocused {
			t.Fatalf("%s capture-only window became focused", plugin.ID)
		}
		delivery, _ := presentation["delivery"].(string)
		reports = append(reports, terminalThemeReport{
			Provider: plugin.ID, Delivery: delivery, WindowFocused: windowFocused, Theme: theme,
		})
	}
	body, err := json.MarshalIndent(map[string]any{
		"presentationContract": map[string]any{
			"id": plan.PresentationContract.ID, "version": plan.PresentationContract.Version,
			"sourceRepository": plan.PresentationContract.SourceRepository,
			"sourceCommit":     plan.PresentationContract.SourceCommit,
		},
		"ansiBase": plan.PresentationContract.Data.ANSI.Base, "reports": reports, "pixelReports": pixelReports,
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lifecycle.config.EvidenceDir, "terminal-theme.json"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.Finish(); err != nil {
		t.Fatal(err)
	}
}
