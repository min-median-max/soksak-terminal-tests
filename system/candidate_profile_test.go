package system

import (
	"testing"

	"github.com/min-median-max/soksak-terminal-tests/fleet"
)

func TestCandidatePlanOwnsTheExpectedInstalledVersions(t *testing.T) {
	profile := fleet.Profile{
		Platform: "darwin", Target: "aarch64-apple-darwin",
		Plugins:  []fleet.Plugin{{Component: fleet.Component{ID: "plugin-a", Version: "1.0.0"}, Sidecar: "sidecar-a"}},
		Sidecars: []fleet.Component{{ID: "sidecar-a", Version: "1.0.0"}},
	}
	plan := CandidatePlan{Components: []CandidateComponent{
		{Kind: "plugin", ID: "plugin-a", Version: "1.1.0"},
		{Kind: "sidecar", ID: "sidecar-a", Version: "1.2.0", Target: "aarch64-apple-darwin"},
	}}
	configured, err := applyCandidateProfile(profile, plan)
	if err != nil {
		t.Fatal(err)
	}
	if configured.Plugins[0].Version != "1.1.0" || configured.Sidecars[0].Version != "1.2.0" {
		t.Fatalf("configured profile=%+v", configured)
	}
	plan.Components = plan.Components[:1]
	if _, err := applyCandidateProfile(profile, plan); err == nil {
		t.Fatal("candidate plan missing a configured sidecar was accepted")
	}
}
