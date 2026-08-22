package release

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"testing"
)

func TestInspectArchiveRequiresTheDeclaredWindowsProcess(t *testing.T) {
	component := Component{ID: "soksak-sidecar-example", Version: "0.0.1"}
	valid := archiveFixture(t, map[string][]byte{
		"sidecar.json":                    []byte(`{"id":"soksak-sidecar-example","version":"0.0.1","interface":{"id":"soksak-spec-sidecar-example","version":"0.0.1"},"process":"dist/soksak-sidecar-example.exe"}`),
		"dist/soksak-sidecar-example.exe": []byte("binary"),
	})
	if err := inspectArchive(valid, "sidecar", component); err != nil {
		t.Fatal(err)
	}
	missing := archiveFixture(t, map[string][]byte{
		"sidecar.json": []byte(`{"id":"soksak-sidecar-example","version":"0.0.1","interface":{"id":"soksak-spec-sidecar-example","version":"0.0.1"},"process":"dist/soksak-sidecar-example.exe"}`),
	})
	if err := inspectArchive(missing, "sidecar", component); err == nil {
		t.Fatal("missing Windows process was accepted")
	}
}

func TestInspectArchiveRejectsLinks(t *testing.T) {
	filename := archiveFixtureWithLink(t)
	if err := inspectArchive(filename, "sidecar", Component{ID: "soksak-sidecar-example", Version: "0.0.1"}); err == nil {
		t.Fatal("archive link was accepted")
	}
}

func archiveFixture(t *testing.T, files map[string][]byte) string {
	t.Helper()
	filename := t.TempDir() + "/fixture.tar.gz"
	file, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(file)
	w := tar.NewWriter(gz)
	for name, body := range files {
		if err := w.WriteHeader(&tar.Header{Name: name, Mode: 0o600, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return filename
}

func archiveFixtureWithLink(t *testing.T) string {
	t.Helper()
	filename := t.TempDir() + "/link.tar.gz"
	file, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(file)
	w := tar.NewWriter(gz)
	if err := w.WriteHeader(&tar.Header{Name: "dist/process.exe", Linkname: "../outside", Typeflag: tar.TypeSymlink}); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return filename
}
