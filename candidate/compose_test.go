package candidate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fixtureSource struct {
	ID           string `json:"id"`
	Repository   string `json:"repository"`
	SourceCommit string `json:"sourceCommit"`
	Language     string `json:"language,omitempty"`
	Profile      string `json:"profile,omitempty"`
}

type fixturePlan struct {
	Schema   string          `json:"schema"`
	Target   string          `json:"target"`
	Spec     fixtureSource   `json:"spec"`
	Contract fixtureSource   `json:"contract"`
	Kit      fixtureSource   `json:"kit"`
	Plugins  []fixtureSource `json:"plugins"`
	Sidecars []fixtureSource `json:"sidecars"`
}

func hash(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func writeJSON(t *testing.T, path string, value any) []byte {
	t.Helper()
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, '\n')
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	return body
}

func createCandidateFixture(t *testing.T, root string, source fixtureSource, kind, target string, dependencies []fixtureSource) {
	t.Helper()
	directory := filepath.Join(root, source.ID)
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	manifestName := kind + ".json"
	manifest := writeJSON(t, filepath.Join(directory, manifestName), map[string]any{"id": source.ID, "version": "0.0.1"})
	archiveName := source.ID + "-0.0.1-any.tgz"
	format := "tgz"
	artifactTarget := "any"
	if kind == "sidecar" {
		archiveName = source.ID + "-0.0.1-" + target + ".tar.gz"
		format = "tar.gz"
		artifactTarget = target
	}
	archive := []byte("candidate bytes for " + source.ID)
	if err := os.WriteFile(filepath.Join(directory, archiveName), archive, 0o600); err != nil {
		t.Fatal(err)
	}
	repository := "https://github.com/" + source.Repository
	release := map[string]any{
		"kind": kind, "id": source.ID, "version": "0.0.1",
		"manifest": map[string]any{"url": repository + "/releases/download/v0.0.1/" + manifestName, "size": len(manifest), "sha256": hash(manifest)},
		"source":   map[string]any{"repository": repository, "commit": source.SourceCommit},
		"artifacts": []map[string]any{{
			"target": artifactTarget, "url": repository + "/releases/download/v0.0.1/" + archiveName,
			"size": len(archive), "sha256": hash(archive), "format": format, "manifest": manifestName,
		}},
		"evidence": []any{},
	}
	releaseBody := writeJSON(t, filepath.Join(directory, "release.json"), release)
	files := []map[string]any{
		{"path": archiveName, "size": len(archive), "sha256": hash(archive)},
		{"path": manifestName, "size": len(manifest), "sha256": hash(manifest)},
		{"path": "release.json", "size": len(releaseBody), "sha256": hash(releaseBody)},
	}
	buildEvidence := []map[string]any{}
	for index, dependency := range dependencies {
		name := "dependency-" + string(rune('a'+index)) + ".json"
		receipt := writeJSON(t, filepath.Join(directory, name), map[string]any{
			"schema":    "soksak-candidate-input-receipt-v1",
			"artifact":  map[string]any{"name": "fixture", "sha256": hash([]byte(dependency.ID)), "candidateManifestSHA256": hash([]byte(dependency.SourceCommit))},
			"component": map[string]any{"kind": map[string]string{"soksak-contract-plugin-terminal": "contract", "soksak-kit-plugin-terminal": "kit"}[dependency.ID], "id": dependency.ID, "version": "0.0.1"},
			"source":    map[string]any{"repository": "https://github.com/" + dependency.Repository, "commit": dependency.SourceCommit},
		})
		entry := map[string]any{"path": name, "size": len(receipt), "sha256": hash(receipt)}
		files = append(files, entry)
		buildEvidence = append(buildEvidence, entry)
	}
	for left := 0; left < len(files); left++ {
		for right := left + 1; right < len(files); right++ {
			if files[right]["path"].(string) < files[left]["path"].(string) {
				files[left], files[right] = files[right], files[left]
			}
		}
	}
	writeJSON(t, filepath.Join(directory, "candidate-artifact.json"), map[string]any{
		"schema":        "soksak-candidate-artifact-v1",
		"component":     map[string]any{"kind": kind, "id": source.ID, "version": "0.0.1"},
		"source":        map[string]any{"repository": repository, "commit": source.SourceCommit},
		"release":       map[string]any{"path": "release.json", "size": len(releaseBody), "sha256": hash(releaseBody)},
		"buildEvidence": buildEvidence,
		"files":         files,
	})
}

