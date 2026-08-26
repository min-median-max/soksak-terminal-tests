package candidate

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

type packageMetadata struct {
	Name                 string            `json:"name"`
	Version              string            `json:"version"`
	Exports              json.RawMessage   `json:"exports"`
	Dependencies         map[string]string `json:"dependencies"`
	DevDependencies      map[string]string `json:"devDependencies"`
	PeerDependencies     map[string]string `json:"peerDependencies"`
	OptionalDependencies map[string]string `json:"optionalDependencies"`
}

type verifiedPackage struct {
	artifact string
	metadata packageMetadata
}

func commandOutput(directory, name string, args ...string) (string, error) {
	command := exec.Command(name, args...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output)), nil
}

func canonicalGitHubRepository(remote string) string {
	remote = strings.TrimSpace(strings.TrimSuffix(remote, ".git"))
	if strings.HasPrefix(remote, "git@github.com:") {
		remote = "https://github.com/" + strings.TrimPrefix(remote, "git@github.com:")
	}
	if strings.HasPrefix(remote, "ssh://git@github.com/") {
		remote = "https://github.com/" + strings.TrimPrefix(remote, "ssh://git@github.com/")
	}
	return remote
}

func discoverSourceRoots(workspace string, plan sourcePlan) (map[string]string, error) {
	if !filepath.IsAbs(workspace) {
		return nil, fmt.Errorf("source workspace must be absolute")
	}
	root, err := filepath.EvalSymlinks(workspace)
	if err != nil || root != filepath.Clean(workspace) {
		return nil, fmt.Errorf("source workspace must not use a symbolic path")
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("source workspace must be a regular directory")
	}
	wanted := expectedSources(plan)
	byRepository := make(map[string]sourceComponent, len(wanted))
	for _, source := range wanted {
		if previous, exists := byRepository[source.Repository]; exists {
			return nil, fmt.Errorf("source repository is repeated: %s and %s", previous.ID, source.ID)
		}
		byRepository[source.Repository] = source
	}
	found := map[string]string{}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.IsDir() {
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		depth := 0
		if relative != "." {
			depth = len(strings.Split(relative, string(filepath.Separator)))
		}
		if depth > 4 {
			return filepath.SkipDir
		}
		if depth > 0 {
			switch entry.Name() {
			case "node_modules", "target", ".pnpm-store", ".cache", "backup", "evidence":
				return filepath.SkipDir
			}
		}
		marker := filepath.Join(path, ".git")
		markerInfo, markerErr := os.Lstat(marker)
		if markerErr != nil {
			if os.IsNotExist(markerErr) {
				return nil
			}
			return markerErr
		}
		if markerInfo.Mode()&os.ModeSymlink != 0 || (!markerInfo.IsDir() && !markerInfo.Mode().IsRegular()) {
			return fmt.Errorf("Git marker is not regular: %s", marker)
		}
		remote, runErr := commandOutput(path, "git", "remote", "get-url", "origin")
		if runErr != nil {
			return filepath.SkipDir
		}
		canonical := canonicalGitHubRepository(remote)
		for repository, source := range byRepository {
			if canonical != "https://github.com/"+repository {
				continue
			}
			if found[source.ID] != "" {
				return fmt.Errorf("source repository is discovered more than once: %s", source.ID)
			}
			status, statusErr := commandOutput(path, "git", "status", "--porcelain")
			if statusErr != nil {
				return statusErr
			}
			if status != "" {
				return fmt.Errorf("source repository is dirty: %s", source.ID)
			}
			commit, commitErr := commandOutput(path, "git", "rev-parse", "HEAD")
			if commitErr != nil {
				return commitErr
			}
			if commit != source.SourceCommit {
				return fmt.Errorf("source commit differs: %s=%s want %s", source.ID, commit, source.SourceCommit)
			}
			found[source.ID] = path
			break
		}
		return filepath.SkipDir
	})
	if err != nil {
		return nil, err
	}
	if len(found) != len(wanted) {
		missing := []string{}
		for id := range wanted {
			if found[id] == "" {
				missing = append(missing, id)
			}
		}
		sort.Strings(missing)
		return nil, fmt.Errorf("source repositories are missing: %s", strings.Join(missing, ","))
	}
	return found, nil
}

