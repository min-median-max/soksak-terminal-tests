package releaseaudit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/min-median-max/soksak-terminal-tests/fleet"
)

type Invocation struct {
	SpecPackage string
	Repository  string
	Tag         string
}

type Failure struct {
	Tag    string `json:"tag"`
	Target string `json:"target"`
	Error  string `json:"error"`
}

type RepositoryReport struct {
	Schema     string    `json:"schema"`
	Repository string    `json:"repository"`
	Releases   int       `json:"releases"`
	Artifacts  int       `json:"artifacts"`
	Failures   []Failure `json:"failures"`
}

type FleetReport struct {
	Schema       string             `json:"schema"`
	Platform     string             `json:"platform"`
	Target       string             `json:"target"`
	Repositories int                `json:"repositories"`
	Releases     int                `json:"releases"`
	Artifacts    int                `json:"artifacts"`
	Results      []RepositoryReport `json:"results"`
}

type Runner func(context.Context, Invocation) ([]byte, error)

func AuditCurrentFleet(ctx context.Context, profile fleet.Profile, specPackage string, run Runner) (FleetReport, error) {
	report := FleetReport{Schema: "soksak-current-sidecar-fleet-audit-v1", Platform: profile.Platform, Target: profile.Target}
	if len(profile.Sidecars) == 0 || specPackage == "" || run == nil {
		return report, fmt.Errorf("current fleet audit requires sidecars, spec package and runner")
	}
	seen := map[string]bool{}
	failures := 0
	for _, sidecar := range profile.Sidecars {
		if sidecar.ID == "" || sidecar.Version == "" || seen[sidecar.ID] {
			return report, fmt.Errorf("current fleet contains an invalid sidecar identity")
		}
		seen[sidecar.ID] = true
		invocation := Invocation{
			SpecPackage: specPackage,
			Repository:  "https://github.com/soksak-ai/" + sidecar.ID,
			Tag:         "v" + sidecar.Version,
		}
		body, runErr := run(ctx, invocation)
		var current RepositoryReport
		decodeErr := json.Unmarshal(body, &current)
		if decodeErr != nil || current.Schema != "soksak-sidecar-release-audit-v1" ||
			current.Repository != invocation.Repository || current.Releases != 1 || current.Artifacts < 1 {
			return report, fmt.Errorf("%s audit result is invalid: run=%v decode=%v", sidecar.ID, runErr, decodeErr)
		}
		report.Repositories++
		report.Releases += current.Releases
		report.Artifacts += current.Artifacts
		report.Results = append(report.Results, current)
		failures += len(current.Failures)
		if runErr != nil && len(current.Failures) == 0 {
			return report, fmt.Errorf("%s audit command failed without a reported artifact failure: %w", sidecar.ID, runErr)
		}
	}
	if failures > 0 {
		return report, fmt.Errorf("current fleet contains %d sidecar release failures", failures)
	}
	return report, nil
}

func WriteEvidence(path string, body []byte) error {
	if path == "" || len(body) == 0 {
		return fmt.Errorf("audit evidence path and bytes are required")
	}
	if previous, err := os.ReadFile(path); err == nil {
		if bytes.Equal(previous, body) {
			return nil
		}
		return fmt.Errorf("audit evidence conflicts with existing bytes: %s", path)
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := file.Write(body); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}
