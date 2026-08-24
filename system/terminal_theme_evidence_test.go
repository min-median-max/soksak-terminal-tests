package system

import (
	"strconv"
	"testing"
)

func themeContractFixture() CandidatePresentationData {
	data := CandidatePresentationData{}
	data.ANSI.Base = []string{
		"#2e3436", "#cc0000", "#4e9a06", "#c4a000", "#3465a4", "#75507b", "#06989a", "#d3d7cf",
		"#555753", "#ef2929", "#8ae234", "#fce94f", "#729fcf", "#ad7fa8", "#34e2e2", "#eeeeec",
	}
	data.ANSI.Cube = []int{0, 95, 135, 175, 215, 255}
	data.ANSI.Grayscale.Start = 8
	data.ANSI.Grayscale.Step = 10
	data.ANSI.Grayscale.Count = 24
	data.Theme.Properties.Cursor = "--soksak-terminal-cursor"
	data.Theme.Properties.CursorAccent = "--soksak-terminal-cursor-accent"
	data.Theme.Properties.SelectionBackground = "--soksak-terminal-selection-background"
	data.Theme.Properties.ANSIPrefix = "--soksak-terminal-ansi-"
	return data
}

func TestTerminalThemeEvidenceRequiresComputedDefaultsAndEveryANSIProperty(t *testing.T) {
	contract := themeContractFixture()
	style := map[string]any{
		"color": "rgb(238, 238, 236)", "backgroundColor": "rgb(30, 30, 30)",
		contract.Theme.Properties.Cursor:              "#ffffff",
		contract.Theme.Properties.CursorAccent:        "#1e1e1e",
		contract.Theme.Properties.SelectionBackground: "#555753",
	}
	for index, color := range terminalANSIPalette(contract) {
		style[contract.Theme.Properties.ANSIPrefix+strconv.Itoa(index)] = color
	}
	presentation := map[string]any{"theme": map[string]any{
		"foreground": "#eeeeec", "background": "#1e1e1e", "cursor": "#ffffff",
		"cursorAccent": "#1e1e1e", "selectionBackground": "#555753",
	}}
	evidence, err := readTerminalThemeEvidence(map[string]any{"style": style}, presentation, contract)
	if err != nil {
		t.Fatal(err)
	}
	if len(evidence.ANSI) != 256 || evidence.Computed.Foreground == evidence.Computed.Background {
		t.Fatalf("theme evidence=%+v", evidence)
	}
	delete(style, contract.Theme.Properties.ANSIPrefix+"255")
	if _, err := readTerminalThemeEvidence(map[string]any{"style": style}, presentation, contract); err == nil {
		t.Fatal("missing ANSI property was accepted")
	}
}
