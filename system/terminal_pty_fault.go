package system

import (
	"fmt"

	"github.com/min-median-max/soksak-terminal-tests/fleet"
)

type ptyFaultSession struct {
	Session  uint64
	ShellPID uint32
}

func VerifyPtyFaultRecovery(profile fleet.Profile, cli CLI, views []TerminalResult) error {
	beforeStatus, err := ptyStatus(cli)
	if err != nil {
		return err
	}
	before, err := ptyFaultSessions(beforeStatus)
	if err != nil {
		return err
	}
	oldDaemon, err := runningSidecarPID(cli, "soksak-sidecar-pty")
	if err != nil {
		return err
	}
	if _, err := cli.Call("sidecar_stop", map[string]any{"name": "soksak-sidecar-pty"}); err != nil {
		return err
	}

	for _, view := range views {
		if _, err := cli.Call("tab.activate", map[string]any{"tab": view.View}); err != nil {
			return err
		}
		if _, err := cli.Call("ui.layout.wait-settled", map[string]any{"timeoutMs": 8000}); err != nil {
			return err
		}
		live, err := terminal(cli, view.Plugin, "wait", view.View, map[string]any{
			"phase": "live", "timeoutMs": 30000,
		})
		if err != nil {
			return fmt.Errorf("%s did not recover from the PTY fault: %w", view.Plugin, err)
		}
		if live["fidelity"] != "complete" {
			return fmt.Errorf("%s recovered with incomplete fidelity: %+v", view.Plugin, live)
		}
		marker := "SOKSAK_PTY_RESTART_" + view.Plugin
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
			return fmt.Errorf("%s did not render post-restart output: %w", view.Plugin, err)
		}
		if err := captureTerminal(cli, view.Plugin+"-pty-restart"); err != nil {
			return err
		}
	}

	afterStatus, err := ptyStatus(cli)
	if err != nil {
		return err
	}
	after, err := ptyFaultSessions(afterStatus)
	if err != nil {
		return err
	}
	if len(after) != len(before) || len(after) != len(views) {
		return fmt.Errorf("PTY session count changed across restart: %d -> %d, views=%d", len(before), len(after), len(views))
	}
	for _, view := range views {
		old, oldOK := before[view.Pane]
		fresh, freshOK := after[view.Pane]
		if !oldOK || !freshOK {
			return fmt.Errorf("PTY pane ownership changed for %s: before=%v after=%v", view.Pane, oldOK, freshOK)
		}
		if old.ShellPID == fresh.ShellPID {
			return fmt.Errorf("PTY pane %s reused ended shell PID %d", view.Pane, old.ShellPID)
		}
		gone, err := processGone(old.ShellPID)
		if err != nil || !gone {
			return fmt.Errorf("old PTY shell %d remains after recovery: gone=%v err=%v", old.ShellPID, gone, err)
		}
	}
	newDaemon, err := runningSidecarPID(cli, "soksak-sidecar-pty")
	if err != nil {
		return err
	}
	if newDaemon == oldDaemon {
		return fmt.Errorf("PTY sidecar PID did not change: %d", oldDaemon)
	}
	gone, err := processGone(oldDaemon)
	if err != nil || !gone {
		return fmt.Errorf("old PTY sidecar %d remains after recovery: gone=%v err=%v", oldDaemon, gone, err)
	}
	return nil
}

func ptyFaultSessions(status map[string]any) (map[string]ptyFaultSession, error) {
	values, _ := status["sessions"].([]any)
	result := make(map[string]ptyFaultSession, len(values))
	for _, value := range values {
		session, _ := value.(map[string]any)
		pane, _ := session["paneId"].(string)
		id, _ := session["session"].(float64)
		pid, _ := session["shellPid"].(float64)
		if pane == "" || id < 1 || pid < 1 {
			return nil, fmt.Errorf("PTY status contains an invalid session: %+v", session)
		}
		if _, duplicate := result[pane]; duplicate {
			return nil, fmt.Errorf("PTY status contains duplicate pane %s", pane)
		}
		result[pane] = ptyFaultSession{Session: uint64(id), ShellPID: uint32(pid)}
	}
	return result, nil
}

func runningSidecarPID(cli CLI, name string) (uint32, error) {
	status, err := cli.Call("sidecar_status", map[string]any{})
	if err != nil {
		return 0, err
	}
	open, _ := status["open"].([]any)
	for _, value := range open {
		entry, _ := value.(map[string]any)
		if entry["name"] != name {
			continue
		}
		pid, _ := entry["pid"].(float64)
		if pid >= 1 {
			return uint32(pid), nil
		}
	}
	return 0, fmt.Errorf("sidecar %s has no running PID: %+v", name, status)
}
