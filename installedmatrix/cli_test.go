package installedmatrix

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPublicCLIEnvelopeNormalizesRendererAndDirectControlSuccess(t *testing.T) {
	data, err := decodePublicEnvelope("sidecar.status", []byte(`{"ok":true,"data":{"units":[]}}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if units, ok := data["units"].([]any); !ok || len(units) != 0 {
		t.Fatalf("public data = %+v", data)
	}
	direct, err := os.ReadFile(filepath.Join("testdata", "surface-composition-direct.json"))
	if err != nil {
		t.Fatal(err)
	}
	composition, err := decodePublicEnvelope("surface.composition", direct, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateComposition(composition); err != nil {
		t.Fatal(err)
	}
	if surfaces, ok := composition["surfaces"].([]any); !ok || len(surfaces) != 14 {
		t.Fatalf("real composition surfaces = %T/%d", composition["surfaces"], len(surfaces))
	}
	for _, run := range []struct {
		body   string
		runErr error
	}{
		{`{"code":"REFUSED","data":{"units":[]}}`, nil},
		{`{"ok":false,"code":"REFUSED","error":"not ready"}`, nil},
		{`{"ok":false,"code":"OK","data":{"units":[]}}`, nil},
		{`{"ok":true,"code":"REFUSED","data":{"units":[]}}`, nil},
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
