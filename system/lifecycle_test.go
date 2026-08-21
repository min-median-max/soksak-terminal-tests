package system

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLifecycleRequiresDeclaredAbsoluteInputs(t *testing.T) {
	_, err := NewLifecycle(LifecycleConfig{
		App: "soksak", CLI: "/bin/sok", Socket: "/tmp/s.sock", Home: "/tmp/home",
		Runtime: "/tmp/runtime", Workspace: "/tmp/work", EvidenceDir: "/tmp/evidence", Identifier: "test",
	})
	if err == nil {
		t.Fatal("relative app path was accepted")
	}
}

func TestLifecycleRejectsUnsupportedControlSocketPath(t *testing.T) {
	root := t.TempDir()
	_, err := NewLifecycle(LifecycleConfig{
		App: filepath.Join(root, "soksak"), CLI: filepath.Join(root, "sok"),
		Socket: filepath.Join(root, strings.Repeat("runtime", 20), "control.sock"),
		Home:   filepath.Join(root, "home"), Runtime: filepath.Join(root, "runtime"),
		Workspace: filepath.Join(root, "work"), EvidenceDir: filepath.Join(root, "evidence"),
		Identifier: "test",
	})
	if err == nil {
		t.Fatal("unsupported control socket path was accepted")
	}
}

func TestCountExactLineIgnoresShellEchoAndMarkerPrefixes(t *testing.T) {
	text := "prefix=SOKSAK_DETACHED_\nSOKSAK_SCHEDULED_7\nprompt SOKSAK_DETACHED_7\njob done: ${prefix}7\n"
	if count := countExactLine(text, "SOKSAK_DETACHED_7"); count != 1 {
		t.Fatalf("count=%d", count)
	}
}

func TestDetachedMarkerCommandDoesNotContainTheWaitedMarker(t *testing.T) {
	const marker = "SOKSAK_DETACHED_7"
	command := detachedMarkerCommand(7, "SOKSAK_SCHEDULED_7")
	if strings.Contains(command, marker) {
		t.Fatalf("command echo contains waited marker: %s", command)
	}
}
