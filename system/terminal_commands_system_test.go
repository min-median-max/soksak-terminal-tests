//go:build system

package system

import (
	"os"
	"testing"
)

func TestInstalledTerminalCommands(t *testing.T) {
	cli := CLI{Path: os.Getenv("SOKSAK_TEST_CLI"), Socket: os.Getenv("SOKSAK_TEST_SOCKET"), Window: os.Getenv("SOKSAK_TEST_WINDOW"), EvidenceDir: os.Getenv("SOKSAK_TEST_EVIDENCE")}
	results, err := VerifyTerminalCommands(cli)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 7 {
		t.Fatalf("verified %d terminal plugins", len(results))
	}
	if err := VerifyInstalledUI(cli); err != nil {
		t.Fatal(err)
	}
}
