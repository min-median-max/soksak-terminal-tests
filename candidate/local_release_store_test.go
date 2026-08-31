package candidate

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFixtureFile(t *testing.T, root, name string, body []byte) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
}

func createLocalRelease(t *testing.T, store, kind, id, version, commit, target string, manifest map[string]any, runtime any) fileReference {
	t.Helper()
	directory := filepath.Join(store, kind+"s", id, version)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	manifestName := kind + ".json"
	manifestBody, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	manifestBody = append(manifestBody, '\n')
	writeFixtureFile(t, directory, manifestName, manifestBody)
	artifactTarget, extension := "any", "any.tgz"
	if kind == "sidecar" {
		artifactTarget, extension = target, target+".tar.gz"
	}
	artifactName := id + "-" + version + "-" + extension
	artifactBody := []byte("immutable artifact for " + id)
	writeFixtureFile(t, directory, artifactName, artifactBody)
	release := map[string]any{
		"kind": kind, "id": id, "version": version,
		"manifest": map[string]any{"file": manifestName, "size": len(manifestBody), "sha256": hash(manifestBody)},
		"source":   map[string]any{"repository": "https://github.com/soksak-ai/" + id, "commit": commit},
		"artifacts": []map[string]any{{
			"target": artifactTarget, "file": artifactName, "size": len(artifactBody), "sha256": hash(artifactBody),
			"format": map[bool]string{true: "tar.gz", false: "tgz"}[kind == "sidecar"], "manifest": manifestName,
		}},
		"evidence": []any{},
	}
	if runtime != nil {
		release["runtimeDependencies"] = runtime
	}
	releaseBody := writeJSON(t, filepath.Join(directory, "release.json"), release)
	return fileReference{Path: filepath.Join(directory, "release.json"), Size: int64(len(releaseBody)), SHA256: hash(releaseBody)}
}

func TestComposeLocalReleaseStoreProducesCurrentImmutablePlan(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	store := filepath.Join(root, "store")
	contractCommit := strings.Repeat("1", 40)
	pluginCommit := strings.Repeat("2", 40)
	sidecarCommit := strings.Repeat("3", 40)
	contractRelease := createLocalRelease(t, store, "contract", "soksak-contract-plugin-terminal", "0.0.8", contractCommit, "", map[string]any{
		"id": "soksak-contract-plugin-terminal", "version": "0.0.8",
	}, nil)
	sidecarRelease := createLocalRelease(t, store, "sidecar", "soksak-sidecar-example", "0.0.3", sidecarCommit, "aarch64-apple-darwin", map[string]any{
		"id": "soksak-sidecar-example", "version": "0.0.3",
	}, nil)
	pluginRelease := createLocalRelease(t, store, "plugin", "soksak-plugin-terminal-example", "0.0.2", pluginCommit, "", map[string]any{
		"id": "soksak-plugin-terminal-example", "version": "0.0.2",
		"implements": []map[string]string{{"id": "soksak-spec-plugin-terminal", "version": "0.0.8"}},
	}, map[string]any{"sidecars": []map[string]any{{
		"id": "soksak-sidecar-example", "version": "0.0.3", "size": sidecarRelease.Size, "sha256": sidecarRelease.SHA256,
	}}})
	plan := localCandidatePlan{
		Schema: "soksak-terminal-local-candidate-v1", Target: "aarch64-apple-darwin",
		Contract: localReleaseSelection{ID: "soksak-contract-plugin-terminal", Version: "0.0.8", ReleaseSHA256: contractRelease.SHA256},
		Plugins:  []localReleaseSelection{{ID: "soksak-plugin-terminal-example", Version: "0.0.2", ReleaseSHA256: pluginRelease.SHA256}},
		Sidecars: []localReleaseSelection{{ID: "soksak-sidecar-example", Version: "0.0.3", ReleaseSHA256: sidecarRelease.SHA256}},
	}
	planPath := filepath.Join(root, "selection-plan.json")
	writeJSON(t, planPath, plan)
	output := filepath.Join(root, "candidate-plan.json")
	options := LocalComposeOptions{Plan: planPath, Store: store, Output: output}
	state, err := ComposeLocal(options)
	if err != nil || state != "written" {
		t.Fatalf("compose state=%s error=%v", state, err)
	}
	state, err = ComposeLocal(options)
	if err != nil || state != "unchanged" {
		t.Fatalf("repeat state=%s error=%v", state, err)
	}
	body, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(body, []byte(`"version": "0.0.2"`)) || !bytes.Contains(body, []byte(`"version": "0.0.3"`)) ||
		!bytes.Contains(body, []byte(`"sourceCommit": "`+contractCommit+`"`)) {
		t.Fatalf("candidate plan is incomplete: %s", body)
	}
	pluginReleasePath := filepath.Join(store, "plugins", "soksak-plugin-terminal-example", "0.0.2", "release.json")
	pluginReleaseBody, err := os.ReadFile(pluginReleasePath)
	if err != nil {
		t.Fatal(err)
	}
	changed := bytes.Replace(pluginReleaseBody, []byte(sidecarRelease.SHA256), []byte(strings.Repeat("0", 64)), 1)
	if bytes.Equal(changed, pluginReleaseBody) {
		t.Fatal("fixture did not contain the sidecar release digest")
	}
	if err := os.WriteFile(pluginReleasePath, changed, 0o600); err != nil {
		t.Fatal(err)
	}
	plan.Plugins[0].ReleaseSHA256 = hash(changed)
	changedPlan := filepath.Join(root, "changed-selection-plan.json")
	writeJSON(t, changedPlan, plan)
	_, err = ComposeLocal(LocalComposeOptions{Plan: changedPlan, Store: store, Output: filepath.Join(root, "changed-plan.json")})
	if err == nil || !strings.Contains(err.Error(), "runtime dependency") {
		t.Fatalf("changed runtime release was accepted: %v", err)
	}
}

