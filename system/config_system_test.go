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
	return LifecycleConfig{
		App: os.Getenv("SOKSAK_TEST_APP"), CLI: os.Getenv("SOKSAK_TEST_CLI"),
		Socket: filepath.Join(os.Getenv("SOKSAK_TEST_RUNTIME"), scenario+".sock"), Home: filepath.Join(root, "home"),
		Runtime: os.Getenv("SOKSAK_TEST_RUNTIME"), Workspace: filepath.Join(root, "workspace"),
		EvidenceDir: filepath.Join(os.Getenv("SOKSAK_TEST_EVIDENCE"), scenario), Identifier: os.Getenv("SOKSAK_TEST_IDENTIFIER") + "." + scenario,
	}
}
