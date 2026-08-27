package system

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
		pid, err := shellPID(status, view.Pane)
		if err != nil {
			return fmt.Errorf("%s: %w", view.Plugin, err)
		}
		marker := fmt.Sprintf("SOKSAK_DETACHED_%d", index)
		scheduled := fmt.Sprintf("SOKSAK_SCHEDULED_%d", index)
		detached, err := detachedMarkerCommand(profile.Platform, marker, scheduled)
		if err != nil {
			return err
		}
		if err := typeTerminalCommand(cli, view.Plugin, view.View, detached); err != nil {
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
		if err := activateRestoredTerminal(cli, view.View); err != nil {
			return err
		}
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
		pid, err := shellPID(status, view.Pane)
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
		if err := writeArchivedCheckpointEvidence(lifecycle.config.EvidenceDir, cli, view, "archive", true); err != nil {
			return err
		}
	}
	markers := make([]string, 0, len(restore))
	for _, view := range restore {
		markers = append(markers, view.Marker)
	}
	if err := verifyEncryptedCheckpoints(lifecycle.config.Home, profile.RecoverySidecarIDs(), markers, len(restore)); err != nil {
		return err
	}
	oldDaemon, err := runningSidecarPID(cli, "soksak-sidecar-pty")
	if err != nil {
		return err
	}
	if err := lifecycle.Shutdown(); err != nil {
		return err
	}
	if err := terminateProcess(oldDaemon); err != nil {
		return fmt.Errorf("terminate detached PTY sidecar %d: %w", oldDaemon, err)
	}
	if err := lifecycle.Start(); err != nil {
		return err
	}
	if err := lifecycle.AwaitWindow(); err != nil {
		return err
	}
	cli = lifecycle.Client()
	for index, view := range restore {
		if err := activateRestoredTerminal(cli, view.View); err != nil {
			return err
		}
		if _, err := cli.Call("tab.mount.wait", map[string]any{"tab": view.View, "timeoutMs": 20000}); err != nil {
			return err
		}
		archived, err := terminal(cli, view.Plugin, "wait", view.View, map[string]any{
			"phase": "live", "timeoutMs": 30000,
		})
		if err != nil {
			return terminalRestoreDiagnostic(cli, view, "archived restore wait", err)
		}
		if archived["recoveryOutcome"] != "archived" || archived["fidelity"] != "complete" {
			return fmt.Errorf("%s archived restore is incomplete: %+v", view.Plugin, archived)
		}
		status, err := ptyStatus(cli)
		if err != nil {
			return err
		}
		pid, err := shellPID(status, view.Pane)
		if err != nil || pid == view.ShellPID {
			return fmt.Errorf("%s did not replace archived shell PID %d: pid=%d err=%v", view.Plugin, view.ShellPID, pid, err)
		}
		marker := fmt.Sprintf("SOKSAK_ARCHIVED_RESTART_%d", index)
		command, err := terminalPrintCommand(profile.Platform, marker)
		if err != nil {
			return err
		}
		if err := typeTerminalCommand(cli, view.Plugin, view.View, command); err != nil {
			return err
		}
		if _, err := terminal(cli, view.Plugin, "wait", view.View, map[string]any{
			"phase": "live", "contains": marker, "timeoutMs": 12000,
		}); err != nil {
			return fmt.Errorf("%s did not accept input after archived recovery: %w", view.Plugin, err)
		}
		markerCount, err := countExactLineInTerminalHistory(cli, view.Plugin, view.View, view.Marker)
		if err != nil || markerCount != 1 {
			recoveryCount, recoveryErr := countExactLineInRecoveryHistory(lifecycle.config.EvidenceDir, cli, view, view.Marker)
			postArchiveErr := writeArchivedCheckpointEvidence(lifecycle.config.EvidenceDir, cli, view, "post-recovery-archive", false)
			return fmt.Errorf("%s discarded or duplicated archived screen after starting a shell: presenterCount=%d presenterErr=%v recoveryCount=%d recoveryErr=%v postArchiveErr=%v", view.Plugin, markerCount, err, recoveryCount, recoveryErr, postArchiveErr)
		}
		if err := captureTerminal(cli, view.Plugin+"-archived-restart"); err != nil {
			return err
		}
	}
	newDaemon, err := runningSidecarPID(cli, "soksak-sidecar-pty")
	if err != nil || newDaemon == oldDaemon {
		return fmt.Errorf("PTY sidecar did not restart after archive fault: %d -> %d: %v", oldDaemon, newDaemon, err)
	}
	if gone, err := processGone(oldDaemon); err != nil || !gone {
		return fmt.Errorf("old PTY sidecar %d remains after archived recovery: gone=%v err=%v", oldDaemon, gone, err)
	}
	return nil
}

