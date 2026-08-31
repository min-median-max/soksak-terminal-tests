package system

import (
	"reflect"
	"testing"
)

func TestHighOutputWaitsForTheRestoredWideTerminalSize(t *testing.T) {
	want := map[string]any{"phase": "live", "cols": float64(91), "timeoutMs": 8000}
	if got := wideResizeWait(91); !reflect.DeepEqual(got, want) {
		t.Fatalf("wide resize barrier = %#v", got)
	}
}

func TestResizeEvidenceIdentifiesRestoreFailure(t *testing.T) {
	requested := &terminalSize{Cols: 56, Rows: 30}
	pty := &terminalSequencedSize{terminalSize: *requested, EventSequence: 2, OutputSequence: 3}
	recovery := &terminalRecoverySize{terminalSequencedSize: *pty, Gaps: 0}
	rendered := &terminalRenderedSize{terminalSize: *requested, OutputSequence: 3}
	evidence := terminalResizeEvidence{
		Wide: terminalResizeSample{
			DOM:          terminalDOMSize{Width: 739},
			ReportedCols: 126,
		},
		Narrow: terminalResizeSample{
			DOM:          terminalDOMSize{Width: 440},
			ReportedCols: 56,
			Requested:    requested,
			PTY:          pty,
			Recovery:     recovery,
			Rendered:     rendered,
		},
		Restored: &terminalResizeSample{
			DOM:          terminalDOMSize{Width: 739},
			ReportedCols: 56,
		},
	}
	if got := evidence.failureBoundary(); got != "plugin-size-restore" {
		t.Fatalf("restore failure boundary = %q", got)
	}
}

func TestResizeEvidenceRequiresRestoreObservation(t *testing.T) {
	requested := &terminalSize{Cols: 56, Rows: 30}
	pty := &terminalSequencedSize{terminalSize: *requested, EventSequence: 2, OutputSequence: 3}
	recovery := &terminalRecoverySize{terminalSequencedSize: *pty, Gaps: 0}
	rendered := &terminalRenderedSize{terminalSize: *requested, OutputSequence: 3}
	evidence := terminalResizeEvidence{
		Wide: terminalResizeSample{DOM: terminalDOMSize{Width: 739}, ReportedCols: 94},
		Narrow: terminalResizeSample{
			DOM: terminalDOMSize{Width: 440}, ReportedCols: 56, Requested: requested,
			PTY: pty, Recovery: recovery, Rendered: rendered,
		},
	}
	if got := evidence.failureBoundary(); got != "restore-observation" {
		t.Fatalf("missing restore boundary = %q", got)
	}
}
