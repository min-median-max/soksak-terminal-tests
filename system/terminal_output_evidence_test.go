package system

import "testing"

func TestOutputEvidenceIdentifiesTheFirstFailedBoundary(t *testing.T) {
	evidence := terminalOutputEvidence{}
	if got := evidence.failureBoundary(); got != "pty-output" {
		t.Fatalf("PTY boundary = %q", got)
	}
	sequence := float64(highOutputPayloadBytes + 100)
	evidence.PTY = &terminalSequencedSize{OutputSequence: sequence}
	if got := evidence.failureBoundary(); got != "recovery-output" {
		t.Fatalf("recovery boundary = %q", got)
	}
	evidence.Recovery = &terminalRecoverySize{terminalSequencedSize: terminalSequencedSize{OutputSequence: sequence - 1}}
	if got := evidence.failureBoundary(); got != "recovery-output" {
		t.Fatalf("recovery sequence boundary = %q", got)
	}
	evidence.Recovery.OutputSequence = sequence
	evidence.Recovery.Gaps = 1
	if got := evidence.failureBoundary(); got != "recovery-output" {
		t.Fatalf("gap boundary = %q", got)
	}
	evidence.Recovery.Gaps = 0
	if got := evidence.failureBoundary(); got != "renderer-output" {
		t.Fatalf("renderer boundary = %q", got)
	}
	evidence.Rendered = &terminalRenderedSize{OutputSequence: sequence - 1}
	if got := evidence.failureBoundary(); got != "renderer-output" {
		t.Fatalf("renderer sequence boundary = %q", got)
	}
	evidence.Rendered.OutputSequence = sequence
	if got := evidence.failureBoundary(); got != "composition-throughput" {
		t.Fatalf("missing throughput boundary = %q", got)
	}
	evidence.ElapsedMS = 50
	evidence.OutputBytes = highOutputPayloadBytes + 100
	evidence.ThroughputMBs = 5.24488
	if got := evidence.failureBoundary(); got != "" {
		t.Fatalf("complete boundary = %q", got)
	}
	evidence.MarkerObserved = true
	evidence.PTY.OutputSequence = sequence + 3
	evidence.Recovery.OutputSequence = sequence + 2
	evidence.Rendered.OutputSequence = sequence + 1
	if got := evidence.failureBoundary(); got != "" {
		t.Fatalf("post-marker monotonic boundary = %q", got)
	}
	evidence.Rendered.OutputSequence = sequence + 4
	if got := evidence.failureBoundary(); got != "renderer-output" {
		t.Fatalf("renderer-ahead boundary = %q", got)
	}
}
