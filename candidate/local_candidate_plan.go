package candidate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func pluginManifestContract(path, contractVersion string) (string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var manifest struct {
		Implements []struct {
			ID      string `json:"id"`
			Version string `json:"version"`
		} `json:"implements"`
		Contributes struct {
			Programs []struct {
				ID   string `json:"id"`
				Kind string `json:"kind"`
			} `json:"programs"`
		} `json:"contributes"`
	}
	if err := json.Unmarshal(body, &manifest); err != nil {
		return "", err
	}
	implementsContract := false
	for _, implemented := range manifest.Implements {
		if implemented.ID == "soksak-spec-plugin-terminal" && implemented.Version == contractVersion {
			implementsContract = true
		}
	}
	if !implementsContract {
		return "", fmt.Errorf("plugin manifest does not implement terminal contract %s", contractVersion)
	}
	program := ""
	for _, candidate := range manifest.Contributes.Programs {
		if candidate.Kind == "view" {
			if program != "" || !candidateIDPattern.MatchString(candidate.ID) {
				return "", fmt.Errorf("plugin manifest must expose one view program")
			}
			program = candidate.ID
		}
	}
	if program == "" {
		return "", fmt.Errorf("plugin manifest exposes no view program")
	}
	return program, nil
}

func validateRuntimeDependencies(plugin verifiedLocalRelease, sidecars map[string]verifiedLocalRelease) (map[string]bool, error) {
	seen := map[string]bool{}
	for _, dependency := range plugin.document.RuntimeDependencies.Sidecars {
		sidecar, exists := sidecars[dependency.ID]
		if !exists || seen[dependency.ID] || dependency.Version != sidecar.document.Version ||
			dependency.Size != sidecar.releaseBytes.Size || dependency.SHA256 != sidecar.releaseBytes.SHA256 {
			return nil, fmt.Errorf("plugin runtime dependency mismatch: %s=%s", plugin.document.ID, dependency.ID)
		}
		seen[dependency.ID] = true
	}
	if len(seen) == 0 {
		return nil, fmt.Errorf("plugin runtime dependency set is empty: %s", plugin.document.ID)
	}
	return seen, nil
}

func writeImmutablePlan(path string, body []byte) (string, error) {
	if existing, err := os.ReadFile(path); err == nil {
		if bytes.Equal(existing, body) {
			return "unchanged", nil
		}
		return "", fmt.Errorf("candidate plan already exists with different bytes: %s", path)
	} else if !os.IsNotExist(err) {
		return "", err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", err
	}
	if _, err = file.Write(body); err != nil {
		_ = file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return "written", nil
}

func ComposeLocal(options LocalComposeOptions) (string, error) {
	for name, value := range map[string]string{
		"plan": options.Plan, "release store": options.Store, "output": options.Output,
	} {
		if !filepath.IsAbs(value) {
			return "", fmt.Errorf("%s path must be absolute", name)
		}
	}
	store, err := regularDirectory(options.Store)
	if err != nil {
		return "", err
	}
	var selection localCandidatePlan
	if err := decodeStrict(options.Plan, &selection); err != nil {
		return "", err
	}
	if selection.Schema != "soksak-terminal-local-candidate-v1" || selection.Target == "" ||
		len(selection.Plugins) == 0 || len(selection.Sidecars) == 0 {
		return "", fmt.Errorf("local candidate plan identity is invalid")
	}
	contract, err := resolveLocalRelease(store, selection.Contract, "contract", selection.Target)
	if err != nil {
		return "", err
	}
	sidecars := map[string]verifiedLocalRelease{}
	for _, selected := range selection.Sidecars {
		if _, exists := sidecars[selected.ID]; exists {
			return "", fmt.Errorf("local candidate repeats sidecar: %s", selected.ID)
		}
		release, resolveErr := resolveLocalRelease(store, selected, "sidecar", selection.Target)
		if resolveErr != nil {
			return "", resolveErr
		}
		sidecars[selected.ID] = release
	}
	plugins := map[string]verifiedLocalRelease{}
	programs := map[string]string{}
	usedSidecars := map[string]bool{}
	for _, selected := range selection.Plugins {
		if _, exists := plugins[selected.ID]; exists {
			return "", fmt.Errorf("local candidate repeats plugin: %s", selected.ID)
		}
		release, resolveErr := resolveLocalRelease(store, selected, "plugin", selection.Target)
		if resolveErr != nil {
			return "", resolveErr
		}
		program, err := pluginManifestContract(filepath.Join(release.directory, release.document.Manifest.File), contract.document.Version)
		if err != nil {
			return "", err
		}
		dependencies, err := validateRuntimeDependencies(release, sidecars)
		if err != nil {
			return "", err
		}
		for id := range dependencies {
			usedSidecars[id] = true
		}
		plugins[selected.ID] = release
		programs[selected.ID] = program
	}
	if len(usedSidecars) != len(sidecars) {
		return "", fmt.Errorf("local candidate contains an unused sidecar")
	}
	contractArtifact, err := relativeArtifact(options.Output, filepath.Join(contract.directory, contract.artifact.File))
	if err != nil {
		return "", err
	}
	output := outputPlan{PresentationContract: outputPresentation{
		ID: contract.document.ID, Version: contract.document.Version, Artifact: contractArtifact,
		SourceRepository: contract.document.Source.Repository, SourceCommit: contract.document.Source.Commit,
	}}
	for _, selected := range selection.Sidecars {
		release := sidecars[selected.ID]
		artifact, relativeErr := relativeArtifact(options.Output, filepath.Join(release.directory, release.artifact.File))
		if relativeErr != nil {
			return "", relativeErr
		}
		output.Components = append(output.Components, outputComponent{
			Kind: "sidecar", ID: selected.ID, Version: release.document.Version, Artifact: artifact,
			Manifest: release.artifact.Manifest, Target: selection.Target,
			SourceRepository: release.document.Source.Repository, SourceCommit: release.document.Source.Commit,
		})
	}
	for _, selected := range selection.Plugins {
		release := plugins[selected.ID]
		artifact, relativeErr := relativeArtifact(options.Output, filepath.Join(release.directory, release.artifact.File))
		if relativeErr != nil {
			return "", relativeErr
		}
		output.Components = append(output.Components, outputComponent{
			Kind: "plugin", ID: selected.ID, Version: release.document.Version, Artifact: artifact,
			Manifest: release.artifact.Manifest, Program: programs[selected.ID], SourceRepository: release.document.Source.Repository,
			SourceCommit: release.document.Source.Commit,
		})
	}
	body, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}
	body = append(body, '\n')
	return writeImmutablePlan(options.Output, body)
}
