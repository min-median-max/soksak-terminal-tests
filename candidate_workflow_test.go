package terminaltests

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestDarwinCandidateWorkflowUsesTheExactOwnerWorkflowsAndNativeGate(t *testing.T) {
	body, err := os.ReadFile("candidate/terminal-native-darwin-arm64.json")
	if err != nil {
		t.Fatal(err)
	}
	var plan candidateSourcePlan
	if err := json.Unmarshal(body, &plan); err != nil {
		t.Fatal(err)
	}
	workflowBytes, err := os.ReadFile(".github/workflows/darwin-candidate-native-input.yml")
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(workflowBytes)
	for _, required := range []string{
		"workflow_call:", "tests_repository: { required: true, type: string }", "tests_ref:", "core_artifact_name:", "core_source_commit:",
		"soksak-ai/soksak-spec/.github/workflows/candidate.yml@" + plan.Spec.SourceCommit,
		"soksak-ai/soksak-spec/.github/workflows/node-candidate.yml@" + plan.Spec.SourceCommit,
		"soksak-ai/soksak-spec/.github/workflows/sidecar-candidate.yml@" + plan.Spec.SourceCommit,
		"soksak-ai/soksak-contract-plugin-terminal/.github/workflows/candidate.yml@" + plan.Contract.SourceCommit,
		"actions/download-artifact@3e5f45b2cfb9172054b4087a40e8e0b5a5461e7c",
		"make -C tests compose-candidate-plan", "system-native-input", "candidate-actions-artifacts.json",
		"actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a",
		"if-no-files-found: error",
	} {
		if !strings.Contains(workflow, required) {
			t.Errorf("candidate workflow omits %q", required)
		}
	}
	for _, forbidden := range []string{
		"contents: write", "create-github-app-token", "publish-canonical-release", "gh release create",
		"SOKSAK_PRESENTATION=capture-only",
	} {
		if strings.Contains(workflow, forbidden) {
			t.Errorf("candidate workflow contains forbidden %q", forbidden)
		}
	}
}
