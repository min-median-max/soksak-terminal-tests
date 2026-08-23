//go:build system

package system

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLifecycleExecutableSnapshotDoesNotFollowLaterBuildWrites(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.WriteFile(source, []byte("verified bytes"), 0o700); err != nil {
		t.Fatal(err)
	}
	snapshot := snapshotExecutable(t, t.TempDir(), source, "soksak")
	if err := os.WriteFile(source, []byte("later build"), 0o700); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "verified bytes" {
		t.Fatalf("snapshot changed to %q", body)
	}
}

func TestWindowsExecutableSnapshotKeepsThePEExtensionWithoutUnixMode(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "soksak.exe")
	if err := os.WriteFile(executable, []byte("PE bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(executable)
	if err != nil {
		t.Fatal(err)
	}
	if !validExecutableSnapshot(executable, info) {
		t.Fatal("a non-empty PE file was judged by a Unix execute bit")
	}
}