func readPackageArchive(path string) (map[string][]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("package artifact is not a regular file: %s", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()
	entries := map[string][]byte{}
	tarReader := tar.NewReader(gzipReader)
	for {
		header, nextErr := tarReader.Next()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			return nil, nextErr
		}
		if header.Typeflag == tar.TypeDir {
			continue
		}
		if header.Typeflag != tar.TypeReg || header.Size < 0 || header.Size > 64<<20 || !strings.HasPrefix(header.Name, "package/") {
			return nil, fmt.Errorf("package archive contains a non-regular entry: %s", header.Name)
		}
		name := strings.TrimPrefix(header.Name, "package/")
		if name == "" || filepath.IsAbs(name) || strings.Contains(name, "..") || entries[name] != nil {
			return nil, fmt.Errorf("package archive entry is invalid: %s", header.Name)
		}
		body, readErr := io.ReadAll(io.LimitReader(tarReader, header.Size))
		if readErr != nil {
			return nil, readErr
		}
		entries[name] = body
	}
	return entries, nil
}

func comparePackageFile(root string, archive map[string][]byte, name string) error {
	source, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(source, archive[name]) {
		return fmt.Errorf("package artifact differs from source: %s", name)
	}
	return nil
}

func verifyPackageSourceTree(root string, archive map[string][]byte, kind string) error {
	required := []string{kind + ".json", "src/index.ts"}
	if kind == "contract" {
		required = append(required, "presentation.json")
	} else {
		required = append(required, "dist/index.js")
	}
	for _, name := range required {
		if err := comparePackageFile(root, archive, name); err != nil {
			return err
		}
	}
	return filepath.WalkDir(filepath.Join(root, "src"), func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("package source contains a symbolic link: %s", path)
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".ts") || strings.HasSuffix(entry.Name(), ".test.ts") {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		return comparePackageFile(root, archive, filepath.ToSlash(relative))
	})
}

func verifyRegistryPackage(sourceRoot, registry, kind, id string) (verifiedPackage, error) {
	manifestBody, err := os.ReadFile(filepath.Join(sourceRoot, kind+".json"))
	if err != nil {
		return verifiedPackage{}, err
	}
	var identity struct {
		ID      string `json:"id"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(manifestBody, &identity); err != nil || identity.ID != id || !candidateVersionPattern.MatchString(identity.Version) {
		return verifiedPackage{}, fmt.Errorf("%s package identity is invalid", kind)
	}
	artifact := filepath.Join(registry, "@soksak", id, id+"-"+identity.Version+".tgz")
	archive, err := readPackageArchive(artifact)
	if err != nil {
		return verifiedPackage{}, err
	}
	var sourceMetadata, packedMetadata packageMetadata
	sourcePackage, err := os.ReadFile(filepath.Join(sourceRoot, "package.json"))
	if err != nil {
		return verifiedPackage{}, err
	}
	if err := json.Unmarshal(sourcePackage, &sourceMetadata); err != nil {
		return verifiedPackage{}, err
	}
	if err := json.Unmarshal(archive["package.json"], &packedMetadata); err != nil {
		return verifiedPackage{}, fmt.Errorf("package artifact metadata is invalid: %s", id)
	}
	if !reflect.DeepEqual(sourceMetadata, packedMetadata) || sourceMetadata.Name != "@soksak/"+id || sourceMetadata.Version != identity.Version {
		return verifiedPackage{}, fmt.Errorf("package artifact metadata differs from source: %s", id)
	}
	if err := verifyPackageSourceTree(sourceRoot, archive, kind); err != nil {
		return verifiedPackage{}, err
	}
	return verifiedPackage{artifact: artifact, metadata: sourceMetadata}, nil
}
