package candidate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type sourceComponent struct {
	ID           string `json:"id"`
	Repository   string `json:"repository"`
	SourceCommit string `json:"sourceCommit"`
	Language     string `json:"language,omitempty"`
	Profile      string `json:"profile,omitempty"`
}

type sourcePlan struct {
	Schema   string            `json:"schema"`
	Target   string            `json:"target"`
	Spec     sourceComponent   `json:"spec"`
	Contract sourceComponent   `json:"contract"`
	Kit      sourceComponent   `json:"kit"`
	Plugins  []sourceComponent `json:"plugins"`
	Sidecars []sourceComponent `json:"sidecars"`
}

type fileReference struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type candidateIdentity struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	Version string `json:"version"`
}

type candidateSource struct {
	Repository string `json:"repository"`
	Commit     string `json:"commit"`
}

type artifactEnvelope struct {
	Schema        string            `json:"schema"`
	Component     candidateIdentity `json:"component"`
	Source        candidateSource   `json:"source"`
	Release       fileReference     `json:"release"`
	BuildEvidence []fileReference   `json:"buildEvidence"`
	Files         []fileReference   `json:"files"`
}

type integrityReference struct {
	URL    string `json:"url"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type releaseArtifact struct {
	Target   string `json:"target"`
	URL      string `json:"url"`
	Size     int64  `json:"size"`
	SHA256   string `json:"sha256"`
	Format   string `json:"format"`
	Manifest string `json:"manifest"`
}

type releaseDocument struct {
	Kind                string               `json:"kind"`
	ID                  string               `json:"id"`
	Version             string               `json:"version"`
	Manifest            integrityReference   `json:"manifest"`
	Source              candidateSource      `json:"source"`
	Artifacts           []releaseArtifact    `json:"artifacts"`
	Evidence            []integrityReference `json:"evidence"`
	RuntimeDependencies json.RawMessage      `json:"runtimeDependencies,omitempty"`
}

type inputReceipt struct {
	Schema    string            `json:"schema"`
	Artifact  json.RawMessage   `json:"artifact"`
	Component candidateIdentity `json:"component"`
	Source    candidateSource   `json:"source"`
}

type verifiedCandidate struct {
	directory    string
	envelope     artifactEnvelope
	release      releaseDocument
	artifact     string
	dependencies map[string]string
}

type outputComponent struct {
	Kind              string            `json:"kind"`
	ID                string            `json:"id"`
	Version           string            `json:"version"`
	Artifact          string            `json:"artifact"`
	Manifest          string            `json:"manifest"`
	Target            string            `json:"target,omitempty"`
	SourceRepository  string            `json:"sourceRepository"`
	SourceCommit      string            `json:"sourceCommit"`
	DependencyCommits map[string]string `json:"dependencyCommits,omitempty"`
}

type outputPresentation struct {
	ID               string `json:"id"`
	Version          string `json:"version"`
	Artifact         string `json:"artifact"`
	SourceRepository string `json:"sourceRepository"`
	SourceCommit     string `json:"sourceCommit"`
}

type outputPlan struct {
	Components           []outputComponent  `json:"components"`
	PresentationContract outputPresentation `json:"presentationContract"`
}

var commitPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)
var digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

func decodeStrict(path string, value any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if token, err := decoder.Token(); err != io.EOF || token != nil {
		return fmt.Errorf("%s: trailing JSON", path)
	}
	return nil
}

func fileDigest(path string) (int64, string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return 0, "", err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return 0, "", fmt.Errorf("candidate file is not regular: %s", path)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return 0, "", err
	}
	sum := sha256.Sum256(body)
	return int64(len(body)), hex.EncodeToString(sum[:]), nil
}

func expectedSources(plan sourcePlan) map[string]sourceComponent {
	result := map[string]sourceComponent{}
	for _, component := range append([]sourceComponent{plan.Spec, plan.Contract, plan.Kit}, append(plan.Plugins, plan.Sidecars...)...) {
		result[component.ID] = component
	}
	return result
}

func verifyEnvelope(directory string, expected sourceComponent, target string) (verifiedCandidate, error) {
	var envelope artifactEnvelope
	if err := decodeStrict(filepath.Join(directory, "candidate-artifact.json"), &envelope); err != nil {
		return verifiedCandidate{}, err
	}
	if envelope.Schema != "soksak-candidate-artifact-v1" || envelope.Component.ID != expected.ID ||
		envelope.Source.Repository != "https://github.com/"+expected.Repository || envelope.Source.Commit != expected.SourceCommit ||
		!commitPattern.MatchString(envelope.Source.Commit) {
		return verifiedCandidate{}, fmt.Errorf("candidate envelope identity mismatch: %s", expected.ID)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return verifiedCandidate{}, err
	}
	actual := []string{}
	for _, entry := range entries {
		if entry.Name() == "candidate-artifact.json" {
			continue
		}
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return verifiedCandidate{}, fmt.Errorf("candidate artifact contains a non-regular entry: %s", entry.Name())
		}
		actual = append(actual, entry.Name())
	}
	expectedNames := make([]string, len(envelope.Files))
	for index, file := range envelope.Files {
		if filepath.Base(file.Path) != file.Path || file.Size <= 0 || !digestPattern.MatchString(file.SHA256) {
			return verifiedCandidate{}, fmt.Errorf("candidate file reference is invalid: %s", file.Path)
		}
		expectedNames[index] = file.Path
		size, digest, err := fileDigest(filepath.Join(directory, file.Path))
		if err != nil || size != file.Size || digest != file.SHA256 {
			return verifiedCandidate{}, fmt.Errorf("candidate file digest mismatch: %s", file.Path)
		}
	}
	sort.Strings(actual)
	if strings.Join(actual, "\n") != strings.Join(expectedNames, "\n") {
		return verifiedCandidate{}, fmt.Errorf("candidate artifact inventory mismatch: %s", expected.ID)
	}
	if envelope.Release.Path != "release.json" {
		return verifiedCandidate{}, fmt.Errorf("candidate release reference mismatch: %s", expected.ID)
	}
	var release releaseDocument
	if err := decodeStrict(filepath.Join(directory, "release.json"), &release); err != nil {
		return verifiedCandidate{}, err
	}
	if release.Kind != envelope.Component.Kind || release.ID != envelope.Component.ID || release.Version != envelope.Component.Version ||
		release.Source != envelope.Source || len(release.Artifacts) == 0 {
		return verifiedCandidate{}, fmt.Errorf("candidate release identity mismatch: %s", expected.ID)
	}
	wantedTarget := "any"
	if release.Kind == "sidecar" {
		wantedTarget = target
	}
	var selected *releaseArtifact
	for index := range release.Artifacts {
		if release.Artifacts[index].Target == wantedTarget {
			if selected != nil {
				return verifiedCandidate{}, fmt.Errorf("candidate repeats target %s: %s", wantedTarget, expected.ID)
			}
			selected = &release.Artifacts[index]
		}
	}
	if selected == nil {
		return verifiedCandidate{}, fmt.Errorf("candidate omits target %s: %s", wantedTarget, expected.ID)
	}
	name := filepath.Base(selected.URL)
	size, digest, err := fileDigest(filepath.Join(directory, name))
	if err != nil || size != selected.Size || digest != selected.SHA256 {
		return verifiedCandidate{}, fmt.Errorf("candidate release artifact mismatch: %s", expected.ID)
	}
	dependencies := map[string]string{}
	for _, evidence := range envelope.BuildEvidence {
		var receipt inputReceipt
		if err := decodeStrict(filepath.Join(directory, evidence.Path), &receipt); err != nil || receipt.Schema != "soksak-candidate-input-receipt-v1" {
			continue
		}
		dependencies[receipt.Component.ID] = receipt.Source.Commit
	}
	return verifiedCandidate{directory: directory, envelope: envelope, release: release, artifact: filepath.Join(directory, name), dependencies: dependencies}, nil
}

func relativeArtifact(output, artifact string) (string, error) {
	relative, err := filepath.Rel(filepath.Dir(output), artifact)
	if err != nil || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("candidate artifact is outside plan root: %s", artifact)
	}
	return filepath.ToSlash(relative), nil
}

func Compose(sourcePlanPath, artifactsRoot, output string) error {
	if !filepath.IsAbs(sourcePlanPath) || !filepath.IsAbs(artifactsRoot) || !filepath.IsAbs(output) {
		return fmt.Errorf("candidate composition paths must be absolute")
	}
	var sources sourcePlan
	if err := decodeStrict(sourcePlanPath, &sources); err != nil {
		return err
	}
	if sources.Schema != "soksak-terminal-native-candidate-v1" || sources.Target != "aarch64-apple-darwin" {
		return fmt.Errorf("candidate source plan identity is invalid")
	}
	wanted := expectedSources(sources)
	entries, err := os.ReadDir(artifactsRoot)
	if err != nil {
		return err
	}
	verified := map[string]verifiedCandidate{}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("candidate root contains a non-directory: %s", entry.Name())
		}
		directory := filepath.Join(artifactsRoot, entry.Name())
		var envelope artifactEnvelope
		if err := decodeStrict(filepath.Join(directory, "candidate-artifact.json"), &envelope); err != nil {
			return err
		}
		expected, ok := wanted[envelope.Component.ID]
		if !ok || verified[envelope.Component.ID].directory != "" {
			return fmt.Errorf("unexpected or duplicate candidate: %s", envelope.Component.ID)
		}
		candidate, err := verifyEnvelope(directory, expected, sources.Target)
		if err != nil {
			return err
		}
		verified[envelope.Component.ID] = candidate
	}
	if len(verified) != len(wanted) {
		return fmt.Errorf("candidate closure count=%d want=%d", len(verified), len(wanted))
	}
	contract := verified[sources.Contract.ID]
	if contract.release.Kind != "contract" || contract.envelope.Component.ID != "soksak-contract-plugin-terminal" {
		return fmt.Errorf("presentation contract candidate is invalid")
	}
	contractArtifact, err := relativeArtifact(output, contract.artifact)
	if err != nil {
		return err
	}
	plan := outputPlan{PresentationContract: outputPresentation{
		ID: contract.release.ID, Version: contract.release.Version, Artifact: contractArtifact,
		SourceRepository: contract.release.Source.Repository, SourceCommit: contract.release.Source.Commit,
	}}
	for _, source := range append(sources.Sidecars, sources.Plugins...) {
		candidate := verified[source.ID]
		artifact, err := relativeArtifact(output, candidate.artifact)
		if err != nil {
			return err
		}
		component := outputComponent{
			Kind: candidate.release.Kind, ID: candidate.release.ID, Version: candidate.release.Version,
			Artifact: artifact, Manifest: candidate.release.Artifacts[0].Manifest,
			SourceRepository: candidate.release.Source.Repository, SourceCommit: candidate.release.Source.Commit,
		}
		if component.Kind == "sidecar" {
			component.Target = sources.Target
		} else if component.Kind == "plugin" {
			for _, dependency := range []sourceComponent{sources.Contract, sources.Kit} {
				if candidate.dependencies[dependency.ID] != dependency.SourceCommit {
					return fmt.Errorf("plugin %s dependency commit mismatch: %s", component.ID, dependency.ID)
				}
			}
			component.DependencyCommits = map[string]string{
				sources.Contract.ID: sources.Contract.SourceCommit,
				sources.Kit.ID:      sources.Kit.SourceCommit,
			}
		} else {
			return fmt.Errorf("runtime candidate has invalid kind: %s", component.Kind)
		}
		plan.Components = append(plan.Components, component)
	}
	body, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	file, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err = file.Write(body); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}
