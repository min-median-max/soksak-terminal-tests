package system

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type terminalOutputEvidence struct {
	Plugin          string                 `json:"plugin"`
	View            string                 `json:"view"`
	PTY             *terminalSequencedSize `json:"pty"`
	Recovery        *terminalRecoverySize  `json:"recovery"`
	Rendered        *terminalRenderedSize  `json:"rendered"`
	MarkerObserved  bool                   `json:"markerObserved"`
	BeforeOutput    float64                `json:"beforeOutputSequence"`
	OutputBytes     float64                `json:"outputBytes"`
	ElapsedMS       float64                `json:"elapsedMs"`
	ThroughputMBs   float64                `json:"throughputMbS"`
	FailureBoundary string                 `json:"failureBoundary,omitempty"`
	Failure         string                 `json:"failure,omitempty"`
}

const minimumCompositionThroughputMBs = 3.0

func terminalOutputStatus(plugin, view string, status map[string]any) terminalOutputEvidence {
	return terminalOutputEvidence{
		Plugin: plugin, View: view, PTY: readTerminalSequencedSize(status["pty"]),
		Recovery: readTerminalRecoverySize(status["recovery"]),
		Rendered: readTerminalRenderedSize(status["rendered"]),
	}
}

func (evidence terminalOutputEvidence) failureBoundary() string {
	if evidence.PTY == nil {
		return "pty-output"
	}
	if evidence.PTY.OutputSequence-evidence.BeforeOutput < 262144 {
		return "pty-output"
	}
	if evidence.Recovery == nil || evidence.Recovery.Gaps != 0 ||
		evidence.Recovery.OutputSequence > evidence.PTY.OutputSequence ||
		(!evidence.MarkerObserved && evidence.Recovery.OutputSequence < evidence.PTY.OutputSequence) {
		return "recovery-output"
	}
	if evidence.Rendered == nil || evidence.Rendered.OutputSequence > evidence.Recovery.OutputSequence ||
		(!evidence.MarkerObserved && evidence.Rendered.OutputSequence < evidence.Recovery.OutputSequence) {
		return "renderer-output"
	}
	if evidence.OutputBytes < 262144 || evidence.ElapsedMS <= 0 || evidence.ThroughputMBs < minimumCompositionThroughputMBs {
		return "composition-throughput"
	}
	return ""
}

func writeTerminalOutputEvidence(directory string, evidence terminalOutputEvidence) error {
	evidence.FailureBoundary = evidence.failureBoundary()
	body, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(directory, evidence.Plugin+"-output.json")
	if err := os.WriteFile(path, append(body, '\n'), 0o600); err != nil {
		return fmt.Errorf("write terminal output evidence: %w", err)
	}
	return nil
}
