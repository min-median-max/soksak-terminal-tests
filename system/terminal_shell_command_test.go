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
	const highOutputMarker = "SOKSAK_HIGH_OUTPUT_TAIL_7"
	windowsOutput, err := terminalHighOutputCommand("windows", highOutputMarker)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(windowsOutput, "powershell.exe -NoLogo -NoProfile -NonInteractive -EncodedCommand ") {
		t.Fatalf("Windows output command = %q", windowsOutput)
	}
	script := decodePowerShellForTest(t, windowsOutput[strings.LastIndex(windowsOutput, " ")+1:])
	for _, required := range []string{"'X' * 262144", "$prefix", "'7'"} {
		if !strings.Contains(script, required) {
			t.Errorf("Windows output script omits %q: %s", required, script)
		}
	}
	if strings.Contains(windowsOutput, highOutputMarker) || strings.Contains(script, highOutputMarker) {
		t.Fatalf("Windows high-output command exposes the awaited marker: %s", script)
	}
	if strings.Contains(windowsOutput, "yes X") || strings.Contains(windowsOutput, "head -c") {
		t.Fatalf("Windows output command uses POSIX utilities: %s", windowsOutput)
	}

	unixOutput, err := terminalHighOutputCommand("linux", highOutputMarker)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(unixOutput, "head -c 262144 /dev/zero | tr '\\0' X") || !strings.Contains(unixOutput, "; printf") {
		t.Fatalf("Linux output command = %q", unixOutput)
	}
	if strings.Contains(unixOutput, "yes X") {
		t.Fatalf("Linux output command creates one parser event per line: %s", unixOutput)
	}
	if strings.Contains(unixOutput, highOutputMarker) {
		t.Fatalf("Linux high-output command exposes the awaited marker: %s", unixOutput)
	}
	if _, err := terminalHighOutputCommand("unknown", "TAIL"); err == nil {
		t.Fatal("unsupported platform was accepted")
	}
}

func TestDetachedMarkerCommandsUseThePlatformShell(t *testing.T) {
	windows, err := detachedMarkerCommand("windows", "SOKSAK_DETACHED_7", "SOKSAK_SCHEDULED_7")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(windows, `start "" /b powershell.exe -NoLogo -NoProfile -NonInteractive -EncodedCommand `) {
		t.Fatalf("Windows detached command = %q", windows)
	}
	background := strings.Split(windows, " & echo ")[0]
	fields := strings.Fields(background)
	script := decodePowerShellForTest(t, fields[len(fields)-1])
	for _, required := range []string{
		"Start-Sleep -Seconds 10", "[Console]::Out.WriteLine()",
		"[Console]::Out.WriteLine('SOKSAK_DETACHED_7')",
	} {
		if !strings.Contains(script, required) {
			t.Errorf("Windows detached script omits %q: %s", required, script)
		}
	}
	if !strings.HasSuffix(windows, " & echo SOKSAK_SCHEDULED_7\r") || strings.Contains(windows, "sleep 10") {
		t.Fatalf("Windows detached command mixes shell syntax: %s", windows)
	}
	unix, err := detachedMarkerCommand("linux", "SOKSAK_DETACHED_7", "SOKSAK_SCHEDULED_7")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(unix, "(sleep 10; printf '\n%s\n'") || strings.Contains(unix, "SOKSAK_DETACHED_7") {
		t.Fatalf("Linux detached command = %q", unix)
	}
}

func TestWindowsDelayedMarkerSeparatesFromThePrompt(t *testing.T) {
	const prompt = `D:\a\soksak\tests\system>`
	const marker = "SOKSAK_DETACHED_7"
	withoutLeadingNewline := prompt + marker + "\r\n"
	if countExactLine(withoutLeadingNewline, marker) != 0 {
		t.Fatal("a marker attached to the prompt was accepted")
	}
	withLeadingNewline := prompt + "\r\n" + marker + "\r\n"
	if countExactLine(withLeadingNewline, marker) != 1 {
		t.Fatalf("a separated delayed marker was not accepted: %q", withLeadingNewline)
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
