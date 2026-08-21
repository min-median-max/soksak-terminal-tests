//go:build system

package system

import "testing"

func TestInstalledTerminalWarmAndArchivedRestore(t *testing.T) {
	lifecycle, err := NewLifecycle(lifecycleConfigFromEnvironment())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(lifecycle.Close)
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