func TestComposeLocalReleaseStoreAcceptsSidecarsPartitionedAcrossPlugins(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	store := filepath.Join(root, "store")
	contractRelease := createLocalRelease(t, store, "contract", "soksak-contract-plugin-terminal", "0.0.8", strings.Repeat("1", 40), "", map[string]any{
		"id": "soksak-contract-plugin-terminal", "version": "0.0.8",
	}, nil)
	var plugins []localReleaseSelection
	var sidecars []localReleaseSelection
	for index, suffix := range []string{"alpha", "beta"} {
		sidecarID := "soksak-sidecar-" + suffix
		sidecarRelease := createLocalRelease(t, store, "sidecar", sidecarID, "0.0.3", strings.Repeat(string(rune('2'+index)), 40), "aarch64-apple-darwin", map[string]any{
			"id": sidecarID, "version": "0.0.3",
		}, nil)
		pluginID := "soksak-plugin-terminal-" + suffix
		pluginRelease := createLocalRelease(t, store, "plugin", pluginID, "0.0.2", strings.Repeat(string(rune('4'+index)), 40), "", map[string]any{
			"id": pluginID, "version": "0.0.2",
			"implements": []map[string]string{{"id": "soksak-spec-plugin-terminal", "version": "0.0.8"}},
		}, map[string]any{"sidecars": []map[string]any{{
			"id": sidecarID, "version": "0.0.3", "size": sidecarRelease.Size, "sha256": sidecarRelease.SHA256,
		}}})
		plugins = append(plugins, localReleaseSelection{ID: pluginID, Version: "0.0.2", ReleaseSHA256: pluginRelease.SHA256})
		sidecars = append(sidecars, localReleaseSelection{ID: sidecarID, Version: "0.0.3", ReleaseSHA256: sidecarRelease.SHA256})
	}
	planPath := filepath.Join(root, "selection-plan.json")
	writeJSON(t, planPath, localCandidatePlan{
		Schema: "soksak-terminal-local-candidate-v1", Target: "aarch64-apple-darwin",
		Contract: localReleaseSelection{ID: "soksak-contract-plugin-terminal", Version: "0.0.8", ReleaseSHA256: contractRelease.SHA256},
		Plugins:  plugins, Sidecars: sidecars,
	})
	if _, err := ComposeLocal(LocalComposeOptions{Plan: planPath, Store: store, Output: filepath.Join(root, "candidate-plan.json")}); err != nil {
		t.Fatalf("partitioned runtime dependencies were rejected: %v", err)
	}
}

func TestLocalCandidateCompositionDoesNotReadComponentSourceOrRegistryStorage(t *testing.T) {
	for _, name := range []string{"local_candidate_plan.go", "local_release_store.go"} {
		body, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		text := string(body)
		for _, forbidden := range []string{"SourceWorkspace", "Workspace", "Registry", "discoverSourceRoots", "frontend/package.json"} {
			if strings.Contains(text, forbidden) {
				t.Errorf("%s reads forbidden input %q", name, forbidden)
			}
		}
	}
}
