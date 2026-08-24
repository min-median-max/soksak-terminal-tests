package system

import (
	"fmt"
	"testing"

	"github.com/min-median-max/soksak-terminal-tests/fleet"
	platformspec "github.com/soksak-ai/soksak-spec/go/platformspec"
)

type recordedCall struct {
	command string
	params  map[string]any
}
type recordingCaller struct {
	calls []recordedCall
}

func (caller *recordingCaller) Call(command string, params map[string]any) (map[string]any, error) {
	caller.calls = append(caller.calls, recordedCall{command, params})
	switch command {
	case "artifact_install_wait":
		return map[string]any{"phase": "committed", "sequence": float64(1)}, nil
	case "plugin.consent.chain":
		return map[string]any{"pending": []any{params["id"]}}, nil
	case "plugin.consent.grant":
		return map[string]any{"granted": true}, nil
	default:
		return map[string]any{"ok": true}, nil
	}
}
func (caller *recordingCaller) CallValue(command string, params map[string]any) (any, error) {
	return nil, fmt.Errorf("unexpected value call: %s", command)
}

func TestInstallTerminalFleetUsesPublicInstallConsentAndActivationCommands(t *testing.T) {
	profile, err := fleet.ForTarget("windows", "x86_64-pc-windows-msvc")
	if err != nil {
		t.Fatal(err)
	}
	caller := &recordingCaller{}
	if err := InstallTerminalFleet(profile, caller); err != nil {
		t.Fatal(err)
	}
	if caller.calls[0].command != "plugin.catalog" || caller.calls[0].params["refresh"] != true {
		t.Fatalf("first call = %+v", caller.calls[0])
	}
	for _, plugin := range profile.Plugins {
		started, materialized, observed := false, false, false
		for _, call := range caller.calls {
			if call.params["pluginId"] != plugin.ID && call.params["rootId"] != plugin.ID {
				continue
			}
			switch call.command {
			case "plugin.install":
				started = true
				if call.params["registryId"] != "official" || len(call.params) != 2 {
					t.Fatalf("%s install params = %+v", plugin.ID, call.params)
				}
				if _, deadline := call.params["timeoutMs"]; deadline {
					t.Fatalf("%s start request owns the transaction deadline", plugin.ID)
				}
			case "plugin.install.wait":
				observed = true
				if call.params["phase"] != "installed" || call.params["timeoutMs"] != 30000 {
					t.Fatalf("%s install wait params = %+v", plugin.ID, call.params)
				}
			case "artifact_install_wait":
				materialized = true
				if call.params["rootId"] != plugin.ID || call.params["afterSequence"] != uint64(0) || call.params["timeoutMs"] != 30000 {
					t.Fatalf("%s artifact wait params = %+v", plugin.ID, call.params)
				}
			}
			if _, legacy := call.params["sidecars"]; legacy {
				t.Fatalf("%s install request selects sidecars", plugin.ID)
			}
		}
		if !started || !materialized || !observed {
			t.Errorf("%s transaction started=%t materialized=%t observed=%t", plugin.ID, started, materialized, observed)
		}
	}
}

type environmentCaller struct {
	environment platformspec.Environment
}

func (caller environmentCaller) Call(string, map[string]any) (map[string]any, error) { return nil, nil }
func (caller environmentCaller) CallValue(command string, _ map[string]any) (any, error) {
	if command != "environment_get" {
		return nil, fmt.Errorf("unexpected command: %s", command)
	}
	return caller.environment, nil
}

func TestReadRuntimeEnvironmentParsesTheCoreEnvironment(t *testing.T) {
	want := platformspec.EmptyEnvironment()
	got, err := ReadRuntimeEnvironment(environmentCaller{environment: want})
	if err != nil {
		t.Fatal(err)
	}
	if got.Revision != want.Revision || len(got.Plugins) != 0 {
		t.Fatalf("environment = %+v", got)
	}
}
