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

func TestSidecarResponseUsesTheContractPayload(t *testing.T) {
	response := map[string]any{
		"ok": true,
		"result": map[string]any{
			"code": "OK",
			"data": map[string]any{"sessions": []any{map[string]any{"shellPid": float64(42)}}},
		},
	}
	payload, err := sidecarPayload("pty", "pty.status", response)
	if err != nil || payload["sessions"] == nil || payload["code"] != nil {
		t.Fatalf("payload=%+v err=%v", payload, err)
	}
}
