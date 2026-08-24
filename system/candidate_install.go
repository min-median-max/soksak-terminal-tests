package system

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type CandidateComponent struct {
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

type CandidatePlan struct {
	Components           []CandidateComponent  `json:"components"`
	PresentationContract CandidatePresentation `json:"presentationContract"`
}

type CandidateBudgets struct {
	RenderMs          float64 `json:"renderMs"`
	InputToPtyWriteMs float64 `json:"inputToPtyWriteMs"`
}

type CandidatePresentation struct {
	ID               string                    `json:"id"`
	Version          string                    `json:"version"`
	Artifact         string                    `json:"artifact"`
	SourceRepository string                    `json:"sourceRepository"`
	SourceCommit     string                    `json:"sourceCommit"`
	Data             CandidatePresentationData `json:"-"`
}

type CandidatePresentationData struct {
	Version int `json:"version"`
	ANSI    struct {
		Base      []string `json:"base"`
		Cube      []int    `json:"cube"`
		Grayscale struct {
			Start int `json:"start"`
			Step  int `json:"step"`
			Count int `json:"count"`
		} `json:"grayscale"`
	} `json:"ansi"`
	Budgets CandidateBudgets `json:"budgets"`
	Theme   struct {
		Tokens struct {
			Foreground          string `json:"foreground"`
			Background          string `json:"background"`
			Cursor              string `json:"cursor"`
			CursorAccent        string `json:"cursorAccent"`
			SelectionBackground string `json:"selectionBackground"`
		} `json:"tokens"`
		Properties struct {
			Cursor              string `json:"cursor"`
			CursorAccent        string `json:"cursorAccent"`
			SelectionBackground string `json:"selectionBackground"`
			ANSIPrefix          string `json:"ansiPrefix"`
		} `json:"properties"`
	} `json:"theme"`
}

type preparedCandidate struct {
	CandidateComponent
	path   string
	size   int64
	digest string
}

var candidateIdentity = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,127}$`)
var candidateVersion = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)
var candidateCommit = regexp.MustCompile(`^[0-9a-f]{40}$`)

func InstallCandidateFleet(planPath string, cli commandCaller) error {
	plan, root, err := readCandidatePlan(planPath)
	if err != nil {
		return err
	}
	prepared := make([]preparedCandidate, 0, len(plan.Components))
	var transactionRoot *CandidateComponent
	for index := range plan.Components {
		component := plan.Components[index]
		candidate, err := prepareCandidate(root, component)
		if err != nil {
			return err
		}
		prepared = append(prepared, candidate)
		if transactionRoot == nil && component.Kind == "plugin" {
			copy := component
			transactionRoot = &copy
		}
	}
	if transactionRoot == nil {
		return fmt.Errorf("candidate plan has no plugin transaction root")
	}
	revision, err := candidateEnvironmentRevision(cli)
	if err != nil {
		return err
	}
	server, baseURL, err := startCandidateServer(root)
	if err != nil {
		return err
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	started, err := cli.Call("artifact_install_begin", map[string]any{
		"registryId": "candidate",
		"root":       map[string]any{"kind": transactionRoot.Kind, "id": transactionRoot.ID, "version": transactionRoot.Version},
	})
	if err != nil {
		return fmt.Errorf("begin candidate transaction: %w", err)
	}
	transactionID, _ := started["transactionId"].(string)
	if transactionID == "" {
		return fmt.Errorf("candidate transaction returned no id")
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = cli.Call("artifact_install_rollback", map[string]any{"transactionId": transactionID})
		}
	}()

	components := make([]map[string]any, 0, len(prepared))
	for _, candidate := range prepared {
		artifactURL := baseURL + "/" + escapeCandidatePath(candidate.Artifact)
		identity := map[string]any{"kind": candidate.Kind, "id": candidate.ID, "version": candidate.Version}
		staged, err := cli.Call("artifact_install_stage", map[string]any{
			"transactionId": transactionID, "registryId": "candidate", "identity": identity,
			"artifact": map[string]any{
				"url": artifactURL, "size": candidate.size, "sha256": candidate.digest,
				"format": "tgz", "manifest": candidate.Manifest, "entrypoints": []string{candidate.Manifest},
			},
		})
		if err != nil {
			return fmt.Errorf("stage candidate %s: %w", candidate.ID, err)
		}
		handle, _ := staged["handle"].(string)
		if handle == "" || staged["sha256"] != candidate.digest || numericInt64(staged["size"]) != candidate.size {
			return fmt.Errorf("candidate %s returned invalid staging evidence: %+v", candidate.ID, staged)
		}
		raw, err := cli.CallValue("artifact_install_read_utf8", map[string]any{
			"transactionId": transactionID, "handle": handle, "path": candidate.Manifest,
		})
		if err != nil {
			return fmt.Errorf("read candidate manifest %s: %w", candidate.ID, err)
		}
		text, ok := raw.(string)
		if !ok || !candidateManifestIdentity([]byte(text), candidate.ID, candidate.Version) {
			return fmt.Errorf("candidate manifest identity mismatch: %s", candidate.ID)
		}
		component := map[string]any{
			"kind": candidate.Kind, "id": candidate.ID, "version": candidate.Version,
			"registryId": "candidate", "sourceRepository": candidate.SourceRepository,
			"sourceCommit": candidate.SourceCommit, "artifactUrl": artifactURL,
			"artifactSha256": candidate.digest, "stagedHandle": handle,
		}
		if candidate.Kind == "sidecar" {
			component["target"] = candidate.Target
		}
		components = append(components, component)
	}
	if _, err := cli.Call("artifact_install_commit", map[string]any{
		"transactionId": transactionID, "expectedRevision": revision, "components": components,
	}); err != nil {
		return fmt.Errorf("commit candidate transaction: %w", err)
	}
	committed = true
	return nil
}

func readCandidatePlan(planPath string) (CandidatePlan, string, error) {
	if !filepath.IsAbs(planPath) {
		return CandidatePlan{}, "", fmt.Errorf("candidate plan path must be absolute: %s", planPath)
	}
	body, err := os.ReadFile(planPath)
	if err != nil {
		return CandidatePlan{}, "", err
	}
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	var plan CandidatePlan
	if err := decoder.Decode(&plan); err != nil {
		return CandidatePlan{}, "", err
	}
	if len(plan.Components) == 0 {
		return CandidatePlan{}, "", fmt.Errorf("candidate plan has no components")
	}
	root := filepath.Dir(planPath)
	presentation, err := readCandidatePresentation(root, plan.PresentationContract)
	if err != nil {
		return CandidatePlan{}, "", err
	}
	plan.PresentationContract.Data = presentation
	return plan, root, nil
}

var candidateColour = regexp.MustCompile(`^#[0-9a-f]{6}$`)
var candidateCSSProperty = regexp.MustCompile(`^--[a-z][a-z0-9-]*$`)

