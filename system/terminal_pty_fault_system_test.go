//go:build system

package system

import "testing"

func TestInstalledTerminalPtyFaultRecovery(t *testing.T) {
	profile := profileFromEnvironment(t)
	lifecycle, err := NewLifecycle(lifecycleConfigFromEnvironment(t, "pty-fault"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(lifecycle.Close)
	if err := lifecycle.Start(); err != nil {
		t.Fatal(err)
	}
	if err := InstallConfiguredTerminalFleet(profile, lifecycle.Client()); err != nil {
		t.Fatal(err)
	}
	if _, err := lifecycle.OpenWorkspace(); err != nil {
		t.Fatal(err)
	}
	views, err := VerifyTerminalCommands(profile, lifecycle.Client())
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyPtyFaultRecovery(profile, lifecycle.Client(), views); err != nil {
		t.Fatal(err)
	}
	if err := VerifyInstalledUI(lifecycle.Client()); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.Finish(); err != nil {
		t.Fatal(err)
	}
}
