package system

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type terminalDOMSize struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type terminalSize struct {
	Cols float64 `json:"cols"`
	Rows float64 `json:"rows"`
}

type terminalSequencedSize struct {
	terminalSize
	EventSequence float64 `json:"eventSequence"`
}

type terminalResizeSample struct {
	DOM          terminalDOMSize        `json:"dom"`
	ReportedCols float64                `json:"reportedCols"`
	ReportedRows float64                `json:"reportedRows"`
	Requested    *terminalSize          `json:"requested"`
	PTY          *terminalSequencedSize `json:"pty"`
	Recovery     *terminalSequencedSize `json:"recovery"`
	Rendered     *terminalSize          `json:"rendered"`
	Status       map[string]any         `json:"status"`
}

type terminalResizeEvidence struct {
	Plugin          string               `json:"plugin"`
	View            string               `json:"view"`
	NodeAddress     string               `json:"nodeAddress"`
	ResizeReceipt   map[string]any       `json:"resizeReceipt,omitempty"`
	Wide            terminalResizeSample `json:"wide"`
	Narrow          terminalResizeSample `json:"narrow"`
	FailureBoundary string               `json:"failureBoundary,omitempty"`
	Failure         string               `json:"failure,omitempty"`
}

func (evidence terminalResizeEvidence) failureBoundary() string {
	if evidence.Narrow.DOM.Width >= evidence.Wide.DOM.Width {
		return "core-layout"
	}
	if evidence.Narrow.ReportedCols >= evidence.Wide.ReportedCols {
		return "plugin-size"
	}
	if evidence.Narrow.Requested == nil || evidence.Narrow.PTY == nil ||
		evidence.Narrow.PTY.Cols != evidence.Narrow.Requested.Cols ||
		evidence.Narrow.PTY.Rows != evidence.Narrow.Requested.Rows {
		return "pty-observation"
	}
	if evidence.Narrow.Recovery == nil ||
		evidence.Narrow.Recovery.Cols != evidence.Narrow.PTY.Cols ||
		evidence.Narrow.Recovery.Rows != evidence.Narrow.PTY.Rows ||
		evidence.Narrow.Recovery.EventSequence != evidence.Narrow.PTY.EventSequence {
		return "recovery-observation"
	}
	if evidence.Narrow.Rendered == nil ||
		evidence.Narrow.Rendered.Cols != evidence.Narrow.Recovery.Cols ||
		evidence.Narrow.Rendered.Rows != evidence.Narrow.Recovery.Rows {
		return "rendered-frame"
	}
	return ""
}

func terminalNodeAddress(tree map[string]any, plugin, view, nodePath string) (string, error) {
	nodes, _ := tree["nodes"].([]any)
	for _, value := range nodes {
		node, _ := value.(map[string]any)
		address, _ := node["address"].(string)
		path, _ := node["nodePath"].(string)
		if path == nodePath && strings.Contains(address, plugin) && strings.Contains(address, view) {
			return address, nil
		}
	}
	return "", fmt.Errorf("%s view %s exposes no %s", plugin, view, nodePath)
}

func measureTerminalResize(cli CLI, plugin, view, address string) (terminalResizeSample, error) {
	measured, err := cli.Call("ui.measure", map[string]any{"address": address})
	if err != nil {
		return terminalResizeSample{}, err
	}
	rect, _ := measured["rect"].(map[string]any)
	width, _ := rect["w"].(float64)
	height, _ := rect["h"].(float64)
	if width <= 0 || height <= 0 {
		return terminalResizeSample{}, fmt.Errorf("terminal node has no measurable area: %+v", measured)
	}
	status, err := terminal(cli, plugin, "status", view, nil)
	if err != nil {
		return terminalResizeSample{}, err
	}
	requested := readTerminalSize(status["requested"])
	pty := readTerminalSequencedSize(status["pty"])
	recovery := readTerminalSequencedSize(status["recovery"])
	rendered := readTerminalSize(status["rendered"])
	var cols, rows float64
	if rendered != nil {
		cols, rows = rendered.Cols, rendered.Rows
	}
	return terminalResizeSample{
		DOM:          terminalDOMSize{Width: width, Height: height},
		ReportedCols: cols, ReportedRows: rows, Requested: requested, PTY: pty,
		Recovery: recovery, Rendered: rendered, Status: status,
	}, nil
}

func readTerminalSize(value any) *terminalSize {
	object, _ := value.(map[string]any)
	cols, colsOK := object["cols"].(float64)
	rows, rowsOK := object["rows"].(float64)
	if !colsOK || !rowsOK || cols <= 0 || rows <= 0 {
		return nil
	}
	return &terminalSize{Cols: cols, Rows: rows}
}

func readTerminalSequencedSize(value any) *terminalSequencedSize {
	size := readTerminalSize(value)
	object, _ := value.(map[string]any)
	sequence, ok := object["eventSequence"].(float64)
	if size == nil || !ok || sequence < 0 {
		return nil
	}
	return &terminalSequencedSize{terminalSize: *size, EventSequence: sequence}
}

func writeTerminalResizeEvidence(directory string, evidence terminalResizeEvidence) error {
	evidence.FailureBoundary = evidence.failureBoundary()
	body, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(directory, evidence.Plugin+"-resize.json"), append(body, '\n'), 0o600)
}