func activateRestoredTerminal(cli CLI, view string) error {
	if _, err := cli.Call("tab.activate", map[string]any{"tab": view}); err != nil {
		return err
	}
	_, err := cli.Call("ui.layout.wait-settled", map[string]any{"timeoutMs": 8000})
	return err
}

func terminalRestoreDiagnostic(cli CLI, view RestoreView, stage string, cause error) error {
	status, statusErr := terminal(cli, view.Plugin, "status", view.View, nil)
	read, readErr := terminal(cli, view.Plugin, "read", view.View, nil)
	pty, ptyErr := ptyStatus(cli)
	sidecars, sidecarErr := cli.Call("sidecar_status", map[string]any{})
	health, healthErr := cli.Call("state.health", map[string]any{})
	return fmt.Errorf("%s %s failed: %w; status=%+v statusErr=%v read=%+v readErr=%v pty=%+v ptyErr=%v sidecars=%+v sidecarErr=%v health=%+v healthErr=%v", view.Plugin, stage, cause, status, statusErr, read, readErr, pty, ptyErr, sidecars, sidecarErr, health, healthErr)
}

func countExactLine(text, wanted string) int {
	count := 0
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if line == wanted {
			count++
		}
	}
	return count
}

func countExactLineInTerminalHistory(cli CLI, plugin, view, wanted string) (int, error) {
	positions := map[int]bool{}
	scroll, err := terminal(cli, plugin, "scroll", view, map[string]any{"edge": "bottom"})
	if err != nil {
		return 0, err
	}
	defer func() { _, _ = terminal(cli, plugin, "scroll", view, map[string]any{"edge": "bottom"}) }()
	for {
		offset, offsetOK := exactInt(scroll["offset"])
		historySize, historyOK := exactInt(scroll["historySize"])
		if !offsetOK || !historyOK || offset < 0 || historySize < offset {
			return 0, fmt.Errorf("invalid scroll state: %+v", scroll)
		}
		read, readErr := terminal(cli, plugin, "read", view, nil)
		if readErr != nil {
			return 0, readErr
		}
		rows := recordExactLinePositions(positions, fmt.Sprint(read["text"]), wanted, offset)
		if offset == historySize {
			return len(positions), nil
		}
		next := offset + rows
		if next > historySize {
			next = historySize
		}
		if next <= offset {
			return 0, fmt.Errorf("terminal history did not advance: offset=%d rows=%d history=%d", offset, rows, historySize)
		}
		scroll, err = terminal(cli, plugin, "scroll", view, map[string]any{"offset": next})
		if err != nil {
			return 0, err
		}
	}
}

func recordExactLinePositions(positions map[int]bool, text, wanted string, offset int) int {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	for row, line := range lines {
		if line == wanted {
			positions[offset+len(lines)-1-row] = true
		}
	}
	return len(lines)
}

