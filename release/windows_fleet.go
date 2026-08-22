package release

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	platformspec "github.com/soksak-ai/soksak-spec/go/platformspec"
)

type Component struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type Fleet struct {
	Target   string      `json:"target"`
	Plugins  []Component `json:"plugins"`
	Sidecars []Component `json:"sidecars"`
}

type artifact struct {
	Target, URL, SHA256, Format, Manifest string
	Size                                  int64
}

type releaseDocument struct {
	Plugin  *Component `json:"plugin"`
	Sidecar *Component `json:"sidecar"`
	Source  struct {
		Repository string `json:"repository"`
		Commit     string `json:"commit"`
	} `json:"source"`
	Artifacts []artifact `json:"artifacts"`
	Reports   []struct {
		URL, SHA256 string
	} `json:"reports"`
}

func ReadFleet(filename string) (Fleet, error) {
	body, err := os.ReadFile(filename)
	if err != nil {
		return Fleet{}, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	var fleet Fleet
	if err := decoder.Decode(&fleet); err != nil {
		return Fleet{}, err
	}
	if fleet.Target != "x86_64-pc-windows-msvc" || len(fleet.Plugins) == 0 || len(fleet.Sidecars) == 0 {
		return Fleet{}, fmt.Errorf("Windows fleet requires a target, plugins, and sidecars")
	}
	return fleet, nil
}

func Verify(ctx context.Context, client *http.Client, fleet Fleet) error {
	for _, component := range fleet.Plugins {
		if err := verifyComponent(ctx, client, fleet.Target, "plugin", component); err != nil {
			return err
		}
	}
	for _, component := range fleet.Sidecars {
		if err := verifyComponent(ctx, client, fleet.Target, "sidecar", component); err != nil {
			return err
		}
	}
	return nil
}

func verifyComponent(ctx context.Context, client *http.Client, target, kind string, component Component) error {
	repository := "https://github.com/soksak-ai/" + component.ID
	releaseURL := repository + "/releases/download/v" + component.Version + "/release.json"
	body, err := get(ctx, client, releaseURL, 4<<20)
	if err != nil {
		return fmt.Errorf("%s: %w", component.ID, err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	var release releaseDocument
	if err := decoder.Decode(&release); err != nil {
		return fmt.Errorf("%s release.json: %w", component.ID, err)
	}
	identity := release.Plugin
	if kind == "sidecar" {
		identity = release.Sidecar
	}
	if identity == nil || *identity != component || release.Source.Repository != repository || len(release.Source.Commit) != 40 || len(release.Reports) == 0 {
		return fmt.Errorf("%s release identity is invalid", component.ID)
	}
	wantedTarget := "any"
	if kind == "sidecar" {
		wantedTarget = target
	}
	var selected *artifact
	for index := range release.Artifacts {
		if release.Artifacts[index].Target == wantedTarget {
			selected = &release.Artifacts[index]
		}
	}
	if selected == nil || selected.Size <= 0 || len(selected.SHA256) != 64 || selected.Manifest != kind+".json" {
		return fmt.Errorf("%s has no valid %s artifact", component.ID, wantedTarget)
	}
	if !strings.HasPrefix(selected.URL, repository+"/releases/download/v"+component.Version+"/") {
		return fmt.Errorf("%s artifact URL is outside its release", component.ID)
	}
	archive, err := download(ctx, client, selected.URL, selected.Size, selected.SHA256)
	if err != nil {
		return fmt.Errorf("%s: %w", component.ID, err)
	}
	defer os.Remove(archive)
	return inspectArchive(archive, kind, component)
}

func get(ctx context.Context, client *http.Client, address string, limit int64) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", address, response.Status)
	}
	return io.ReadAll(io.LimitReader(response.Body, limit))
}

func download(ctx context.Context, client *http.Client, address string, size int64, digest string) (string, error) {
	parsed, err := url.Parse(address)
	if err != nil {
		return "", err
	}
	file, err := os.CreateTemp("", "soksak-windows-release-*"+path.Ext(parsed.Path))
	if err != nil {
		return "", err
	}
	filename := file.Name()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		file.Close()
		os.Remove(filename)
		return "", err
	}
	response, err := client.Do(request)
	if err != nil {
		file.Close()
		os.Remove(filename)
		return "", err
	}
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(file, hash), response.Body)
	closeErr := file.Close()
	response.Body.Close()
	if response.StatusCode != http.StatusOK || copyErr != nil || closeErr != nil || written != size || hex.EncodeToString(hash.Sum(nil)) != digest {
		os.Remove(filename)
		return "", fmt.Errorf("artifact bytes do not match release.json")
	}
	return filename, nil
}

func inspectArchive(filename, kind string, component Component) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzipReader.Close()
	reader := tar.NewReader(gzipReader)
	manifestName := kind + ".json"
	var manifest []byte
	files := map[string]bool{}
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		rawName := filepath.ToSlash(header.Name)
		name := strings.TrimPrefix(rawName, "./")
		if header.Typeflag == tar.TypeSymlink || header.Typeflag == tar.TypeLink || strings.HasPrefix(rawName, "/") || strings.Contains("/"+name+"/", "/../") {
			return fmt.Errorf("%s archive contains unsafe entry %q", component.ID, header.Name)
		}
		if name == "" && header.Typeflag != tar.TypeDir {
			return fmt.Errorf("%s archive contains unnamed entry", component.ID)
		}
		if header.Typeflag == tar.TypeReg || header.Typeflag == tar.TypeRegA {
			files[name] = true
			if name == manifestName {
				manifest, err = io.ReadAll(io.LimitReader(reader, 1<<20))
				if err != nil {
					return err
				}
			}
		}
	}
	if len(manifest) == 0 {
		return fmt.Errorf("%s archive has no %s", component.ID, manifestName)
	}
	if kind == "plugin" {
		var identity Component
		if err := json.Unmarshal(manifest, &identity); err != nil || identity.ID != component.ID || identity.Version != component.Version {
			return fmt.Errorf("%s plugin manifest identity is invalid", component.ID)
		}
		return nil
	}
	sidecar, err := platformspec.ParseSidecarManifest(manifest)
	if err != nil || sidecar.ID != component.ID || sidecar.Version != component.Version || !strings.HasSuffix(sidecar.Process, ".exe") || !files[sidecar.Process] {
		return fmt.Errorf("%s Windows process does not match its archive: %v", component.ID, err)
	}
	return nil
}
