package fleet

import "testing"

func TestProfilesDeclareExactSupportedTerminalFleets(t *testing.T) {
	linux, err := ForPlatform("linux")
	if err != nil {
		t.Fatal(err)
	}
	if len(linux.Plugins) != 7 || len(linux.RecoverySidecars) != 6 {
		t.Fatalf("linux plugins=%d recovery=%d", len(linux.Plugins), len(linux.RecoverySidecars))
	}
	windows, err := ForPlatform("windows")
	if err != nil {
		t.Fatal(err)
	}
	if len(windows.Plugins) != 5 || len(windows.RecoverySidecars) != 4 {
		t.Fatalf("windows plugins=%d recovery=%d", len(windows.Plugins), len(windows.RecoverySidecars))
	}
	for _, forbidden := range []string{"soksak-plugin-terminal-kitty", "soksak-plugin-terminal-shitty"} {
		for _, plugin := range windows.Plugins {
			if plugin.ID == forbidden {
				t.Fatalf("Windows profile contains unsupported plugin %s", forbidden)
			}
		}
	}
	darwin, err := ForPlatform("darwin")
	if err != nil || len(darwin.Plugins) != 7 || len(darwin.RecoverySidecars) != 6 {
		t.Fatalf("darwin=%+v err=%v", darwin, err)
	}
}

func TestProfileRejectsUnknownPlatform(t *testing.T) {
	if _, err := ForPlatform("freebsd"); err == nil {
		t.Fatal("accepted an undeclared platform")
	}
}
