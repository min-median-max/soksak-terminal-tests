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
	EventSequence  float64 `json:"eventSequence"`
	OutputSequence float64 `json:"outputSequence"`
}

type terminalRecoverySize struct {
	terminalSequencedSize
	Gaps float64 `json:"gaps"`
}

type terminalRenderedSize struct {
	terminalSize
	OutputSequence float64 `json:"outputSequence"`
}

type terminalResizeSample struct {
	DOM          terminalDOMSize        `json:"dom"`
	ReportedCols float64                `json:"reportedCols"`
	ReportedRows float64                `json:"reportedRows"`
	Requested    *terminalSize          `json:"requested"`
	PTY          *terminalSequencedSize `json:"pty"`
	Recovery     *terminalRecoverySize  `json:"recovery"`
	Rendered     *terminalRenderedSize  `json:"rendered"`
	Status       map[string]any         `json:"status"`
}

type terminalResizeEvidence struct {
	Plugin          string                `json:"plugin"`
	View            string                `json:"view"`
	NodeAddress     string                `json:"nodeAddress"`
	ResizeReceipt   map[string]any        `json:"resizeReceipt,omitempty"`
	Wide            terminalResizeSample  `json:"wide"`
	Narrow          terminalResizeSample  `json:"narrow"`
	Restored        *terminalResizeSample `json:"restored,omitempty"`
	FailureBoundary string                `json:"failureBoundary,omitempty"`
	Failure         string                `json:"failure,omitempty"`
}

func (evidence terminalResizeEvidence) failureBoundary() string {
	if evidence.Narrow.DOM.Width >= evidence.Wide.DOM.Width {
		return "core-layout"
	}
	if evidence.Narrow.ReportedCols >= evidence.Wide.ReportedCols {
		return "plugin-size"
	}
	if evidence.Narrow.Requested == nil {
		return "plugin-request"
	}
	observed := evidence.Narrow.PTY
	if observed == nil && evidence.Narrow.Recovery != nil {
		copy := evidence.Narrow.Recovery.terminalSequencedSize
		observed = &copy
	}
	if observed == nil || observed.Cols != evidence.Narrow.Requested.Cols ||
		observed.Rows != evidence.Narrow.Requested.Rows {
		return "source-observation"
	}
	if evidence.Narrow.Recovery == nil || evidence.Narrow.Recovery.Gaps != 0 ||
		evidence.Narrow.Recovery.Cols != observed.Cols ||
		evidence.Narrow.Recovery.Rows != observed.Rows ||
		evidence.Narrow.Recovery.EventSequence != observed.EventSequence ||
		evidence.Narrow.Recovery.OutputSequence != observed.OutputSequence {
		return "recovery-observation"
	}
	if evidence.Narrow.Rendered == nil ||
		evidence.Narrow.Rendered.Cols != evidence.Narrow.Recovery.Cols ||
		evidence.Narrow.Rendered.Rows != evidence.Narrow.Recovery.Rows ||
		evidence.Narrow.Rendered.OutputSequence <= 0 ||
		evidence.Narrow.Rendered.OutputSequence > evidence.Narrow.Recovery.OutputSequence {
		return "rendered-frame"
	}
	if evidence.Restored == nil {
		return "restore-observation"
	}
	if evidence.Restored.DOM.Width < evidence.Wide.DOM.Width {
		return "core-layout-restore"
	}
	if evidence.Restored.ReportedCols != evidence.Wide.ReportedCols {
		return "plugin-size-restore"
	}
	return ""
}

func terminalNodeAddress(tree map[string]any, plugin, view, nodePath string) (string, error) {
	nodes, _ := tree["nodes"].([]any)
	for _, dynamic := range []bool{false, true} {
		for _, value := range nodes {
			node, _ := value.(map[string]any)
			address, _ := node["address"].(string)
			path, _ := node["nodePath"].(string)
			matches := path == nodePath || dynamic && strings.HasPrefix(path, nodePath+"/")
			if matches && strings.Contains(address, plugin) && strings.Contains(address, view) {
				return address, nil
			}
		}
	}
	return "", fmt.Errorf("%s view %s exposes no %s", plugin, view, nodePath)
}

func terminalNodePathMatches(actual, declared string) bool {
	return actual == declared || strings.HasPrefix(actual, declared+"/")
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
		overlays, overlayErr := cli.Call("ui.plugin-view.overlay", map[string]any{})
		return terminalResizeSample{}, fmt.Errorf("terminal node has no measurable area: %+v; overlays=%+v overlayErr=%v", measured, overlays, overlayErr)
	}
	status, err := terminal(cli, plugin, "status", view, nil)
	if err != nil {
		sidecars, sidecarErr := cli.Call("sidecar_status", map[string]any{})
		return terminalResizeSample{}, fmt.Errorf("terminal resize status failed after ui.measure=%+v: %w; sidecars=%+v sidecarErr=%v",
			measured, err, sidecars, sidecarErr)
	}
	requested := readTerminalSize(status["requested"])
	pty := readTerminalSequencedSize(status["pty"])
	recovery := readTerminalRecoverySize(status["recovery"])
	rendered := readTerminalRenderedSize(status["rendered"])
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
	eventSequence, eventOK := object["eventSequence"].(float64)
	outputSequence, outputOK := object["outputSequence"].(float64)
	if size == nil || !eventOK || eventSequence < 0 || !outputOK || outputSequence < 0 {
		return nil
	}
	return &terminalSequencedSize{terminalSize: *size, EventSequence: eventSequence, OutputSequence: outputSequence}
}

func readTerminalRecoverySize(value any) *terminalRecoverySize {
	sequenced := readTerminalSequencedSize(value)
	object, _ := value.(map[string]any)
	gaps, ok := object["gaps"].(float64)
	if sequenced == nil || !ok || gaps < 0 {
		return nil
	}
	return &terminalRecoverySize{terminalSequencedSize: *sequenced, Gaps: gaps}
}

func readTerminalRenderedSize(value any) *terminalRenderedSize {
	size := readTerminalSize(value)
	object, _ := value.(map[string]any)
	sequence, ok := object["outputSequence"].(float64)
	if size == nil || !ok || sequence < 0 {
		return nil
	}
	return &terminalRenderedSize{terminalSize: *size, OutputSequence: sequence}
}

func writeTerminalResizeEvidence(directory string, evidence terminalResizeEvidence) error {
	evidence.FailureBoundary = evidence.failureBoundary()
	body, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(directory, evidence.Plugin+"-resize.json"), append(body, '\n'), 0o600)
}
