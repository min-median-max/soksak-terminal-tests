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
		found := false
		for _, call := range caller.calls {
			if call.command != "plugin.install" || call.params["pluginId"] != plugin.ID {
				continue
			}
			found = true
			if call.params["registryId"] != "official" || call.params["timeoutMs"] != 60000 || len(call.params) != 3 {
				t.Fatalf("%s install params = %+v", plugin.ID, call.params)
			}
			if _, legacy := call.params["sidecars"]; legacy {
				t.Fatalf("%s install request selects sidecars", plugin.ID)
			}
		}
		if !found {
			t.Errorf("%s was not installed", plugin.ID)
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
