package development

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/min-median-max/soksak-terminal-tests/fleet"
	platformspec "github.com/soksak-ai/soksak-spec/go/platformspec"
)

const version = "0.0.1"

type ArtifactInput struct {
	Path           string `json:"path"`
	Repository     string `json:"repository"`
	Commit         string `json:"commit"`
	ArtifactSHA256 string `json:"artifactSha256"`
	Target         string `json:"target,omitempty"`
}

type Input struct {
	Platform string                   `json:"platform"`
	Home     string                   `json:"home"`
	Plugins  map[string]ArtifactInput `json:"plugins"`
	Sidecars map[string]ArtifactInput `json:"sidecars"`
}

type StatePaths struct {
	Settings  string `json:"settings"`
	Installed string `json:"installed"`
}

var repositoryPattern = regexp.MustCompile(`^https://github\.com/[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
var commitPattern = regexp.MustCompile(`^[a-f0-9]{40}$`)
var digestPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

func DecodeInput(reader io.Reader) (Input, error) {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	var input Input
	if err := decoder.Decode(&input); err != nil {
		return Input{}, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return Input{}, fmt.Errorf("input has trailing data")
	}
	return input, nil
}

func PrepareState(input Input) (StatePaths, error) {
	profile, err := fleet.ForPlatform(input.Platform)
	if err != nil {
		return StatePaths{}, err
	}
	if err := validateDirectory(input.Home, false); err != nil {
		return StatePaths{}, fmt.Errorf("home: %w", err)
	}
	settings := platformspec.EmptySettings()
	installed := platformspec.EmptyInstalled()
	for _, provider := range profile.Plugins {
		artifact, ok := input.Plugins[provider.ID]
		if !ok {
			return StatePaths{}, fmt.Errorf("missing plugin artifact: %s", provider.ID)
		}
		digest, componentVersion, err := validatePlugin(artifact, provider.ID)
		if err != nil {
			return StatePaths{}, err
		}
		settings.Plugins[provider.ID] = platformspec.PluginPreference{
			Enabled: true, Development: &platformspec.Development{Path: artifact.Path},
			Providers: map[string]string{"pty": "soksak-sidecar-pty", provider.Requirement: provider.Sidecar},
		}
		installed.Plugins[provider.ID] = installedComponent(artifact, digest, componentVersion, false)
	}
	if len(input.Plugins) != len(profile.Plugins) {
		return StatePaths{}, fmt.Errorf("plugins must contain exactly %d entries", len(profile.Plugins))
	}
	sidecarIDs := append([]string{"soksak-sidecar-pty"}, profile.RecoverySidecars...)
	for _, id := range sidecarIDs {
		artifact, ok := input.Sidecars[id]
		if !ok {
			return StatePaths{}, fmt.Errorf("missing sidecar artifact: %s", id)
		}
		if artifact.Target != profile.Target {
			return StatePaths{}, fmt.Errorf("%s target is %s, want %s", id, artifact.Target, profile.Target)
		}
		digest, componentVersion, err := validateSidecar(artifact, id)
		if err != nil {
			return StatePaths{}, err
		}
		settings.Sidecars[id] = platformspec.ComponentPreference{Development: &platformspec.Development{Path: artifact.Path}}
		installed.Sidecars[id] = installedComponent(artifact, digest, componentVersion, true)
	}
	if len(input.Sidecars) != len(sidecarIDs) {
		return StatePaths{}, fmt.Errorf("sidecars must contain exactly %d entries", len(sidecarIDs))
	}
	if err := platformspec.ValidateSettings(settings); err != nil {
		return StatePaths{}, err
	}
	if err := platformspec.ValidateInstalled(installed); err != nil {
		return StatePaths{}, err
	}
	if err := os.MkdirAll(input.Home, 0o700); err != nil {
		return StatePaths{}, err
	}
	installedPath, err := prepareDocument(input.Home, platformspec.InstalledFile, installed)
	if err != nil {
		return StatePaths{}, err
	}
	settingsPath, err := prepareDocument(input.Home, platformspec.SettingsFile, settings)
	if err != nil {
		_ = os.Remove(installedPath)
		return StatePaths{}, err
	}
	finalInstalled := filepath.Join(input.Home, platformspec.InstalledFile)
	finalSettings := filepath.Join(input.Home, platformspec.SettingsFile)
	if err := replaceFile(installedPath, finalInstalled); err != nil {
		_ = os.Remove(settingsPath)
		return StatePaths{}, err
	}
	if err := replaceFile(settingsPath, finalSettings); err != nil {
		return StatePaths{}, err
	}
	return StatePaths{Settings: finalSettings, Installed: finalInstalled}, nil
}

func installedComponent(input ArtifactInput, manifestDigest, componentVersion string, sidecar bool) platformspec.InstalledComponent {
	target := ""
	if sidecar {
		target = input.Target
	}
	return platformspec.InstalledComponent{
		Version: componentVersion, Path: input.Path, RegistryID: "system-test", Repository: input.Repository,
		SourceCommit: input.Commit, ManifestSHA256: manifestDigest, ArtifactSHA256: input.ArtifactSHA256, Target: target,
	}
}

func validatePlugin(input ArtifactInput, id string) (string, string, error) {
	body, digest, err := readManifest(input, "plugin.json")
	if err != nil {
		return "", "", fmt.Errorf("%s: %w", id, err)
	}
	var identity struct {
		ID      string `json:"id"`
		Version string `json:"version"`
	}
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&identity); err != nil {
		return "", "", err
	}
	validVersion, versionErr := platformspec.DependencyRequirementSatisfied(identity.Version, identity.Version)
	if identity.ID != id || versionErr != nil || !validVersion {
		return "", "", fmt.Errorf("%s manifest identity is %s@%s", id, identity.ID, identity.Version)
	}
	return digest, identity.Version, nil
}

func validateSidecar(input ArtifactInput, id string) (string, string, error) {
	body, digest, err := readManifest(input, "sidecar.json")
	if err != nil {
		return "", "", fmt.Errorf("%s: %w", id, err)
	}
	manifest, err := platformspec.ParseSidecarManifest(body)
	if err != nil || manifest.ID != id {
		return "", "", fmt.Errorf("%s has an invalid sidecar manifest: %v", id, err)
	}
	process, err := os.Lstat(filepath.Join(input.Path, filepath.FromSlash(manifest.Process)))
	if err != nil || !process.Mode().IsRegular() {
		return "", "", fmt.Errorf("%s process is not a regular file: %v", id, err)
	}
	if input.Target == "" {
		return "", "", fmt.Errorf("%s target is required", id)
	}
	return digest, manifest.Version, nil
}

func readManifest(input ArtifactInput, name string) ([]byte, string, error) {
	if err := validateArtifact(input); err != nil {
		return nil, "", err
	}
	path := filepath.Join(input.Path, name)
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, "", fmt.Errorf("manifest is not a regular file: %v", err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	digest := sha256.Sum256(body)
	return body, hex.EncodeToString(digest[:]), nil
}

func validateArtifact(input ArtifactInput) error {
	if err := validateDirectory(input.Path, true); err != nil {
		return err
	}
	if !repositoryPattern.MatchString(input.Repository) || !commitPattern.MatchString(input.Commit) || !digestPattern.MatchString(input.ArtifactSHA256) {
		return fmt.Errorf("artifact provenance is invalid")
	}
	return nil
}

func prepareDocument(home, name string, value any) (string, error) {
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", err
	}
	temporary, err := os.CreateTemp(home, name+"-*.next")
	if err != nil {
		return "", err
	}
	path := temporary.Name()
	if err := temporary.Chmod(0o600); err == nil {
		_, err = temporary.Write(append(body, '\n'))
	}
	if err == nil {
		err = temporary.Sync()
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(path)
		return "", err
	}
	return path, nil
}

func validateDirectory(path string, mustExist bool) error {
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return fmt.Errorf("path must be absolute and clean")
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) && !mustExist {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("path must be a directory without a symbolic link")
	}
	return nil
}
