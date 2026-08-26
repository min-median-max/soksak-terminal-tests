package candidate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

type LocalComposeOptions struct {
	SourcePlan string
	Store      string
	Registry   string
	Workspace  string
	Output     string
}

type localFileReference struct {
	File   string `json:"file"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type localReleaseArtifact struct {
	Target   string `json:"target"`
	File     string `json:"file"`
	Size     int64  `json:"size"`
	SHA256   string `json:"sha256"`
	Format   string `json:"format"`
	Manifest string `json:"manifest"`
}

type localRuntimeReference struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Size    int64  `json:"size"`
	SHA256  string `json:"sha256"`
}

type localRuntimeDependencies struct {
	Sidecars []localRuntimeReference `json:"sidecars"`
}

type localReleaseDocument struct {
	Kind                string                   `json:"kind"`
	ID                  string                   `json:"id"`
	Version             string                   `json:"version"`
	Manifest            localFileReference       `json:"manifest"`
	Source              candidateSource          `json:"source"`
	Artifacts           []localReleaseArtifact   `json:"artifacts"`
	RuntimeDependencies localRuntimeDependencies `json:"runtimeDependencies,omitempty"`
	Evidence            []localFileReference     `json:"evidence"`
}

type verifiedLocalRelease struct {
	directory    string
	document     localReleaseDocument
	artifact     localReleaseArtifact
	releaseBytes fileReference
}

var candidateVersionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)

func regularDirectory(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("directory must be absolute: %s", path)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || resolved != filepath.Clean(path) {
		return "", fmt.Errorf("directory must not use a symbolic path: %s", path)
	}
	info, err := os.Lstat(resolved)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("regular directory required: %s", path)
	}
	return resolved, nil
}

func verifyLocalFile(directory string, reference localFileReference) error {
	if filepath.Base(reference.File) != reference.File || reference.Size <= 0 || !digestPattern.MatchString(reference.SHA256) {
		return fmt.Errorf("local release file reference is invalid: %s", reference.File)
	}
	size, digest, err := fileDigest(filepath.Join(directory, reference.File))
	if err != nil || size != reference.Size || digest != reference.SHA256 {
		return fmt.Errorf("local release file digest mismatch: %s", reference.File)
	}
	return nil
}

func readLocalRelease(directory string, expected sourceComponent, kind, target string) (verifiedLocalRelease, error) {
	var document localReleaseDocument
	releasePath := filepath.Join(directory, "release.json")
	if err := decodeStrict(releasePath, &document); err != nil {
		return verifiedLocalRelease{}, err
	}
	if document.Kind != kind || document.ID != expected.ID || !candidateVersionPattern.MatchString(document.Version) ||
		document.Source.Repository != "https://github.com/"+expected.Repository || document.Source.Commit != expected.SourceCommit {
		return verifiedLocalRelease{}, fmt.Errorf("local release identity mismatch: %s", expected.ID)
	}
	if err := verifyLocalFile(directory, document.Manifest); err != nil {
		return verifiedLocalRelease{}, err
	}
	wanted := "any"
	if kind == "sidecar" {
		wanted = target
	}
	var selected *localReleaseArtifact
	files := map[string]bool{"release.json": true, document.Manifest.File: true}
	for index := range document.Artifacts {
		artifact := &document.Artifacts[index]
		if err := verifyLocalFile(directory, localFileReference{File: artifact.File, Size: artifact.Size, SHA256: artifact.SHA256}); err != nil {
			return verifiedLocalRelease{}, err
		}
		if artifact.Manifest != document.Manifest.File || (artifact.Format != "tgz" && artifact.Format != "tar.gz") {
			return verifiedLocalRelease{}, fmt.Errorf("local release artifact contract mismatch: %s", expected.ID)
		}
		files[artifact.File] = true
		if artifact.Target == wanted {
			if selected != nil {
				return verifiedLocalRelease{}, fmt.Errorf("local release repeats target %s: %s", wanted, expected.ID)
			}
			selected = artifact
		}
	}
	if selected == nil {
		return verifiedLocalRelease{}, fmt.Errorf("local release omits target %s: %s", wanted, expected.ID)
	}
	for _, evidence := range document.Evidence {
		if err := verifyLocalFile(directory, evidence); err != nil {
			return verifiedLocalRelease{}, err
		}
		files[evidence.File] = true
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return verifiedLocalRelease{}, err
	}
	if len(entries) != len(files) {
		return verifiedLocalRelease{}, fmt.Errorf("local release inventory mismatch: %s", expected.ID)
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !files[entry.Name()] {
			return verifiedLocalRelease{}, fmt.Errorf("local release contains an undeclared entry: %s", entry.Name())
		}
	}
	size, digest, err := fileDigest(releasePath)
	if err != nil {
		return verifiedLocalRelease{}, err
	}
	return verifiedLocalRelease{
		directory: directory, document: document, artifact: *selected,
		releaseBytes: fileReference{Path: releasePath, Size: size, SHA256: digest},
	}, nil
}

func resolveLocalRelease(store string, expected sourceComponent, kind, target string) (verifiedLocalRelease, error) {
	root := filepath.Join(store, kind+"s", expected.ID)
	entries, err := os.ReadDir(root)
	if err != nil {
		return verifiedLocalRelease{}, fmt.Errorf("local release is missing: %s: %w", expected.ID, err)
	}
	matches := []verifiedLocalRelease{}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return verifiedLocalRelease{}, fmt.Errorf("local release version is not a regular directory: %s", entry.Name())
		}
		candidate, readErr := readLocalRelease(filepath.Join(root, entry.Name()), expected, kind, target)
		if readErr == nil {
			matches = append(matches, candidate)
		}
	}
	if len(matches) != 1 {
		return verifiedLocalRelease{}, fmt.Errorf("local release match count=%d: %s", len(matches), expected.ID)
	}
	return matches[0], nil
}