func completeCandidateTheme(data CandidatePresentationData) bool {
	values := []string{
		data.Theme.Tokens.Foreground, data.Theme.Tokens.Background, data.Theme.Tokens.Cursor,
		data.Theme.Tokens.CursorAccent, data.Theme.Tokens.SelectionBackground,
		data.Theme.Properties.Cursor, data.Theme.Properties.CursorAccent,
		data.Theme.Properties.SelectionBackground, data.Theme.Properties.ANSIPrefix,
	}
	for _, value := range values {
		if !candidateCSSProperty.MatchString(value) {
			return false
		}
	}
	return strings.HasSuffix(data.Theme.Properties.ANSIPrefix, "-")
}

func readCandidatePresentation(root string, reference CandidatePresentation) (CandidatePresentationData, error) {
	if reference.ID != "soksak-contract-plugin-terminal" || !candidateVersion.MatchString(reference.Version) ||
		!candidateCommit.MatchString(reference.SourceCommit) ||
		reference.SourceRepository != "https://github.com/soksak-ai/soksak-contract-plugin-terminal" {
		return CandidatePresentationData{}, fmt.Errorf("invalid candidate presentation contract: %+v", reference)
	}
	path, err := candidateArtifactPath(root, reference.Artifact)
	if err != nil {
		return CandidatePresentationData{}, err
	}
	file, err := os.Open(path)
	if err != nil {
		return CandidatePresentationData{}, err
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return CandidatePresentationData{}, fmt.Errorf("presentation contract is not gzip: %w", err)
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)
	var identityBody, presentationBody []byte
	for {
		header, nextErr := tarReader.Next()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			return CandidatePresentationData{}, nextErr
		}
		if header.Typeflag != tar.TypeReg || header.Size < 0 || header.Size > 1<<20 {
			continue
		}
		body, readErr := io.ReadAll(io.LimitReader(tarReader, header.Size))
		if readErr != nil {
			return CandidatePresentationData{}, readErr
		}
		switch header.Name {
		case "package/contract.json":
			if identityBody != nil {
				return CandidatePresentationData{}, fmt.Errorf("presentation contract repeats contract.json")
			}
			identityBody = body
		case "package/presentation.json":
			if presentationBody != nil {
				return CandidatePresentationData{}, fmt.Errorf("presentation contract repeats presentation.json")
			}
			presentationBody = body
		}
	}
	if !candidateManifestIdentity(identityBody, reference.ID, reference.Version) {
		return CandidatePresentationData{}, fmt.Errorf("presentation contract identity mismatch")
	}
	decoder := json.NewDecoder(strings.NewReader(string(presentationBody)))
	decoder.DisallowUnknownFields()
	var data CandidatePresentationData
	if err := decoder.Decode(&data); err != nil {
		return CandidatePresentationData{}, fmt.Errorf("invalid presentation contract data: %w", err)
	}
	if data.Version != 2 || len(data.ANSI.Base) != 16 ||
		data.Budgets.RenderMs <= 0 || data.Budgets.InputToPtyWriteMs <= 0 ||
		len(data.ANSI.Cube) != 6 || data.ANSI.Grayscale.Count != 24 || !completeCandidateTheme(data) {
		return CandidatePresentationData{}, fmt.Errorf("presentation contract data is incomplete")
	}
	for _, colour := range data.ANSI.Base {
		if !candidateColour.MatchString(colour) {
			return CandidatePresentationData{}, fmt.Errorf("presentation contract has invalid colour: %s", colour)
		}
	}
	return data, nil
}

