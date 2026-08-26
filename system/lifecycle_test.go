package system

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type lifecycleCleanupCaller struct {
	calls      []string
	statusRead int
}

func (caller *lifecycleCleanupCaller) Call(command string, params map[string]any) (map[string]any, error) {
	id, _ := params["id"].(string)
	name, _ := params["name"].(string)
	caller.calls = append(caller.calls, command+":"+id+name)
	switch command {
	case "plugin.list":
		return map[string]any{"plugins": []any{
			map[string]any{"id": "plugin-b", "status": "enabled"},
			map[string]any{"id": "plugin-disabled", "status": "disabled"},
			map[string]any{"id": "plugin-a", "status": "enabled"},
		}}, nil
	case "plugin.disable":
		return map[string]any{"id": id, "status": "disabled"}, nil
	case "sidecar_status":
		caller.statusRead++
		if caller.statusRead == 1 {
			return map[string]any{
				"open":     []any{map[string]any{"name": "sidecar-b"}},
				"recorded": []any{map[string]any{"name": "sidecar-a"}},
			}, nil
		}
		return map[string]any{"open": []any{}, "recorded": []any{}}, nil
	case "sidecar_stop":
		return map[string]any{"name": name, "running": false}, nil
	default:
		return nil, fmt.Errorf("unexpected command %s", command)
	}
}

func (caller *lifecycleCleanupCaller) CallValue(string, map[string]any) (any, error) {
	return nil, fmt.Errorf("unexpected value command")
}

func TestLifecycleStartupUsesTheOwnedReadinessEventWithoutPolling(t *testing.T) {
	body, err := os.ReadFile("lifecycle.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(body)
	if strings.Contains(source, "time.Sleep(") {
		t.Fatal("lifecycle startup still polls with time.Sleep")
	}
	if !strings.Contains(source, "soksak.host.ready") {
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

func TestOwnedSidecarNamesIncludesOpenAndRecordedWithoutDuplicates(t *testing.T) {
	status := map[string]any{
		"open": []any{
			map[string]any{"name": "soksak-sidecar-pty"},
		},
		"recorded": []any{
			map[string]any{"name": "soksak-sidecar-pty"},
			map[string]any{"name": "soksak-sidecar-terminal-vt100"},
		},
	}
	names := ownedSidecarNames(status)
	if strings.Join(names, ",") != "soksak-sidecar-pty,soksak-sidecar-terminal-vt100" {
		t.Fatalf("names=%v", names)
	}
}

func TestLifecycleDisablesPluginsBeforeStoppingPersistentSidecars(t *testing.T) {
	caller := &lifecycleCleanupCaller{}
	if err := quiesceTestRuntime(caller); err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"plugin.list:", "plugin.disable:plugin-a", "plugin.disable:plugin-b",
		"sidecar_status:", "sidecar_stop:sidecar-a", "sidecar_stop:sidecar-b", "sidecar_status:",
	}, ",")
	if got := strings.Join(caller.calls, ","); got != want {
		t.Fatalf("cleanup order=%s want=%s", got, want)
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
