package system

import (
	"fmt"
	"strings"

	"github.com/min-median-max/soksak-terminal-tests/fleet"
)

type RestoreView struct {
	TerminalResult
	ShellPID uint32
	Marker   string
}

func VerifyWarmAndArchivedRestore(profile fleet.Profile, lifecycle *Lifecycle, views []TerminalResult) error {
	cli := lifecycle.Client()
	restore := make([]RestoreView, 0, len(views))
	for index, view := range views {
		status, err := ptyStatus(cli)
		if err != nil {
			return err
		}
		pid, err := shellPID(status, view.View)
		if err != nil {
			return fmt.Errorf("%s: %w", view.Plugin, err)
		}
		marker := fmt.Sprintf("SOKSAK_DETACHED_%d", index)
		scheduled := fmt.Sprintf("SOKSAK_SCHEDULED_%d", index)
		detached, err := detachedMarkerCommand(profile.Platform, marker, scheduled)
		if err != nil {
			return err
		}
		if _, err := terminal(cli, view.Plugin, "send", view.View, map[string]any{
			"data": detached,
		}); err != nil {
			return err
		}
		if _, err := terminal(cli, view.Plugin, "wait", view.View, map[string]any{"phase": "live", "contains": scheduled, "timeoutMs": 8000}); err != nil {
			return err
		}
		restore = append(restore, RestoreView{TerminalResult: view, ShellPID: pid, Marker: marker})
	}
	if err := lifecycle.Shutdown(); err != nil {
		return err
	}
	if err := lifecycle.Start(); err != nil {
		return err
	}
	if err := lifecycle.AwaitWindow(); err != nil {
		return err
	}
	cli = lifecycle.Client()
	for _, view := range restore {
		if _, err := cli.Call("tab.mount.wait", map[string]any{"tab": view.View, "timeoutMs": 20000}); err != nil {
			return err
		}
		warm, err := terminal(cli, view.Plugin, "wait", view.View, map[string]any{
			"phase": "live", "contains": view.Marker, "timeoutMs": 20000,
		})
		if err != nil {
			return terminalRestoreDiagnostic(cli, view, "warm restore wait", err)
		}
		if warm["recoveryOutcome"] != "continued" || warm["fidelity"] != "complete" {
			return fmt.Errorf("%s warm restore is incomplete: %+v", view.Plugin, warm)
		}
		status, err := ptyStatus(cli)
		if err != nil {
			return err
		}
		pid, err := shellPID(status, view.View)
		if err != nil || pid != view.ShellPID {
			return fmt.Errorf("%s shell PID changed: %d -> %d: %v", view.Plugin, view.ShellPID, pid, err)
		}
		read, err := terminal(cli, view.Plugin, "read", view.View, nil)
		if err != nil || countExactLine(fmt.Sprint(read["text"]), view.Marker) != 1 {
			return fmt.Errorf("%s detached marker is not present exactly once: %v %+v", view.Plugin, err, read)
		}
		archived, err := terminal(cli, view.Plugin, "archive", view.View, nil)
		bytes, _ := archived["bytes"].(float64)
		if err != nil || archived["archived"] != true || bytes < 1 {
			return fmt.Errorf("%s archive failed: %v %+v", view.Plugin, err, archived)
		}
		if err := closePaneSession(cli, view.View); err != nil {
			return err
		}
	}
	markers := make([]string, 0, len(restore))
	for _, view := range restore {
		markers = append(markers, view.Marker)
	}
	if err := verifyEncryptedCheckpoints(lifecycle.config.Home, profile.RecoverySidecars, markers, len(restore)); err != nil {
		return err
	}
	if err := lifecycle.Shutdown(); err != nil {
		return err
	}
	if err := lifecycle.Start(); err != nil {
		return err
	}
	if err := lifecycle.AwaitWindow(); err != nil {
		return err
	}
	cli = lifecycle.Client()
	for _, view := range restore {
		if _, err := cli.Call("tab.mount.wait", map[string]any{"tab": view.View, "timeoutMs": 20000}); err != nil {
			return err
		}
		archived, err := terminal(cli, view.Plugin, "wait", view.View, map[string]any{
			"phase": "archived", "contains": view.Marker, "timeoutMs": 20000,
		})
		if err != nil {
			return terminalRestoreDiagnostic(cli, view, "archived restore wait", err)
		}
		if archived["recoveryOutcome"] != "archived" || archived["fidelity"] != "complete" {
			return fmt.Errorf("%s archived restore is incomplete: %+v", view.Plugin, archived)
		}
		sent, err := terminal(cli, view.Plugin, "send", view.View, map[string]any{"data": "ARCHIVED_INPUT_MUST_FAIL"})
		if err != nil || sent["sent"] != false {
			return fmt.Errorf("%s accepted archived input: %v %+v", view.Plugin, err, sent)
		}
	}
	return nil
}

