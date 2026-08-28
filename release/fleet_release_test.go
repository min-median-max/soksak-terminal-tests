package release

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestReleaseAssetURLDerivesTheArtifactFileLocation(t *testing.T) {
	got, err := releaseAssetURL(
		"https://github.com/soksak-ai/soksak-sidecar-example",
		"0.0.1",
		"soksak-sidecar-example-0.0.1-aarch64-apple-darwin.tar.gz",
	)
	if err != nil {
		t.Fatal(err)
	}
	want := "https://github.com/soksak-ai/soksak-sidecar-example/releases/download/v0.0.1/soksak-sidecar-example-0.0.1-aarch64-apple-darwin.tar.gz"
	if got != want {
		t.Fatalf("artifact URL = %q, want %q", got, want)
	}
	if _, err := releaseAssetURL("https://github.com/soksak-ai/soksak-sidecar-example", "0.0.1", "../escape"); err == nil {
		t.Fatal("unsafe release file was accepted")
	}
}

func TestInspectArchiveRequiresTheDeclaredPlatformProcess(t *testing.T) {
	component := Component{ID: "soksak-sidecar-example", Version: "0.0.1"}
	valid := archiveFixture(t, map[string][]byte{
		"sidecar.json":                    []byte(`{"id":"soksak-sidecar-example","version":"0.0.1","interface":{"id":"soksak-spec-sidecar-example","version":"0.0.1"},"process":"dist/soksak-sidecar-example.exe"}`),
		"dist/soksak-sidecar-example.exe": []byte("binary"),
	})
	if err := inspectArchive(valid, "sidecar", component, "x86_64-pc-windows-msvc"); err != nil {
		t.Fatal(err)
	}
	missing := archiveFixture(t, map[string][]byte{
		"sidecar.json": []byte(`{"id":"soksak-sidecar-example","version":"0.0.1","interface":{"id":"soksak-spec-sidecar-example","version":"0.0.1"},"process":"dist/soksak-sidecar-example.exe"}`),
	})
	if err := inspectArchive(missing, "sidecar", component, "x86_64-pc-windows-msvc"); err == nil {
		t.Fatal("missing Windows process was accepted")
	}
}

func TestInspectArchiveRejectsLinks(t *testing.T) {
	filename := archiveFixtureWithLink(t)
	if err := inspectArchive(filename, "sidecar", Component{ID: "soksak-sidecar-example", Version: "0.0.1"}, "x86_64-pc-windows-msvc"); err == nil {
		t.Fatal("archive link was accepted")
	}
}

func TestInspectArchiveRejectsDotPrefixedPaths(t *testing.T) {
	component := Component{ID: "soksak-sidecar-example", Version: "0.0.1"}
	archive := archiveFixture(t, map[string][]byte{
		"./sidecar.json":                    []byte(`{"id":"soksak-sidecar-example","version":"0.0.1","interface":{"id":"soksak-spec-sidecar-example","version":"0.0.1"},"process":"dist/soksak-sidecar-example.exe"}`),
		"./dist/soksak-sidecar-example.exe": []byte("binary"),
	})
	if err := inspectArchive(archive, "sidecar", component, "x86_64-pc-windows-msvc"); err == nil {
		t.Fatal("dot-prefixed archive paths were accepted")
	}
}

func TestInspectArchiveRequiresAUnixProcessForDarwinAndLinux(t *testing.T) {
	component := Component{ID: "soksak-sidecar-example", Version: "0.0.1"}
	valid := archiveFixture(t, map[string][]byte{
		"sidecar.json":                []byte(`{"id":"soksak-sidecar-example","version":"0.0.1","interface":{"id":"soksak-spec-sidecar-example","version":"0.0.1"},"process":"dist/soksak-sidecar-example"}`),
		"dist/soksak-sidecar-example": []byte("binary"),
	})
	for _, target := range []string{"aarch64-apple-darwin", "x86_64-unknown-linux-gnu"} {
		if err := inspectArchive(valid, "sidecar", component, target); err != nil {
			t.Fatalf("%s: %v", target, err)
		}
	}
}

func TestExtractArchiveWritesOnlyDeclaredRegularFiles(t *testing.T) {
	archive := archiveFixture(t, map[string][]byte{"plugin.json": []byte("manifest"), "main.js": []byte("first")})
	destination := filepath.Join(t.TempDir(), "plugin")
	if err := os.MkdirAll(destination, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := extractArchive(archive, destination); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(destination, "main.js"))
	if err != nil || string(body) != "first" {
		t.Fatalf("body=%q err=%v", body, err)
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
