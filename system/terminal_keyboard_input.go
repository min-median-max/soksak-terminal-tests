package system

import "fmt"

func terminalTypingKeys(text string) ([]string, error) {
	keys := make([]string, 0, len(text)+1)
	for _, value := range text {
		if value < 0x20 || value > 0x7e {
			return nil, fmt.Errorf("terminal command contains non-keyboard rune %U", value)
		}
		keys = append(keys, string(value))
	}
	return append(keys, "Enter"), nil
}

func typeTerminalCommand(cli CLI, plugin, view, command string) error {
	keys, err := terminalTypingKeys(command)
	if err != nil {
		return err
	}
	tree, err := cli.Call("ui.tree", map[string]any{})
	if err != nil {
		return err
	}
	address, err := terminalNodeAddress(tree, plugin, view, "terminal-input")
	if err != nil {
		return err
	}
	if _, err := cli.Call("ui.input.click", map[string]any{"address": address}); err != nil {
		return err
	}
	for _, key := range keys {
		if _, err := cli.Call("ui.input.key", map[string]any{"address": address, "key": key}); err != nil {
			return err
		}
	}
	return nil
}
