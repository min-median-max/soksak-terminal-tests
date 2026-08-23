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
			"core-darwin-artifact", "chmod +x core/sok core/soksak.app/Contents/MacOS/soksak", "runner.temp", "scripts/run-darwin-system.sh",
		},
		".github/workflows/linux-system.yml": {
			"runs-on: ${{ inputs.runner }}", "architecture:", "target:", "SOKSAK_TEST_PLATFORM: linux", "SOKSAK_TEST_TARGET: ${{ inputs.target }}",
			"core-linux-${{ inputs.architecture }}-artifact", "chmod +x core/sok core/soksak", "test \"$(go env GOARCH)\"", "scripts/run-linux-system.sh", "xvfb", "gnome-keyring",
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
