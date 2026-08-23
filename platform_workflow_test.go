package terminaltests

import (
	"os"
	"strings"
	"testing"
)

func TestNativePlatformWorkflowsRunTheSameInstalledScenarios(t *testing.T) {
	checks := map[string][]string{
		".github/workflows/darwin-system.yml": {
			"runs-on: macos-15", "SOKSAK_TEST_PLATFORM: darwin", "SOKSAK_TEST_TARGET: aarch64-apple-darwin",
			"core-darwin-artifact", "runner.temp", "scripts/run-darwin-system.sh",
		},
		".github/workflows/linux-system.yml": {
			"runs-on: ubuntu-24.04", "SOKSAK_TEST_PLATFORM: linux", "SOKSAK_TEST_TARGET: x86_64-unknown-linux-gnu",
			"core-linux-amd64-artifact", "scripts/run-linux-system.sh", "xvfb", "gnome-keyring",
		},
	}
	for path, required := range checks {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(body)
		for _, token := range append(required, "workflow_call:", "tests_ref:", "go-version-file: tests/go.mod", "TestInstalledTerminalCommands", "TestInstalledTerminalWarmAndArchivedRestore", "if-no-files-found: error") {
			if !strings.Contains(text, token) {
				t.Errorf("%s omits %q", path, token)
			}
		}
	}
}
