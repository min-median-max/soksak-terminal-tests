package fleet

import "testing"

func TestProfilesDeclareExactSupportedTerminalFleets(t *testing.T) {
	linux, err := ForTarget("linux", "aarch64-unknown-linux-gnu")
	if err != nil {
		t.Fatal(err)
	}
	if len(linux.Plugins) != 7 || len(linux.Sidecars) != 7 {
		t.Fatalf("linux plugins=%d sidecars=%d", len(linux.Plugins), len(linux.Sidecars))
	}
	windows, err := ForTarget("windows", "x86_64-pc-windows-msvc")
	if err != nil {
		t.Fatal(err)
	}
	if len(windows.Plugins) != 5 || len(windows.Sidecars) != 5 {
		t.Fatalf("windows plugins=%d sidecars=%d", len(windows.Plugins), len(windows.Sidecars))
	}
	for _, forbidden := range []string{"soksak-plugin-terminal-kitty", "soksak-plugin-terminal-shitty"} {
		for _, plugin := range windows.Plugins {
			if plugin.ID == forbidden {
				t.Fatalf("Windows profile contains unsupported plugin %s", forbidden)
			}
		}
	}
	darwin, err := ForTarget("darwin", "aarch64-apple-darwin")
	if err != nil || len(darwin.Plugins) != 7 || len(darwin.Sidecars) != 7 {
		t.Fatalf("darwin=%+v err=%v", darwin, err)
	}
}

func TestProfileRejectsUnknownPlatform(t *testing.T) {
	if _, err := ForTarget("freebsd", "x86_64-unknown-freebsd"); err == nil {
		t.Fatal("accepted an undeclared platform")
	}
}

func TestProfileRejectsAPlatformTargetMismatch(t *testing.T) {
	if _, err := ForTarget("linux", "aarch64-apple-darwin"); err == nil {
		t.Fatal("accepted a target owned by another platform")
	}
}
