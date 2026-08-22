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

type terminalResizeSample struct {
	DOM          terminalDOMSize `json:"dom"`
	ReportedCols float64         `json:"reportedCols"`
	ReportedRows float64         `json:"reportedRows"`
	Status       map[string]any  `json:"status"`
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
	cols, _ := status["cols"].(float64)
	rows, _ := status["rows"].(float64)
	return terminalResizeSample{
		DOM:          terminalDOMSize{Width: width, Height: height},
		ReportedCols: cols, ReportedRows: rows, Status: status,
	}, nil
}

func writeTerminalResizeEvidence(directory string, evidence terminalResizeEvidence) error {
	evidence.FailureBoundary = evidence.failureBoundary()
	body, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(directory, evidence.Plugin+"-resize.json"), append(body, '\n'), 0o600)
}
