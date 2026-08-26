package system

import "testing"

func TestTerminalTypingKeysAppendsEnterAndRejectsNonKeyboardText(t *testing.T) {
	keys, err := terminalTypingKeys("printf %s value")
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != len("printf %s value")+1 || keys[len(keys)-1] != "Enter" {
		t.Fatalf("unexpected typing sequence: %v", keys)
	}
	if _, err := terminalTypingKeys("한"); err == nil {
		t.Fatal("non-ASCII text bypassed the composition input path")
	}
	if _, err := terminalTypingKeys("a\rb"); err == nil {
		t.Fatal("embedded control input bypassed the Enter key event")
	}
}
