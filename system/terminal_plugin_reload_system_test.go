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
	if len(profile.Plugins) == 0 {
		t.Fatal("terminal fleet has no plugins")
	}
	view, err := openTerminalForScenario(lifecycle.Client(), profile.Plugins[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyPluginReloadLifetime(profile.Platform, lifecycle.Client(), view, 3); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.Finish(); err != nil {
		t.Fatal(err)
	}
}
