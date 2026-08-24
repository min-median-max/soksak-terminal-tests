package system

import "testing"

func TestOutputEvidenceIdentifiesTheFirstFailedBoundary(t *testing.T) {
	evidence := terminalOutputEvidence{}
	if got := evidence.failureBoundary(); got != "pty-output" {
		t.Fatalf("PTY boundary = %q", got)
	}
	evidence.PTY = &terminalSequencedSize{OutputSequence: 262244}
	if got := evidence.failureBoundary(); got != "recovery-output" {
		t.Fatalf("recovery boundary = %q", got)
	}
	evidence.Recovery = &terminalRecoverySize{terminalSequencedSize: terminalSequencedSize{OutputSequence: 262243}}
	if got := evidence.failureBoundary(); got != "recovery-output" {
		t.Fatalf("recovery sequence boundary = %q", got)
	}
	evidence.Recovery.OutputSequence = 262244
	evidence.Recovery.Gaps = 1
	if got := evidence.failureBoundary(); got != "recovery-output" {
		t.Fatalf("gap boundary = %q", got)
	}
	evidence.Recovery.Gaps = 0
	if got := evidence.failureBoundary(); got != "renderer-output" {
		t.Fatalf("renderer boundary = %q", got)
	}
	evidence.Rendered = &terminalRenderedSize{OutputSequence: 262243}
	if got := evidence.failureBoundary(); got != "renderer-output" {
		t.Fatalf("renderer sequence boundary = %q", got)
	}
	evidence.Rendered.OutputSequence = 262244
	if got := evidence.failureBoundary(); got != "composition-throughput" {
		t.Fatalf("missing throughput boundary = %q", got)
	}
	evidence.ElapsedMS = 50
	evidence.OutputBytes = 262244
	evidence.ThroughputMBs = 5.24488
	if got := evidence.failureBoundary(); got != "" {
		t.Fatalf("complete boundary = %q", got)
	}
	evidence.MarkerObserved = true
	evidence.PTY.OutputSequence = 262247
	evidence.Recovery.OutputSequence = 262246
	evidence.Rendered.OutputSequence = 262245
	if got := evidence.failureBoundary(); got != "" {
		t.Fatalf("post-marker monotonic boundary = %q", got)
	}
	evidence.Rendered.OutputSequence = 262248
	if got := evidence.failureBoundary(); got != "renderer-output" {
		t.Fatalf("renderer-ahead boundary = %q", got)
	}
}
