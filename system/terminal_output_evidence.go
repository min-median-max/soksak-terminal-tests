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
	SourceKind      string                 `json:"sourceKind"`
	Source          *terminalSequencedSize `json:"source"`
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
const highOutputPayloadBytes = 1 << 20

func terminalOutputStatus(plugin, view string, status map[string]any) terminalOutputEvidence {
	pty := readTerminalSequencedSize(status["pty"])
	recovery := readTerminalRecoverySize(status["recovery"])
	sourceKind := "pty"
	source := pty
	if source == nil && recovery != nil {
		sourceKind = "recovery"
		copy := recovery.terminalSequencedSize
		source = &copy
	}
	return terminalOutputEvidence{
		Plugin: plugin, View: view, SourceKind: sourceKind, Source: source, PTY: pty,
		Recovery: recovery, Rendered: readTerminalRenderedSize(status["rendered"]),
	}
}

func (evidence terminalOutputEvidence) failureBoundary() string {
	if evidence.Source == nil {
		return "source-output"
	}
	if evidence.Source.OutputSequence-evidence.BeforeOutput < highOutputPayloadBytes {
		return "source-output"
	}
	if evidence.Recovery == nil || evidence.Recovery.Gaps != 0 ||
		evidence.Recovery.OutputSequence > evidence.Source.OutputSequence ||
		(!evidence.MarkerObserved && evidence.Recovery.OutputSequence < evidence.Source.OutputSequence) {
		return "recovery-output"
	}
	if evidence.Rendered == nil || evidence.Rendered.OutputSequence > evidence.Recovery.OutputSequence ||
		(!evidence.MarkerObserved && evidence.Rendered.OutputSequence < evidence.Recovery.OutputSequence) {
		return "renderer-output"
	}
	if evidence.OutputBytes < highOutputPayloadBytes || evidence.ElapsedMS <= 0 || evidence.ThroughputMBs < minimumCompositionThroughputMBs {
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
