package terminaltests

import (
	"os"
	"strings"
	"testing"
)

func TestNativePlatformWorkflowsRunTheSameInstalledScenarios(t *testing.T) {
	checks := map[string][]string{
		".github/workflows/darwin-system.yml": {
			"runs-on: ${{ inputs.runner }}", "artifact:", "architecture:", "target:", "variant:",
			"SOKSAK_TEST_PLATFORM: darwin", "SOKSAK_TEST_TARGET: ${{ inputs.target }}", "name: ${{ inputs.artifact }}",
			"chmod +x core/sok core/soksak.app/Contents/MacOS/soksak", "test \"$(uname -m)\" = \"${{ inputs.architecture }}\"",
			"lipo -archs", "SOKSAK_TEST_RUNTIME: /tmp/soksak-runtime", "make -C tests system-commands",
		},
		".github/workflows/linux-system.yml": {
			"runs-on: ${{ inputs.runner }}", "architecture:", "target:", "SOKSAK_TEST_PLATFORM: linux", "SOKSAK_TEST_TARGET: ${{ inputs.target }}",
			"core-linux-${{ inputs.architecture }}-artifact", "chmod +x core/sok core/soksak", "test \"$(go env GOARCH)\"", "make -C tests system-commands", "xvfb", "gnome-keyring",
		},
	}
	for path, required := range checks {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(body)
		for _, token := range append(required, "workflow_call:", "tests_ref:", "go-version-file: tests/go.mod", "system-commands", "system-restore", "if-no-files-found: error") {
			if !strings.Contains(text, token) {
				t.Errorf("%s omits %q", path, token)
			}
		}
	}
	runner, err := os.ReadFile("scripts/run-linux-system.sh")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"GTK_A11Y=none", "NO_AT_BRIDGE=1"} {
		if !strings.Contains(string(runner), required) {
			t.Errorf("Linux runner does not disable the headless accessibility bridge through %s", required)
		}
	}
}