func prepareCandidate(root string, component CandidateComponent) (preparedCandidate, error) {
	if (component.Kind != "plugin" && component.Kind != "sidecar") ||
		!candidateIdentity.MatchString(component.ID) || !candidateVersion.MatchString(component.Version) ||
		!candidateCommit.MatchString(component.SourceCommit) ||
		(component.Manifest != "plugin.json" && component.Manifest != "sidecar.json") ||
		!strings.HasPrefix(component.SourceRepository, "https://github.com/") {
		return preparedCandidate{}, fmt.Errorf("invalid candidate component: %+v", component)
	}
	if component.Kind == "sidecar" && component.Target == "" {
		return preparedCandidate{}, fmt.Errorf("candidate sidecar %s has no target", component.ID)
	}
	for id, commit := range component.DependencyCommits {
		if !candidateIdentity.MatchString(id) || !candidateCommit.MatchString(commit) {
			return preparedCandidate{}, fmt.Errorf("candidate dependency commit is invalid: %s=%s", id, commit)
		}
	}
	path, err := candidateArtifactPath(root, component.Artifact)
	if err != nil {
		return preparedCandidate{}, err
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return preparedCandidate{}, err
	}
	digest := sha256.Sum256(body)
	return preparedCandidate{
		CandidateComponent: component, path: path, size: int64(len(body)), digest: hex.EncodeToString(digest[:]),
	}, nil
}

func candidateArtifactPath(root, artifact string) (string, error) {
	clean := filepath.Clean(artifact)
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("candidate artifact path escapes the plan: %s", artifact)
	}
	path := filepath.Join(root, clean)
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("candidate artifact is not a regular file: %s", path)
	}
	return path, nil
}

func startCandidateServer(root string) (*http.Server, string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", err
	}
	server := &http.Server{Handler: http.FileServer(http.Dir(root)), ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = server.Serve(listener) }()
	return server, "http://" + listener.Addr().String(), nil
}

func escapeCandidatePath(value string) string {
	parts := strings.Split(filepath.ToSlash(value), "/")
	for index := range parts {
		parts[index] = url.PathEscape(parts[index])
	}
	return strings.Join(parts, "/")
}

func candidateEnvironmentRevision(cli commandCaller) (uint64, error) {
	value, err := cli.CallValue("environment_get", map[string]any{})
	if err != nil {
		return 0, err
	}
	document, ok := value.(map[string]any)
	if !ok {
		return 0, fmt.Errorf("environment_get returned %T", value)
	}
	revision := numericInt64(document["revision"])
	if revision < 0 {
		return 0, fmt.Errorf("environment revision is invalid: %v", document["revision"])
	}
	return uint64(revision), nil
}

func numericInt64(value any) int64 {
	switch number := value.(type) {
	case float64:
		if number >= 0 && number == float64(int64(number)) {
			return int64(number)
		}
	case int64:
		return number
	case int:
		return int64(number)
	case uint64:
		if number <= uint64(^uint64(0)>>1) {
			return int64(number)
		}
	}
	return -1
}

func candidateManifestIdentity(body []byte, id, version string) bool {
	var identity struct {
		ID      string `json:"id"`
		Version string `json:"version"`
	}
	return json.Unmarshal(body, &identity) == nil && identity.ID == id && identity.Version == version
}
