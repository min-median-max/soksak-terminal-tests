package system

import "testing"

func TestShellPIDReadsThePTYStatusContract(t *testing.T) {
	status := map[string]any{
		"sessions": []any{
			map[string]any{"paneId": "pane-b", "shellPid": float64(41)},
			map[string]any{"paneId": "pane-a", "shellPid": float64(42)},
		},
	}
	pid, err := shellPID(status, "pane-a")
	if err != nil || pid != 42 {
		t.Fatalf("pid=%d err=%v", pid, err)
	}
}
