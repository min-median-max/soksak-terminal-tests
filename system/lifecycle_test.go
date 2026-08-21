package system

import "testing"

func TestLifecycleRequiresDeclaredAbsoluteInputs(t *testing.T) {
	_, err := NewLifecycle(LifecycleConfig{
		App: "soksak", CLI: "/bin/sok", Socket: "/tmp/s.sock", Home: "/tmp/home",
		Runtime: "/tmp/runtime", Workspace: "/tmp/work", EvidenceDir: "/tmp/evidence", Identifier: "test",
	})
	if err == nil {
		t.Fatal("relative app path was accepted")
	}
}