func writeArchivedCheckpointEvidence(directory string, cli CLI, view RestoreView, suffix string, requireMarker bool) error {
	status, err := terminal(cli, view.Plugin, "status", view.View, nil)
	if err != nil {
		return err
	}
	engine, _ := status["engineId"].(string)
	if engine == "" {
		return fmt.Errorf("%s archive status has no engine: %+v", view.Plugin, status)
	}
	pty, err := ptyStatus(cli)
	if err != nil {
		return err
	}
	window, err := paneWindowLabel(pty, view.Pane)
	if err != nil {
		return err
	}
	provider := "soksak-sidecar-terminal-" + engine
	archived, err := sidecarRequest(cli, provider, "archived", "terminal.archived", map[string]any{
		"request": map[string]any{"window": window, "pane": view.Pane},
	})
	if err != nil {
		return err
	}
	text := archivedFrameText(archived["frame"])
	if requireMarker && countExactLine(text, view.Marker) != 1 {
		return fmt.Errorf("%s checkpoint frame does not contain marker exactly once: %q", view.Plugin, view.Marker)
	}
	body, err := json.MarshalIndent(map[string]any{
		"plugin": view.Plugin, "engine": engine, "window": window, "pane": view.Pane,
		"marker": view.Marker, "checkpoint": archived,
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(directory, view.Plugin+"-"+suffix+".json"), append(body, '\n'), 0o600)
}

func archivedFrameText(value any) string {
	frame, _ := value.(map[string]any)
	lines, _ := frame["lines"].([]any)
	rows := make([]string, 0, len(lines))
	for _, lineValue := range lines {
		line, _ := lineValue.(map[string]any)
		runs, _ := line["runs"].([]any)
		var row strings.Builder
		for _, runValue := range runs {
			run, _ := runValue.(map[string]any)
			row.WriteString(fmt.Sprint(run["text"]))
		}
		rows = append(rows, row.String())
	}
	return strings.Join(rows, "\n")
}

func paneWindowLabel(status map[string]any, pane string) (string, error) {
	sessions, _ := status["sessions"].([]any)
	for _, value := range sessions {
		session, _ := value.(map[string]any)
		if session["paneId"] == pane {
			window, _ := session["windowLabel"].(string)
			if window != "" {
				return window, nil
			}
		}
	}
	return "", fmt.Errorf("PTY has no window label for pane %s", pane)
}

func countExactLineInRecoveryHistory(directory string, cli CLI, view RestoreView, wanted string) (int, error) {
	status, err := terminal(cli, view.Plugin, "status", view.View, nil)
	if err != nil {
		return 0, err
	}
	engine, _ := status["engineId"].(string)
	pty, err := ptyStatus(cli)
	if err != nil {
		return 0, err
	}
	window, err := paneWindowLabel(pty, view.Pane)
	if err != nil {
		return 0, err
	}
	provider := "soksak-sidecar-terminal-" + engine
	positions := map[int]bool{}
	pages := []map[string]any{}
	offset := 0
	for {
		frame, requestErr := sidecarRequest(cli, provider, "frame", "terminal.frame", map[string]any{
			"request": map[string]any{
				"window": window, "pane": view.Pane, "subscriber": "restore-history", "offset": offset, "timeoutMs": 0,
			},
		})
		if requestErr != nil {
			return 0, requestErr
		}
		actualOffset, offsetOK := exactInt(frame["offset"])
		historySize, historyOK := exactInt(frame["historySize"])
		rows, rowsOK := exactInt(frame["rows"])
		if !offsetOK || !historyOK || !rowsOK || actualOffset < 0 || historySize < actualOffset || rows < 1 {
			return 0, fmt.Errorf("invalid recovery frame state: %+v", frame)
		}
		pages = append(pages, frame)
		lines, _ := frame["lines"].([]any)
		for _, lineValue := range lines {
			line, _ := lineValue.(map[string]any)
			y, yOK := exactInt(line["y"])
			if !yOK || y < 0 || y >= rows {
				continue
			}
			if countExactLine(archivedFrameText(map[string]any{"lines": []any{line}}), wanted) == 1 {
				positions[actualOffset+rows-1-y] = true
			}
		}
		if actualOffset == historySize {
			body, marshalErr := json.MarshalIndent(map[string]any{
				"plugin": view.Plugin, "engine": engine, "window": window, "pane": view.Pane,
				"marker": wanted, "positions": positions, "pages": pages,
			}, "", "  ")
			if marshalErr != nil {
				return 0, marshalErr
			}
			if writeErr := os.WriteFile(filepath.Join(directory, view.Plugin+"-recovery-history.json"), append(body, '\n'), 0o600); writeErr != nil {
				return 0, writeErr
			}
			return len(positions), nil
		}
		next := actualOffset + rows
		if next > historySize {
			next = historySize
		}
		if next <= actualOffset {
			return 0, fmt.Errorf("recovery history did not advance: offset=%d rows=%d history=%d", actualOffset, rows, historySize)
		}
		offset = next
	}
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
