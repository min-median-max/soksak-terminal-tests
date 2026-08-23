package system

import "testing"

func TestOutputEvidenceIdentifiesTheFirstFailedBoundary(t *testing.T) {
	evidence := terminalOutputEvidence{}
	if got := evidence.failureBoundary(); got != "pty-output" {
		t.Fatalf("PTY boundary = %q", got)
	}
	evidence.PTY = &terminalSequencedSize{OutputSequence: 100}
	if got := evidence.failureBoundary(); got != "recovery-output" {
		t.Fatalf("recovery boundary = %q", got)
	}
	evidence.Recovery = &terminalRecoverySize{terminalSequencedSize: terminalSequencedSize{OutputSequence: 99}}
	if got := evidence.failureBoundary(); got != "recovery-output" {
		t.Fatalf("recovery sequence boundary = %q", got)
	}
	evidence.Recovery.OutputSequence = 100
	evidence.Recovery.Gaps = 1
	if got := evidence.failureBoundary(); got != "recovery-output" {
		t.Fatalf("gap boundary = %q", got)
	}
	evidence.Recovery.Gaps = 0
	if got := evidence.failureBoundary(); got != "renderer-output" {
		t.Fatalf("renderer boundary = %q", got)
	}
	evidence.Rendered = &terminalRenderedSize{OutputSequence: 99}
	if got := evidence.failureBoundary(); got != "renderer-output" {
		t.Fatalf("renderer sequence boundary = %q", got)
	}
	evidence.Rendered.OutputSequence = 100
	if got := evidence.failureBoundary(); got != "" {
		t.Fatalf("complete boundary = %q", got)
	}
	evidence.MarkerObserved = true
	evidence.PTY.OutputSequence = 103
	evidence.Recovery.OutputSequence = 102
	evidence.Rendered.OutputSequence = 101
	if got := evidence.failureBoundary(); got != "" {
		t.Fatalf("post-marker monotonic boundary = %q", got)
	}
	evidence.Rendered.OutputSequence = 104
	if got := evidence.failureBoundary(); got != "renderer-output" {
		t.Fatalf("renderer-ahead boundary = %q", got)
	}
}
