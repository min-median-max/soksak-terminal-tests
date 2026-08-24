package system

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLifecycleStartupUsesTheOwnedReadinessEventWithoutPolling(t *testing.T) {
	body, err := os.ReadFile("lifecycle.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(body)
	if strings.Contains(source, "time.Sleep(") {
		t.Fatal("lifecycle startup still polls with time.Sleep")
	}
	if !strings.Contains(source, "soksak.control.ready") {
		t.Fatal("lifecycle startup does not consume the owned readiness event")
	}
}

func TestLifecyclePublishesProcessWindowAndSidecarOwnershipEvidence(t *testing.T) {
	body, err := os.ReadFile("lifecycle.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(body)
	for _, required := range []string{
		"process-window-ownership.json",
		"app.environment",
		"window_list",
		"window.monitors",
		"window.input.state",
		"sidecar_status",
		"applicationExited",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("lifecycle ownership evidence omits %q", required)
		}
	}
}

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
	text := "prefix=SOKSAK_DETACHED_\r\nSOKSAK_SCHEDULED_7\r\nprompt SOKSAK_DETACHED_7\r\nSOKSAK_DETACHED_7\r\njob done: ${prefix}7\r\n"
	if count := countExactLine(text, "SOKSAK_DETACHED_7"); count != 1 {
		t.Fatalf("count=%d", count)
	}
}

func TestDetachedMarkerCommandDoesNotContainTheWaitedMarker(t *testing.T) {
	const marker = "SOKSAK_DETACHED_7"
	for _, platform := range []string{"windows", "darwin", "linux"} {
		command, err := detachedMarkerCommand(platform, marker, "SOKSAK_SCHEDULED_7")
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(command, marker) {
			t.Fatalf("%s command echo contains waited marker: %s", platform, command)
		}
	}
}
