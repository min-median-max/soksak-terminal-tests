//go:build system

package system

import "testing"

func TestInstalledTerminalCommands(t *testing.T) {
	profile := profileFromEnvironment(t)
	lifecycle, err := NewLifecycle(lifecycleConfigFromEnvironment(t, "commands"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(lifecycle.Close)
	if err := lifecycle.Start(); err != nil {
		t.Fatal(err)
	}
	if err := InstallTerminalFleet(profile, lifecycle.Client()); err != nil {
		t.Fatal(err)
	}
	environment, err := ReadRuntimeEnvironment(lifecycle.Client())
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateTerminalInventory(profile, environment); err != nil {
		t.Fatal(err)
	}
	if _, err := lifecycle.OpenWorkspace(); err != nil {
		t.Fatal(err)
	}
	results, err := VerifyTerminalCommands(profile, lifecycle.Client())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != len(profile.Plugins) {
		t.Fatalf("verified %d terminal plugins", len(results))
	}
	if err := VerifyInstalledUI(lifecycle.Client()); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.Finish(); err != nil {
		t.Fatal(err)
	}
}
