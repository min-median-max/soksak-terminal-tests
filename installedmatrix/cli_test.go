package installedmatrix

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPublicCLIEnvelopeUsesOKAndObjectData(t *testing.T) {
	data, err := decodePublicEnvelope("sidecar.status", []byte(`{"ok":true,"data":{"units":[]}}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if units, ok := data["units"].([]any); !ok || len(units) != 0 {
		t.Fatalf("public data = %+v", data)
	}
	for _, run := range []struct {
		body   string
		runErr error
	}{
		{`{"code":"OK","data":{"units":[]}}`, nil},
		{`{"ok":false,"code":"REFUSED","error":"not ready"}`, nil},
		{`{"ok":true,"data":[]}`, nil},
		{"not-json", nil},
		{`{"ok":true,"data":{"units":[]}}`, errors.New("exit status 1")},
	} {
		if _, err := decodePublicEnvelope("sidecar.status", []byte(run.body), run.runErr); err == nil {
			t.Fatalf("invalid public envelope was accepted: %s", run.body)
		}
	}
}

func TestArtifactDigestRequiresARegularAbsoluteFile(t *testing.T) {
	directory := t.TempDir()
	artifact := filepath.Join(directory, "sidecar")
	if err := os.WriteFile(artifact, []byte("artifact"), 0o700); err != nil {
		t.Fatal(err)
	}
	if digest, err := DigestRegularFile(artifact); err != nil || digest == "" {
		t.Fatalf("regular artifact digest=%q err=%v", digest, err)
	}
	if _, err := DigestRegularFile(directory); err == nil {
		t.Fatal("directory was accepted as a sidecar artifact")
	}
}