func TestComposeProducesOnlyRuntimeComponentsAndPresentationContract(t *testing.T) {
	root := t.TempDir()
	artifacts := filepath.Join(root, "artifacts")
	if err := os.Mkdir(artifacts, 0o700); err != nil {
		t.Fatal(err)
	}
	spec := fixtureSource{ID: "soksak-spec", Repository: "soksak-ai/soksak-spec", SourceCommit: strings.Repeat("1", 40)}
	contract := fixtureSource{ID: "soksak-contract-plugin-terminal", Repository: "soksak-ai/soksak-contract-plugin-terminal", SourceCommit: strings.Repeat("2", 40)}
	kit := fixtureSource{ID: "soksak-kit-plugin-terminal", Repository: "soksak-ai/soksak-kit-plugin-terminal", SourceCommit: strings.Repeat("3", 40)}
	plugin := fixtureSource{ID: "soksak-plugin-terminal-example", Repository: "soksak-ai/soksak-plugin-terminal-example", SourceCommit: strings.Repeat("4", 40)}
	sidecar := fixtureSource{ID: "soksak-sidecar-example", Repository: "soksak-ai/soksak-sidecar-example", SourceCommit: strings.Repeat("5", 40), Language: "rust", Profile: "standard"}
	sourcePlan := fixturePlan{Schema: "soksak-terminal-native-candidate-v1", Target: "aarch64-apple-darwin", Spec: spec, Contract: contract, Kit: kit, Plugins: []fixtureSource{plugin}, Sidecars: []fixtureSource{sidecar}}
	sourcePlanPath := filepath.Join(root, "source-plan.json")
	writeJSON(t, sourcePlanPath, sourcePlan)
	createCandidateFixture(t, artifacts, spec, "spec", sourcePlan.Target, nil)
	createCandidateFixture(t, artifacts, contract, "contract", sourcePlan.Target, nil)
	createCandidateFixture(t, artifacts, kit, "kit", sourcePlan.Target, []fixtureSource{contract})
	createCandidateFixture(t, artifacts, plugin, "plugin", sourcePlan.Target, []fixtureSource{contract, kit})
	createCandidateFixture(t, artifacts, sidecar, "sidecar", sourcePlan.Target, nil)
	output := filepath.Join(root, "candidate-plan.json")
	if err := Compose(sourcePlanPath, artifacts, output); err != nil {
		t.Fatal(err)
	}
	var plan struct {
		PresentationContract struct{ ID, SourceCommit string }         `json:"presentationContract"`
		Components           []struct{ Kind, ID, SourceCommit string } `json:"components"`
	}
	body, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, &plan); err != nil {
		t.Fatal(err)
	}
	if plan.PresentationContract.ID != contract.ID || plan.PresentationContract.SourceCommit != contract.SourceCommit {
		t.Fatalf("presentation=%+v", plan.PresentationContract)
	}
	if len(plan.Components) != 2 || plan.Components[0].Kind != "sidecar" || plan.Components[1].Kind != "plugin" {
		t.Fatalf("runtime components=%+v", plan.Components)
	}
	if _, err := os.Stat(filepath.Join(artifacts, spec.ID, "candidate-artifact.json")); err != nil {
		t.Fatal("composition mutated candidate artifacts")
	}
	if err := os.Remove(output); err != nil {
		t.Fatal(err)
	}
	pluginArchive := filepath.Join(artifacts, plugin.ID, plugin.ID+"-0.0.1-any.tgz")
	file, err := os.OpenFile(pluginArchive, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("changed"); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := Compose(sourcePlanPath, artifacts, output); err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("changed candidate bytes were accepted: %v", err)
	}
}
