package candidate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
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

func commitFixtureRepository(t *testing.T, root, id string, files map[string][]byte) string {
	t.Helper()
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		writeFixtureFile(t, root, name, body)
	}
	commands := [][]string{
		{"init", "-q"},
		{"remote", "add", "origin", "https://github.com/soksak-ai/" + id + ".git"},
		{"add", "."},
		{"-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "-qm", "fixture"},
	}
	for _, args := range commands {
		command := exec.Command("git", args...)
		command.Dir = root
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
	command := exec.Command("git", "rev-parse", "HEAD")
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(output))
}

func writePackageArchive(t *testing.T, path string, files map[string][]byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(file)
	gzipWriter.Header.ModTime = time.Unix(0, 0)
	tarWriter := tar.NewWriter(gzipWriter)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		body := files[name]
		header := &tar.Header{Name: "package/" + name, Mode: 0o644, Size: int64(len(body)), ModTime: time.Unix(0, 0)}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := tarWriter.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
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
	workspace := filepath.Join(root, "workspace")
	store := filepath.Join(root, "store")
	registry := filepath.Join(root, "registry")
	contractPackage := map[string]any{
		"name": "@soksak/soksak-contract-plugin-terminal", "version": "0.0.8",
		"exports": map[string]any{".": "./src/index.ts"},
	}
	kitPackage := map[string]any{
		"name": "@soksak/soksak-kit-plugin-terminal", "version": "0.0.55",
		"exports":          map[string]any{".": map[string]any{"types": "./src/index.ts", "default": "./dist/index.js"}},
		"peerDependencies": map[string]string{"@soksak/soksak-contract-plugin-terminal": "0.0.8"},
		"devDependencies":  map[string]string{"@soksak/soksak-contract-plugin-terminal": "0.0.8"},
	}
	pluginPackage := map[string]any{
		"name": "@soksak/soksak-plugin-terminal-example", "version": "0.0.2",
		"dependencies": map[string]string{
			"@soksak/soksak-contract-plugin-terminal": "0.0.8",
			"@soksak/soksak-kit-plugin-terminal":      "0.0.55",
		},
	}
	jsonBody := func(value any) []byte { body, _ := json.Marshal(value); return body }
	specCommit := commitFixtureRepository(t, filepath.Join(workspace, "owners", "spec"), "soksak-spec", map[string][]byte{"README.md": []byte("spec\n")})
	contractFiles := map[string][]byte{
		"contract.json":     jsonBody(map[string]string{"id": "soksak-contract-plugin-terminal", "version": "0.0.8"}),
		"presentation.json": jsonBody(map[string]any{"version": 2}),
		"src/index.ts":      []byte("export const contract = true;\n"),
		"src/pane-key.ts":   []byte("export const paneKey = true;\n"),
		"package.json":      jsonBody(contractPackage),
	}
	contractRoot := filepath.Join(workspace, "owners", "contract")
	contractCommit := commitFixtureRepository(t, contractRoot, "soksak-contract-plugin-terminal", contractFiles)
	kitFiles := map[string][]byte{
		"kit.json":      jsonBody(map[string]string{"id": "soksak-kit-plugin-terminal", "version": "0.0.55"}),
		"dist/index.js": []byte("export const kit = true;\n"),
		"src/index.ts":  []byte("export const kit = true;\n"),
		"package.json":  jsonBody(kitPackage),
	}
	kitRoot := filepath.Join(workspace, "owners", "kit")
	kitCommit := commitFixtureRepository(t, kitRoot, "soksak-kit-plugin-terminal", kitFiles)
	pluginCommit := commitFixtureRepository(t, filepath.Join(workspace, "owners", "plugin"), "soksak-plugin-terminal-example", map[string][]byte{
		"frontend/package.json": jsonBody(pluginPackage),
	})
	unrelated := filepath.Join(workspace, "owners", "unrelated")
	commitFixtureRepository(t, unrelated, "unrelated", map[string][]byte{"README.md": []byte("unrelated\n")})
	removeRemote := exec.Command("git", "remote", "remove", "origin")
	removeRemote.Dir = unrelated
	if output, err := removeRemote.CombinedOutput(); err != nil {
		t.Fatalf("remove unrelated remote: %v: %s", err, output)
	}
	sidecarCommit := commitFixtureRepository(t, filepath.Join(workspace, "owners", "sidecar"), "soksak-sidecar-example", map[string][]byte{
		"sidecar.json": jsonBody(map[string]string{"id": "soksak-sidecar-example", "version": "0.0.3"}),
	})
	writePackageArchive(t, filepath.Join(registry, "@soksak", "soksak-contract-plugin-terminal", "soksak-contract-plugin-terminal-0.0.8.tgz"), contractFiles)
	writePackageArchive(t, filepath.Join(registry, "@soksak", "soksak-kit-plugin-terminal", "soksak-kit-plugin-terminal-0.0.55.tgz"), kitFiles)
	sidecarRelease := createLocalRelease(t, store, "sidecar", "soksak-sidecar-example", "0.0.3", sidecarCommit, "aarch64-apple-darwin", map[string]any{
		"id": "soksak-sidecar-example", "version": "0.0.3",
	}, nil)
	createLocalRelease(t, store, "plugin", "soksak-plugin-terminal-example", "0.0.2", pluginCommit, "", map[string]any{
		"id": "soksak-plugin-terminal-example", "version": "0.0.2",
		"implements": []map[string]string{{"id": "soksak-spec-plugin-terminal", "version": "0.0.8"}},
	}, map[string]any{"sidecars": []map[string]any{{
		"id": "soksak-sidecar-example", "version": "0.0.3", "size": sidecarRelease.Size, "sha256": sidecarRelease.SHA256,
	}}})
	plan := fixturePlan{
		Schema: "soksak-terminal-native-candidate-v1", Target: "aarch64-apple-darwin",
		Spec:     fixtureSource{ID: "soksak-spec", Repository: "soksak-ai/soksak-spec", SourceCommit: specCommit},
		Contract: fixtureSource{ID: "soksak-contract-plugin-terminal", Repository: "soksak-ai/soksak-contract-plugin-terminal", SourceCommit: contractCommit},
		Kit:      fixtureSource{ID: "soksak-kit-plugin-terminal", Repository: "soksak-ai/soksak-kit-plugin-terminal", SourceCommit: kitCommit},
		Plugins:  []fixtureSource{{ID: "soksak-plugin-terminal-example", Repository: "soksak-ai/soksak-plugin-terminal-example", SourceCommit: pluginCommit}},
		Sidecars: []fixtureSource{{ID: "soksak-sidecar-example", Repository: "soksak-ai/soksak-sidecar-example", SourceCommit: sidecarCommit, Language: "rust", Profile: "standard"}},
	}
	planPath := filepath.Join(root, "source-plan.json")
	writeJSON(t, planPath, plan)
	output := filepath.Join(root, "candidate-plan.json")
	options := LocalComposeOptions{SourcePlan: planPath, Store: store, Registry: registry, Workspace: workspace, Output: output}
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
		!bytes.Contains(body, []byte(`"soksak-contract-plugin-terminal": "`+contractCommit+`"`)) {
		t.Fatalf("candidate plan is incomplete: %s", body)
	}
	pluginReleasePath := filepath.Join(store, "plugins", "soksak-plugin-terminal-example", "0.0.2", "release.json")
	pluginRelease, err := os.ReadFile(pluginReleasePath)
	if err != nil {
		t.Fatal(err)
	}
	changed := bytes.Replace(pluginRelease, []byte(sidecarRelease.SHA256), []byte(strings.Repeat("0", 64)), 1)
	if bytes.Equal(changed, pluginRelease) {
		t.Fatal("fixture did not contain the sidecar release digest")
	}
	if err := os.WriteFile(pluginReleasePath, changed, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = ComposeLocal(LocalComposeOptions{SourcePlan: planPath, Store: store, Registry: registry, Workspace: workspace, Output: filepath.Join(root, "changed-plan.json")})
	if err == nil || !strings.Contains(err.Error(), "runtime dependency") {
		t.Fatalf("changed runtime release was accepted: %v", err)
	}
}
