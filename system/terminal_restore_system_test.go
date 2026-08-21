//go:build system

package system

import (
	"os"
	"testing"
)

func TestInstalledTerminalWarmAndArchivedRestore(t *testing.T) {
	config := LifecycleConfig{
		App: os.Getenv("SOKSAK_TEST_APP"), CLI: os.Getenv("SOKSAK_TEST_CLI"),
		Socket: os.Getenv("SOKSAK_TEST_SOCKET"), Home: os.Getenv("SOKSAK_TEST_HOME"),
		Runtime: os.Getenv("SOKSAK_TEST_RUNTIME"), Workspace: os.Getenv("SOKSAK_TEST_WORKSPACE"),
		EvidenceDir: os.Getenv("SOKSAK_TEST_EVIDENCE"), Identifier: os.Getenv("SOKSAK_TEST_IDENTIFIER"),
	}
	lifecycle, err := NewLifecycle(config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(lifecycle.Close)
	if err := lifecycle.Start(); err != nil {
		t.Fatal(err)
	}
	if _, err := lifecycle.OpenWorkspace(); err != nil {
		t.Fatal(err)
	}
	views, err := VerifyTerminalCommands(lifecycle.Client())
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyWarmAndArchivedRestore(lifecycle, views); err != nil {
		t.Fatal(err)
	}
	if err := VerifyInstalledUI(lifecycle.Client()); err != nil {
		t.Fatal(err)
	}
}
