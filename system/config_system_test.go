//go:build system

package system

import "os"

func lifecycleConfigFromEnvironment() LifecycleConfig {
	return LifecycleConfig{
		App: os.Getenv("SOKSAK_TEST_APP"), CLI: os.Getenv("SOKSAK_TEST_CLI"),
		Socket: os.Getenv("SOKSAK_TEST_SOCKET"), Home: os.Getenv("SOKSAK_TEST_HOME"),
		Runtime: os.Getenv("SOKSAK_TEST_RUNTIME"), Workspace: os.Getenv("SOKSAK_TEST_WORKSPACE"),
		EvidenceDir: os.Getenv("SOKSAK_TEST_EVIDENCE"), Identifier: os.Getenv("SOKSAK_TEST_IDENTIFIER"),
	}
}
