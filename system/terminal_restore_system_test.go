//go:build system

package system

import (
	"os"
	"testing"
)

func TestInstalledTerminalWarmAndArchivedRestore(t *testing.T) {
	lifecycle, err := NewLifecycle(lifecycleConfigFromEnvironment(t, "restore"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(lifecycle.Close)
	if err := lifecycle.PrepareHome(os.Getenv("SOKSAK_TEST_SETTINGS")); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.Start(); err != nil {
		t.Fatal(err)
	}
	if _, err := lifecycle.OpenWorkspace(); err != nil {
		t.Fatal(err)
	}
	views, err := VerifyTerminalCommands(lifecycle.Client())
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyWarmAndArchivedRestore(lifecycle, views); err != nil {
		t.Fatal(err)
	}
	if err := VerifyInstalledUI(lifecycle.Client()); err != nil {
		t.Fatal(err)
	}
}
