package system

import (
	"fmt"
	"strings"
)

type RestoreView struct {
	TerminalResult
	ShellPID uint32
	Marker   string
}

func VerifyWarmAndArchivedRestore(lifecycle *Lifecycle, views []TerminalResult) error {
	cli := lifecycle.Client()
	restore := make([]RestoreView, 0, len(views))
	for index, view := range views {
		status, err := terminal(cli, view.Plugin, "status", view.View, nil)
		if err != nil {
			return err
		}
		pid, err := shellPID(status, view.View)
		if err != nil {
			return fmt.Errorf("%s: %w", view.Plugin, err)
		}
		marker := fmt.Sprintf("SOKSAK_DETACHED_%d", index)
		scheduled := fmt.Sprintf("SOKSAK_SCHEDULED_%d", index)
		if _, err := terminal(cli, view.Plugin, "send", view.View, map[string]any{
			"data": detachedMarkerCommand(index, scheduled),
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
			return err
		}
		if warm["recoveryOutcome"] != "continued" || warm["fidelity"] != "complete" {
			return fmt.Errorf("%s warm restore is incomplete: %+v", view.Plugin, warm)
		}
		status, err := terminal(cli, view.Plugin, "status", view.View, nil)
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
			return err
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

func detachedMarkerCommand(index int, scheduled string) string {
	return fmt.Sprintf("prefix=SOKSAK_DETACHED_; (sleep 10; printf '%%s\\n' \"${prefix}%d\") & printf '%%s\\n' %s\r", index, scheduled)
}

func countExactLine(text, wanted string) int {
	count := 0
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if strings.TrimSpace(line) == wanted {
			count++
		}
	}
	return count
}

func closePaneSession(cli CLI, pane string) error {
	held, err := sidecarRequest(cli, "soksak-sidecar-pty", "pane", "pty.pane", map[string]any{
		"request": map[string]any{"paneId": pane},
	})
	if err != nil {
		return err
	}
	data, _ := held["data"].(map[string]any)
	opened, _ := data["opened"].(map[string]any)
	session, _ := opened["session"].(float64)
	if session < 1 {
		return fmt.Errorf("PTY has no session for pane %s: %+v", pane, held)
	}
	_, err = sidecarRequest(cli, "soksak-sidecar-pty", "close", "pty.close", map[string]any{
		"request": map[string]any{"session": uint64(session)},
	})
	return err
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
	if response["ok"] != true {
		return nil, fmt.Errorf("%s refused %s: %+v", name, command, response)
	}
	result, _ := response["result"].(map[string]any)
	return result, nil
}

func shellPID(status map[string]any, pane string) (uint32, error) {
	source, _ := status["source"].(map[string]any)
	pty, _ := source["pty"].(map[string]any)
	sessions, _ := pty["sessions"].([]any)
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
