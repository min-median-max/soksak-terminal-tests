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
