package system

import (
	"encoding/base64"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestTerminalShellCommandsUseThePlatformShell(t *testing.T) {
	windowsMarker, err := terminalPrintCommand("windows", "MARKER")
	if err != nil {
		t.Fatal(err)
	}
	if windowsMarker != "echo MARKER" || strings.Contains(windowsMarker, "printf") {
		t.Fatalf("Windows marker command = %q", windowsMarker)
	}
	windowsOutput, err := terminalHighOutputCommand("windows", "TAIL")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(windowsOutput, "powershell.exe -NoLogo -NoProfile -NonInteractive -EncodedCommand ") {
		t.Fatalf("Windows output command = %q", windowsOutput)
	}
	script := decodePowerShellForTest(t, windowsOutput[strings.LastIndex(windowsOutput, " ")+1:])
	for _, required := range []string{"'X' * 262144", "WriteLine('TAIL')"} {
		if !strings.Contains(script, required) {
			t.Errorf("Windows output script omits %q: %s", required, script)
		}
	}
	if strings.Contains(windowsOutput, "yes X") || strings.Contains(windowsOutput, "head -c") {
		t.Fatalf("Windows output command uses POSIX utilities: %s", windowsOutput)
	}

	unixOutput, err := terminalHighOutputCommand("linux", "TAIL")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(unixOutput, "head -c 262144") || !strings.Contains(unixOutput, "; printf") {
		t.Fatalf("Linux output command = %q", unixOutput)
	}
	if _, err := terminalHighOutputCommand("unknown", "TAIL"); err == nil {
		t.Fatal("unsupported platform was accepted")
	}
}

func decodePowerShellForTest(t *testing.T, encoded string) string {
	t.Helper()
	body, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatal(err)
	}
	values := make([]uint16, len(body)/2)
	for index := range values {
		values[index] = uint16(body[index*2]) | uint16(body[index*2+1])<<8
	}
	return string(utf16.Decode(values))
}
