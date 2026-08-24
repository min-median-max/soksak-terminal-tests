package system

import (
	"fmt"
	"strconv"
	"strings"
)

type terminalThemeValues struct {
	Foreground          string `json:"foreground"`
	Background          string `json:"background"`
	Cursor              string `json:"cursor"`
	CursorAccent        string `json:"cursorAccent"`
	SelectionBackground string `json:"selectionBackground"`
}

type terminalThemeEvidence struct {
	Status   terminalThemeValues `json:"status"`
	Computed terminalThemeValues `json:"computed"`
	ANSI     []string            `json:"ansi"`
}

func terminalANSIPalette(contract CandidatePresentationData) []string {
	palette := append([]string(nil), contract.ANSI.Base...)
	for red := 0; red < len(contract.ANSI.Cube); red++ {
		for green := 0; green < len(contract.ANSI.Cube); green++ {
			for blue := 0; blue < len(contract.ANSI.Cube); blue++ {
				palette = append(palette, fmt.Sprintf("#%02x%02x%02x",
					contract.ANSI.Cube[red], contract.ANSI.Cube[green], contract.ANSI.Cube[blue]))
			}
		}
	}
	for index := 0; index < contract.ANSI.Grayscale.Count; index++ {
		channel := contract.ANSI.Grayscale.Start + index*contract.ANSI.Grayscale.Step
		palette = append(palette, fmt.Sprintf("#%02x%02x%02x", channel, channel, channel))
	}
	return palette
}

func terminalThemeMeasureProperties(contract CandidatePresentationData) []string {
	properties := []string{
		"color", "backgroundColor", contract.Theme.Properties.Cursor,
		contract.Theme.Properties.CursorAccent, contract.Theme.Properties.SelectionBackground,
	}
	for index := range terminalANSIPalette(contract) {
		properties = append(properties, contract.Theme.Properties.ANSIPrefix+strconv.Itoa(index))
	}
	return properties
}

func themeValues(value map[string]any, names terminalThemeValues) (terminalThemeValues, error) {
	read := func(name string) string {
		text, _ := value[name].(string)
		return strings.TrimSpace(text)
	}
	result := terminalThemeValues{
		Foreground: read(names.Foreground), Background: read(names.Background), Cursor: read(names.Cursor),
		CursorAccent: read(names.CursorAccent), SelectionBackground: read(names.SelectionBackground),
	}
	if result.Foreground == "" || result.Background == "" || result.Cursor == "" ||
		result.CursorAccent == "" || result.SelectionBackground == "" {
		return terminalThemeValues{}, fmt.Errorf("terminal theme is incomplete: %+v", result)
	}
	if result.Foreground == result.Background {
		return terminalThemeValues{}, fmt.Errorf("terminal foreground equals background: %s", result.Foreground)
	}
	return result, nil
}

func readTerminalThemeEvidence(
	measurement map[string]any,
	presentation map[string]any,
	contract CandidatePresentationData,
) (terminalThemeEvidence, error) {
	statusTheme, _ := presentation["theme"].(map[string]any)
	status, err := themeValues(statusTheme, terminalThemeValues{
		Foreground: "foreground", Background: "background", Cursor: "cursor",
		CursorAccent: "cursorAccent", SelectionBackground: "selectionBackground",
	})
	if err != nil {
		return terminalThemeEvidence{}, fmt.Errorf("status: %w", err)
	}
	style, _ := measurement["style"].(map[string]any)
	computed, err := themeValues(style, terminalThemeValues{
		Foreground: "color", Background: "backgroundColor", Cursor: contract.Theme.Properties.Cursor,
		CursorAccent:        contract.Theme.Properties.CursorAccent,
		SelectionBackground: contract.Theme.Properties.SelectionBackground,
	})
	if err != nil {
		return terminalThemeEvidence{}, fmt.Errorf("computed style: %w", err)
	}
	expected := terminalANSIPalette(contract)
	if len(expected) != 256 {
		return terminalThemeEvidence{}, fmt.Errorf("contract ANSI palette has %d entries", len(expected))
	}
	ansi := make([]string, len(expected))
	for index, color := range expected {
		name := contract.Theme.Properties.ANSIPrefix + strconv.Itoa(index)
		ansi[index], _ = style[name].(string)
		ansi[index] = strings.TrimSpace(ansi[index])
		if ansi[index] != color {
			return terminalThemeEvidence{}, fmt.Errorf("computed style %s=%q, want %q", name, ansi[index], color)
		}
	}
	return terminalThemeEvidence{Status: status, Computed: computed, ANSI: ansi}, nil
}
