package releaseaudit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/min-median-max/soksak-terminal-tests/fleet"
)

func TestCurrentFleetAuditAddressesEveryExactSidecarRelease(t *testing.T) {
	profile, err := fleet.ForTarget("darwin", "aarch64-apple-darwin")
	if err != nil {
		t.Fatal(err)
	}
	var calls []Invocation
	run := func(_ context.Context, invocation Invocation) ([]byte, error) {
		calls = append(calls, invocation)
		return json.Marshal(RepositoryReport{
			Schema: "soksak-sidecar-release-audit-v1", Repository: invocation.Repository,
			Releases: 1, Artifacts: 5,
		})
	}
	report, err := AuditCurrentFleet(context.Background(), profile, "/spec-package", run)
	if err != nil {
		t.Fatal(err)
	}
	if report.Repositories != len(profile.Sidecars) || report.Releases != len(profile.Sidecars) || len(calls) != len(profile.Sidecars) {
		t.Fatalf("report=%+v calls=%d sidecars=%d", report, len(calls), len(profile.Sidecars))
	}
	for index, sidecar := range profile.Sidecars {
		if calls[index].Repository != "https://github.com/soksak-ai/"+sidecar.ID || calls[index].Tag != "v"+sidecar.Version ||
			calls[index].SpecPackage != "/spec-package" {
			t.Errorf("call %d=%+v sidecar=%+v", index, calls[index], sidecar)
		}
	}
}

func TestCurrentFleetAuditRejectsAnyRepositoryFailure(t *testing.T) {
	profile, err := fleet.ForTarget("darwin", "aarch64-apple-darwin")
	if err != nil {
		t.Fatal(err)
	}
	run := func(_ context.Context, invocation Invocation) ([]byte, error) {
		return json.Marshal(RepositoryReport{
			Schema: "soksak-sidecar-release-audit-v1", Repository: invocation.Repository,
			Releases: 1, Artifacts: 1,
			Failures: []Failure{{Tag: invocation.Tag, Target: "aarch64-apple-darwin", Error: "wrong header"}},
		})
	}
	if _, err := AuditCurrentFleet(context.Background(), profile, "/spec-package", run); err == nil {
		t.Fatal("current fleet audit accepted a repository failure")
	}
}

func TestAuditEvidenceIsIdempotentAndConflictSafe(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.json")
	body := []byte("{\"schema\":\"audit\"}\n")
	if err := WriteEvidence(path, body); err != nil {
		t.Fatal(err)
	}
	if err := WriteEvidence(path, body); err != nil {
		t.Fatalf("identical evidence was not idempotent: %v", err)
	}
	if err := WriteEvidence(path, []byte("different\n")); err == nil {
		t.Fatal("different evidence replaced an existing audit")
	}
	if actual, err := os.ReadFile(path); err != nil || string(actual) != string(body) {
		t.Fatalf("evidence changed: %q %v", actual, err)
	}
}
