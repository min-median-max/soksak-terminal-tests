//go:build system

package system

import (
	"os"
	"path/filepath"
	"testing"
)

func lifecycleConfigFromEnvironment(t *testing.T, scenario string) LifecycleConfig {
	t.Helper()
	root := t.TempDir()
	identifier := os.Getenv("SOKSAK_TEST_IDENTIFIER") + "." + scenario
	runtime := os.Getenv("SOKSAK_TEST_RUNTIME")
	return LifecycleConfig{
		App: os.Getenv("SOKSAK_TEST_APP"), CLI: os.Getenv("SOKSAK_TEST_CLI"),
		Socket: filepath.Join(runtime, identifier+".sock"), Home: filepath.Join(root, "home"),
		Runtime: runtime, Workspace: filepath.Join(root, "workspace"),
		EvidenceDir: filepath.Join(os.Getenv("SOKSAK_TEST_EVIDENCE"), scenario), Identifier: identifier,
	}
}
