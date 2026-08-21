//go:build system

package system

import "testing"

func TestInstalledTerminalCommands(t *testing.T) {
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
	results, err := VerifyTerminalCommands(lifecycle.Client())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 7 {
		t.Fatalf("verified %d terminal plugins", len(results))
	}
	if err := VerifyInstalledUI(lifecycle.Client()); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.Shutdown(); err != nil {
		t.Fatal(err)
	}
}
