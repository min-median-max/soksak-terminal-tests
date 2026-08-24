package terminaltests

import (
	"os"
	"strings"
	"testing"
)

func TestMakeOwnsAcceptanceCommands(t *testing.T) {
	body, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatal(err)
	}
	source := string(body)
	for _, target := range []string{
		"preflight:", "prepare:", "verify:", "fleet:",
		"system-commands:", "system-restore:",
	} {
		if !strings.Contains(source, target) {
			t.Errorf("Makefile omits %s", target)
		}
	}
	if strings.Contains(source, "GO_VERSION :=") {
		t.Error("Makefile duplicates the Go version from go.mod")
	}
}

func TestNativeWorkflowsInvokeAcceptanceThroughMake(t *testing.T) {
	for _, path := range []string{
		".github/workflows/darwin-system.yml",
		".github/workflows/linux-system.yml",
		".github/workflows/windows-system.yml",
	} {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		source := string(body)
		for _, required := range []string{
			"make -C tests fleet TARGET=",
			"make -C tests system-commands TARGET=",
			"make -C tests system-restore TARGET=",
		} {
			if !strings.Contains(source, required) {
				t.Errorf("%s omits %s", path, required)
			}
		}
		for _, bypass := range []string{
			"go run -C tests ./cmd/verify-fleet",
			"scripts/run-darwin-system.sh -run",
			"scripts/run-linux-system.sh -run",
			"go test -C tests -count=1 -tags=system",
		} {
			if strings.Contains(source, bypass) {
				t.Errorf("%s bypasses Make through %s", path, bypass)
			}
		}
	}
}
