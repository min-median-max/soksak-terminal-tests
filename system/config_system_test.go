//go:build system

package system

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/min-median-max/soksak-terminal-tests/fleet"
)

func profileFromEnvironment(t *testing.T) fleet.Profile {
	t.Helper()
	profile, err := fleet.ForPlatform(os.Getenv("SOKSAK_TEST_PLATFORM"))
	if err != nil {
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
	return LifecycleConfig{
		App: os.Getenv("SOKSAK_TEST_APP"), CLI: os.Getenv("SOKSAK_TEST_CLI"),
		Socket: filepath.Join(runtime, identifier+".sock"), Home: filepath.Join(root, "home"),
		Runtime: runtime, Workspace: filepath.Join(root, "workspace"),
		EvidenceDir: filepath.Join(os.Getenv("SOKSAK_TEST_EVIDENCE"), scenario), Identifier: identifier,
	}
}
