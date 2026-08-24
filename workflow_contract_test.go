package terminaltests

import (
	"os"
	"strings"
	"testing"
)

func TestWindowsWorkflowUsesRepositoryRunners(t *testing.T) {
	body, err := os.ReadFile(".github/workflows/windows-system.yml")
	if err != nil {
		t.Fatal(err)
	}
	source := string(body)
	if !strings.Contains(source, "make -C tests fleet TARGET=x86_64-pc-windows-msvc") {
		t.Fatal("Windows workflow does not use the fleet release verifier")
	}
	for _, obsolete := range []string{"install-windows-fleet.ps1", "prepare-development.exe", "development-input.json"} {
		if strings.Contains(source, obsolete) {
			t.Errorf("Windows workflow duplicates removed path %s", obsolete)
		}
	}
}

func TestDockerPreflightUsesTheSameFleetCommand(t *testing.T) {
	body, err := os.ReadFile("scripts/windows-preflight.sh")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "make fleet TARGET=x86_64-pc-windows-msvc") {
		t.Fatal("Docker preflight does not use the fleet release verifier")
	}
}
