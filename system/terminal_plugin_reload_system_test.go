//go:build system

package system

import "testing"

func TestInstalledTerminalPluginReloadLifetime(t *testing.T) {
	profile := profileFromEnvironment(t)
	lifecycle, err := NewLifecycle(lifecycleConfigFromEnvironment(t, "plugin-reload"))
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
	if len(views) == 0 {
		t.Fatal("terminal fleet opened no views")
	}
	if err := VerifyPluginReloadLifetime(profile.Platform, lifecycle.Client(), views[0], 3); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.Finish(); err != nil {
		t.Fatal(err)
	}
}
