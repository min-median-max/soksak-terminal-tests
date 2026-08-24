package fleet

import (
	"os"
	"strings"
	"testing"
)

func TestDarwinWorkflowRunsTheAddressedCoreArtifactOnItsNativeRunner(t *testing.T) {
	body, err := os.ReadFile("../.github/workflows/darwin-system.yml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, required := range []string{
		"runner:",
		"artifact:",
		"architecture:",
		"target:",
		"variant:",
		"runs-on: ${{ inputs.runner }}",
		"name: ${{ inputs.artifact }}",
		"test \"$(uname -m)\" = \"${{ inputs.architecture }}\"",
		"SOKSAK_TEST_TARGET: ${{ inputs.target }}",
		"lipo -archs",
		"darwin-${{ inputs.variant }}-${{ inputs.architecture }}-terminal-system-evidence",
	} {
		if !strings.Contains(text, required) {
			t.Errorf("Darwin system workflow is missing %q", required)
		}
	}
	for _, forbidden := range []string{
		"runs-on: macos-15\n",
		"name: core-darwin-artifact\n",
		"SOKSAK_TEST_TARGET: aarch64-apple-darwin",
	} {
		if strings.Contains(text, forbidden) {
			t.Errorf("Darwin system workflow retains fixed composition %q", forbidden)
		}
	}
}
