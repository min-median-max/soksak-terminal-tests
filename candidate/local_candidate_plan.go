package candidate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func pluginManifestContract(path, contractVersion string) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var manifest struct {
		Implements []struct {
			ID      string `json:"id"`
			Version string `json:"version"`
		} `json:"implements"`
	}
	if err := json.Unmarshal(body, &manifest); err != nil {
		return err
	}
	for _, implemented := range manifest.Implements {
		if implemented.ID == "soksak-spec-plugin-terminal" && implemented.Version == contractVersion {
			return nil
		}
	}
	return fmt.Errorf("plugin manifest does not implement terminal contract %s", contractVersion)
}

func packageDependency(metadata packageMetadata, name string) string {
	for _, dependencies := range []map[string]string{metadata.Dependencies, metadata.PeerDependencies, metadata.DevDependencies, metadata.OptionalDependencies} {
		if version := dependencies[name]; version != "" {
			return version
		}
	}
	return ""
}

func validatePluginDependencies(root string, contract, kit verifiedPackage) error {
	body, err := os.ReadFile(filepath.Join(root, "frontend", "package.json"))
	if err != nil {
		return err
	}
	var metadata packageMetadata
	if err := json.Unmarshal(body, &metadata); err != nil {
		return err
	}
	if packageDependency(metadata, contract.metadata.Name) != contract.metadata.Version ||
		packageDependency(metadata, kit.metadata.Name) != kit.metadata.Version {
		return fmt.Errorf("plugin package dependency versions differ: %s", metadata.Name)
	}
	return nil
}

func validateRuntimeDependencies(plugin verifiedLocalRelease, sidecars map[string]verifiedLocalRelease) error {
	seen := map[string]bool{}
	for _, dependency := range plugin.document.RuntimeDependencies.Sidecars {
		sidecar, exists := sidecars[dependency.ID]
		if !exists || seen[dependency.ID] || dependency.Version != sidecar.document.Version ||
			dependency.Size != sidecar.releaseBytes.Size || dependency.SHA256 != sidecar.releaseBytes.SHA256 {
			return fmt.Errorf("plugin runtime dependency mismatch: %s=%s", plugin.document.ID, dependency.ID)
		}
		seen[dependency.ID] = true
	}
	if len(seen) == 0 {
		return fmt.Errorf("plugin has no runtime dependency: %s", plugin.document.ID)
	}
	return nil
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
		"source plan": options.SourcePlan, "release store": options.Store, "registry": options.Registry,
		"source workspace": options.Workspace, "output": options.Output,
	} {
		if !filepath.IsAbs(value) {
			return "", fmt.Errorf("%s path must be absolute", name)
		}
	}
	store, err := regularDirectory(options.Store)
	if err != nil {
		return "", err
	}
	registry, err := regularDirectory(options.Registry)
	if err != nil {
		return "", err
	}
	var sources sourcePlan
	if err := decodeStrict(options.SourcePlan, &sources); err != nil {
		return "", err
	}
	if sources.Schema != "soksak-terminal-native-candidate-v1" || sources.Target == "" {
		return "", fmt.Errorf("candidate source plan identity is invalid")
	}
	roots, err := discoverSourceRoots(options.Workspace, sources)
	if err != nil {
		return "", err
	}
	contract, err := verifyRegistryPackage(roots[sources.Contract.ID], registry, "contract", sources.Contract.ID)
	if err != nil {
		return "", err
	}
	kit, err := verifyRegistryPackage(roots[sources.Kit.ID], registry, "kit", sources.Kit.ID)
	if err != nil {
		return "", err
	}
	if packageDependency(kit.metadata, contract.metadata.Name) != contract.metadata.Version {
		return "", fmt.Errorf("terminal kit contract version differs")
	}
	sidecars := map[string]verifiedLocalRelease{}
	for _, source := range sources.Sidecars {
		release, resolveErr := resolveLocalRelease(store, source, "sidecar", sources.Target)
		if resolveErr != nil {
			return "", resolveErr
		}
		sidecars[source.ID] = release
	}
	plugins := map[string]verifiedLocalRelease{}
	for _, source := range sources.Plugins {
		release, resolveErr := resolveLocalRelease(store, source, "plugin", sources.Target)
		if resolveErr != nil {
			return "", resolveErr
		}
		if err := pluginManifestContract(filepath.Join(release.directory, release.document.Manifest.File), contract.metadata.Version); err != nil {
			return "", err
		}
		if err := validatePluginDependencies(roots[source.ID], contract, kit); err != nil {
			return "", err
		}
		if err := validateRuntimeDependencies(release, sidecars); err != nil {
			return "", err
		}
		plugins[source.ID] = release
	}
	contractArtifact, err := relativeArtifact(options.Output, contract.artifact)
	if err != nil {
		return "", err
	}
	plan := outputPlan{PresentationContract: outputPresentation{
		ID: sources.Contract.ID, Version: contract.metadata.Version, Artifact: contractArtifact,
		SourceRepository: "https://github.com/" + sources.Contract.Repository, SourceCommit: sources.Contract.SourceCommit,
	}}
	for _, source := range sources.Sidecars {
		release := sidecars[source.ID]
		artifact, relativeErr := relativeArtifact(options.Output, filepath.Join(release.directory, release.artifact.File))
		if relativeErr != nil {
			return "", relativeErr
		}
		plan.Components = append(plan.Components, outputComponent{
			Kind: "sidecar", ID: source.ID, Version: release.document.Version, Artifact: artifact,
			Manifest: release.artifact.Manifest, Target: sources.Target,
			SourceRepository: release.document.Source.Repository, SourceCommit: release.document.Source.Commit,
		})
	}
	for _, source := range sources.Plugins {
		release := plugins[source.ID]
		artifact, relativeErr := relativeArtifact(options.Output, filepath.Join(release.directory, release.artifact.File))
		if relativeErr != nil {
			return "", relativeErr
		}
		plan.Components = append(plan.Components, outputComponent{
			Kind: "plugin", ID: source.ID, Version: release.document.Version, Artifact: artifact,
			Manifest: release.artifact.Manifest, SourceRepository: release.document.Source.Repository, SourceCommit: release.document.Source.Commit,
			DependencyCommits: map[string]string{sources.Contract.ID: sources.Contract.SourceCommit, sources.Kit.ID: sources.Kit.SourceCommit},
		})
	}
	body, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return "", err
	}
	body = append(body, '\n')
	return writeImmutablePlan(options.Output, body)
}
