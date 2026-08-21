//go:build system

package system

import (
	"os"
	"testing"
)

func TestInstalledTerminalInventory(t *testing.T) {
	path := os.Getenv("SOKSAK_TEST_SETTINGS")
	if path == "" {
		t.Fatal("SOKSAK_TEST_SETTINGS must name the installed settings.json")
	}
	settings, err := ReadSettings(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateTerminalInventory(settings); err != nil {
		t.Fatal(err)
	}
}
