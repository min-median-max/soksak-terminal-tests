package terminaltests

import (
	"encoding/json"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

type candidateSource struct {
	ID           string `json:"id"`
	Repository   string `json:"repository"`
	SourceCommit string `json:"sourceCommit"`
	Language     string `json:"language,omitempty"`
	Profile      string `json:"profile,omitempty"`
}

type candidateSourcePlan struct {
	Schema   string            `json:"schema"`
	Target   string            `json:"target"`
	Spec     candidateSource   `json:"spec"`
	Contract candidateSource   `json:"contract"`
	Kit      candidateSource   `json:"kit"`
	Plugins  []candidateSource `json:"plugins"`
	Sidecars []candidateSource `json:"sidecars"`
}

func TestTerminalNativeCandidateSourcePlansAreExactAndPortable(t *testing.T) {
	for name, path := range map[string]string{
		"remote": "candidate/terminal-native-darwin-arm64.json",
		"local":  "candidate/terminal-native-local-darwin-arm64.json",
	} {
		t.Run(name, func(t *testing.T) { assertCandidateSourcePlan(t, path) })
	}
}

func assertCandidateSourcePlan(t *testing.T, path string) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "file:") || strings.Contains(string(body), "/tmp/") || strings.Contains(string(body), "/Users/") {
		t.Fatal("candidate source plan contains a local locator")
	}
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	var plan candidateSourcePlan
	if err := decoder.Decode(&plan); err != nil {
		t.Fatal(err)
	}
	if plan.Schema != "soksak-terminal-native-candidate-v1" || plan.Target != "aarch64-apple-darwin" {
		t.Fatalf("candidate source plan identity=%s/%s", plan.Schema, plan.Target)
	}
	if len(plan.Plugins) != 7 || len(plan.Sidecars) != 7 {
		t.Fatalf("candidate matrix plugins=%d sidecars=%d", len(plan.Plugins), len(plan.Sidecars))
	}
	commit := regexp.MustCompile(`^[0-9a-f]{40}$`)
	all := append([]candidateSource{plan.Spec, plan.Contract, plan.Kit}, append(plan.Plugins, plan.Sidecars...)...)
	for _, component := range all {
		if component.ID == "" || component.Repository != "soksak-ai/"+component.ID || !commit.MatchString(component.SourceCommit) {
			t.Fatalf("invalid candidate source: %+v", component)
		}
	}
	pluginIDs := make([]string, len(plan.Plugins))
	for index, plugin := range plan.Plugins {
		pluginIDs[index] = plugin.ID
		if plugin.Language != "" || plugin.Profile != "" {
			t.Fatalf("plugin source carries native build settings: %+v", plugin)
		}
	}
	sortedPlugins := append([]string(nil), pluginIDs...)
	sort.Strings(sortedPlugins)
	if strings.Join(pluginIDs, "\n") != strings.Join(sortedPlugins, "\n") {
		t.Fatal("plugin candidate sources are not sorted")
	}
	profiles := map[string]string{
		"soksak-sidecar-pty":                "go/standard",
		"soksak-sidecar-terminal-alacritty": "rust/standard",
		"soksak-sidecar-terminal-ghostty":   "rust/ghostty",
		"soksak-sidecar-terminal-kitty":     "rust/kitty",
		"soksak-sidecar-terminal-shitty":    "rust/shitty",
		"soksak-sidecar-terminal-vt100":     "rust/standard",
		"soksak-sidecar-terminal-wezterm":   "rust/standard",
	}
	for _, sidecar := range plan.Sidecars {
		if profiles[sidecar.ID] != sidecar.Language+"/"+sidecar.Profile {
			t.Fatalf("sidecar candidate profile mismatch: %+v", sidecar)
		}
	}
}
