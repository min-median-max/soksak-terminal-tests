//go:build system

package system

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/min-median-max/soksak-terminal-tests/fleet"
)

func profileFromEnvironment(t *testing.T) fleet.Profile {
	t.Helper()
	profile, err := fleet.ForTarget(os.Getenv("SOKSAK_TEST_PLATFORM"), os.Getenv("SOKSAK_TEST_TARGET"))
	if err != nil {
		t.Fatal(err)
	}
	var fingerprint string
	if planPath := os.Getenv("SOKSAK_TEST_CANDIDATE_PLAN"); planPath != "" {
		plan, _, planErr := readCandidatePlan(planPath)
		if planErr != nil {
			t.Fatal(planErr)
		}
		profile, planErr = applyCandidateProfile(profile, plan)
		if planErr != nil {
			t.Fatal(planErr)
		}
		fingerprint, err = candidatePlanFingerprint(planPath)
	} else {
		fingerprint, err = profile.Fingerprint()
	}
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(os.Getenv("SOKSAK_TEST_EVIDENCE"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(os.Getenv("SOKSAK_TEST_EVIDENCE"), "fleet-fingerprint.txt"),
		[]byte(fingerprint+"\n"), 0o600,
	); err != nil {
		t.Fatal(err)
	}
	return profile
}

func lifecycleConfigFromEnvironment(t *testing.T, scenario string) LifecycleConfig {
	t.Helper()
	root := t.TempDir()
	identifier := os.Getenv("SOKSAK_TEST_IDENTIFIER") + "." + scenario
	runtime, err := os.MkdirTemp(os.Getenv("SOKSAK_TEST_RUNTIME"), scenario+"-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(runtime) })
	app := snapshotExecutable(t, root, os.Getenv("SOKSAK_TEST_APP"), "soksak")
	cli := snapshotExecutable(t, root, os.Getenv("SOKSAK_TEST_CLI"), "sok")
	return LifecycleConfig{
		App: app, CLI: cli,
		Socket: controlAddress(runtime, identifier), Home: filepath.Join(root, "home"),
		Runtime: runtime, Workspace: filepath.Join(root, "workspace"),
		EvidenceDir: filepath.Join(os.Getenv("SOKSAK_TEST_EVIDENCE"), scenario), Identifier: identifier,
	}
}

func snapshotExecutable(t *testing.T, root, source, name string) string {
	t.Helper()
	input, err := os.Open(source)
	if err != nil {
		t.Fatalf("open %s: %v", source, err)
	}
	defer input.Close()
	if filepath.Ext(source) == ".exe" {
		name += ".exe"
	}
	destination := filepath.Join(root, name)
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o700)
	if err != nil {
		t.Fatalf("create %s: %v", destination, err)
	}
	if _, err = io.Copy(output, input); err != nil {
		_ = output.Close()
		t.Fatalf("copy %s: %v", source, err)
	}
	if err = output.Close(); err != nil {
		t.Fatalf("close %s: %v", destination, err)
	}
	info, err := os.Stat(destination)
	if err != nil || !validExecutableSnapshot(destination, info) {
		t.Fatalf("invalid executable snapshot %s: %v", destination, fmt.Sprint(err))
	}
	return destination
}

func validExecutableSnapshot(path string, info os.FileInfo) bool {
	return info.Mode().IsRegular() && info.Size() > 0 &&
		(filepath.Ext(path) == ".exe" || info.Mode().Perm()&0o100 != 0)
}
