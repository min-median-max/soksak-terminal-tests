package candidate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

type LocalComposeOptions struct {
	Plan   string
	Store  string
	Output string
}

type localReleaseSelection struct {
	ID            string `json:"id"`
	Version       string `json:"version"`
	ReleaseSHA256 string `json:"releaseSha256"`
}

type localCandidatePlan struct {
	Schema   string                  `json:"schema"`
	Target   string                  `json:"target"`
	Contract localReleaseSelection   `json:"contract"`
	Plugins  []localReleaseSelection `json:"plugins"`
	Sidecars []localReleaseSelection `json:"sidecars"`
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
var candidateIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,127}$`)

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

func readLocalRelease(directory string, selection localReleaseSelection, kind, target string) (verifiedLocalRelease, error) {
	var document localReleaseDocument
	releasePath := filepath.Join(directory, "release.json")
	if err := decodeStrict(releasePath, &document); err != nil {
		return verifiedLocalRelease{}, err
	}
	if document.Kind != kind || document.ID != selection.ID || document.Version != selection.Version ||
		!candidateVersionPattern.MatchString(document.Version) || !commitPattern.MatchString(document.Source.Commit) ||
		document.Source.Repository == "" {
		return verifiedLocalRelease{}, fmt.Errorf("local release identity mismatch: %s", selection.ID)
	}
	if err := verifyLocalFile(directory, document.Manifest); err != nil {
		return verifiedLocalRelease{}, err
	}
	wanted := "any"
	if kind == "sidecar" {
		wanted = target
	}
	var artifact *localReleaseArtifact
	files := map[string]bool{"release.json": true, document.Manifest.File: true}
	for index := range document.Artifacts {
		current := &document.Artifacts[index]
		if err := verifyLocalFile(directory, localFileReference{File: current.File, Size: current.Size, SHA256: current.SHA256}); err != nil {
			return verifiedLocalRelease{}, err
		}
		if current.Manifest != document.Manifest.File || (current.Format != "tgz" && current.Format != "tar.gz") {
			return verifiedLocalRelease{}, fmt.Errorf("local release artifact contract mismatch: %s", selection.ID)
		}
		files[current.File] = true
		if current.Target == wanted {
			if artifact != nil {
				return verifiedLocalRelease{}, fmt.Errorf("local release repeats target %s: %s", wanted, selection.ID)
			}
			artifact = current
		}
	}
	if artifact == nil {
		return verifiedLocalRelease{}, fmt.Errorf("local release omits target %s: %s", wanted, selection.ID)
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
		return verifiedLocalRelease{}, fmt.Errorf("local release inventory mismatch: %s", selection.ID)
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
	if digest != selection.ReleaseSHA256 {
		return verifiedLocalRelease{}, fmt.Errorf("local release document digest mismatch: %s", selection.ID)
	}
	return verifiedLocalRelease{
		directory: directory, document: document, artifact: *artifact,
		releaseBytes: fileReference{Path: releasePath, Size: size, SHA256: digest},
	}, nil
}

func resolveLocalRelease(store string, selected localReleaseSelection, kind, target string) (verifiedLocalRelease, error) {
	if !candidateIDPattern.MatchString(selected.ID) || !candidateVersionPattern.MatchString(selected.Version) ||
		!digestPattern.MatchString(selected.ReleaseSHA256) {
		return verifiedLocalRelease{}, fmt.Errorf("local release selection is invalid: %s", selected.ID)
	}
	directory := filepath.Join(store, kind+"s", selected.ID, selected.Version)
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return verifiedLocalRelease{}, fmt.Errorf("local release directory is invalid: %s", selected.ID)
	}
	return readLocalRelease(directory, selected, kind, target)
}
