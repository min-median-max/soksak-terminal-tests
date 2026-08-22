package terminaltests

import (
	"os"
	"strings"
	"testing"
)

func TestWindowsWorkflowConsumesOwnerArtifacts(t *testing.T) {
	b, e := os.ReadFile(".github/workflows/windows-system.yml")
	if e != nil {
		t.Fatal(e)
	}
	s := string(b)
	for _, v := range []string{
		"actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1",
		"actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e",
		"actions/download-artifact@3e5f45b2cfb9172054b4087a40e8e0b5a5461e7c",
		"actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a",
	} {
		if !strings.Contains(s, v) {
			t.Errorf("Windows workflow does not pin Node 24 action %s", v)
		}
	}
	for _, v := range []string{"workflow_call:", "tests_ref:", "ref: ${{ inputs.tests_ref }}", "cache-dependency-path: tests/go.sum", "windows-2025", "SOKSAK_TEST_PLATFORM: windows", "TestInstalledTerminalCommands", "TestInstalledTerminalWarmAndArchivedRestore"} {
		if !strings.Contains(s, v) {
			t.Errorf("missing %s", v)
		}
	}
	if !strings.Contains(s, "New-Item -ItemType Directory -Force stage/evidence") {
		t.Error("Windows workflow does not create the evidence directory before fleet installation")
	}
	for _, v := range []string{"github.workflow_sha", "soksak-" + "plugins", "soksak-" + "sidecars", "git clone", "cargo build"} {
		if strings.Contains(s, v) {
			t.Errorf("executes owner source: %s", v)
		}
	}
	installer, err := os.ReadFile("scripts/install-windows-fleet.ps1")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(installer), "gh run download") {
		t.Fatal("Windows fleet installs a workflow artifact instead of an immutable release")
	}
	if !strings.Contains(string(installer), `../release/windows-fleet.json`) {
		t.Error("Windows installer does not read the verified fleet declaration")
	}
}
