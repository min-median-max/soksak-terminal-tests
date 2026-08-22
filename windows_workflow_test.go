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
	for _, v := range []string{"workflow_call:", "tests_ref:", "ref: ${{ inputs.tests_ref }}", "windows-2025", "SOKSAK_TEST_PLATFORM: windows", "TestInstalledTerminalCommands", "TestInstalledTerminalWarmAndArchivedRestore"} {
		if !strings.Contains(s, v) {
			t.Errorf("missing %s", v)
		}
	}
	for _, v := range []string{"github.workflow_sha", "soksak-" + "plugins", "soksak-" + "sidecars", "git clone", "cargo build"} {
		if strings.Contains(s, v) {
			t.Errorf("executes owner source: %s", v)
		}
	}
}