func terminalRestoreDiagnostic(cli CLI, view RestoreView, stage string, cause error) error {
	status, statusErr := terminal(cli, view.Plugin, "status", view.View, nil)
	read, readErr := terminal(cli, view.Plugin, "read", view.View, nil)
	return fmt.Errorf("%s %s failed: %w; status=%+v statusErr=%v read=%+v readErr=%v", view.Plugin, stage, cause, status, statusErr, read, readErr)
}

func countExactLine(text, wanted string) int {
	return strings.Count(text, wanted)
}

func closePaneSession(cli CLI, pane string) error {
	held, err := sidecarRequest(cli, "soksak-sidecar-pty", "pane", "pty.pane", map[string]any{
		"request": map[string]any{"paneId": pane},
	})
	if err != nil {
		return err
	}
	session, err := paneSessionID(held, pane)
	if err != nil {
		return err
	}
	_, err = sidecarRequest(cli, "soksak-sidecar-pty", "close", "pty.close", map[string]any{
		"request": map[string]any{"session": uint64(session)},
	})
	return err
}

func paneSessionID(held map[string]any, pane string) (uint64, error) {
	if held["held"] != true {
		return 0, fmt.Errorf("PTY has no session for pane %s: %+v", pane, held)
	}
	opened, _ := held["opened"].(map[string]any)
	session, _ := opened["session"].(float64)
	if session < 1 || session != float64(uint64(session)) {
		return 0, fmt.Errorf("PTY has no session for pane %s: %+v", pane, held)
	}
	return uint64(session), nil
}

func sidecarRequest(cli CLI, name, id, command string, args map[string]any) (map[string]any, error) {
	data, err := cli.Call("sidecar.request", map[string]any{
		"name":    name,
		"request": map[string]any{"id": id, "command": command, "args": args},
	})
	if err != nil {
		return nil, err
	}
	response, _ := data["response"].(map[string]any)
	return sidecarPayload(name, command, response)
}

func sidecarPayload(name, command string, response map[string]any) (map[string]any, error) {
	if response["ok"] != true {
		return nil, fmt.Errorf("%s refused %s: %+v", name, command, response)
	}
	result, _ := response["result"].(map[string]any)
	data, _ := result["data"].(map[string]any)
	if data == nil {
		return nil, fmt.Errorf("%s returned no contract payload for %s: %+v", name, command, response)
	}
	return data, nil
}

func ptyStatus(cli CLI) (map[string]any, error) {
	return sidecarRequest(cli, "soksak-sidecar-pty", "status", "pty.status", map[string]any{})
}

func shellPID(status map[string]any, pane string) (uint32, error) {
	sessions, _ := status["sessions"].([]any)
	for _, value := range sessions {
		session, _ := value.(map[string]any)
		if session["paneId"] != pane {
			continue
		}
		pid, _ := session["shellPid"].(float64)
		if pid >= 1 {
			return uint32(pid), nil
		}
	}
	return 0, fmt.Errorf("no shell PID for pane %s", pane)
}
