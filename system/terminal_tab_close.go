package system

import "fmt"

func VerifyTerminalTabCloseReaps(cli CLI, view TerminalResult) error {
	beforeStatus, err := ptyStatus(cli)
	if err != nil {
		return err
	}
	before, err := ptyFaultSessions(beforeStatus)
	if err != nil {
		return err
	}
	owned, ok := before[view.Pane]
	if !ok {
		return fmt.Errorf("PTY has no session for closing pane %s", view.Pane)
	}
	if _, err := cli.Call("tab.close", map[string]any{"tab": view.View}); err != nil {
		return err
	}
	afterStatus, err := ptyStatus(cli)
	if err != nil {
		return err
	}
	after, err := ptyFaultSessions(afterStatus)
	if err != nil {
		return err
	}
	if _, remains := after[view.Pane]; remains {
		return fmt.Errorf("closed tab left PTY pane %s alive", view.Pane)
	}
	gone, err := processGone(owned.ShellPID)
	if err != nil || !gone {
		return fmt.Errorf("closed tab left shell %d alive: gone=%v err=%v", owned.ShellPID, gone, err)
	}
	return nil
}
