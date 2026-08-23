package terminaltests

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestGoModIsTheOnlyToolchainVersionSource(t *testing.T) {
	moduleBody, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatal(err)
	}
	match := regexp.MustCompile(`(?m)^go ([0-9]+\.[0-9]+\.[0-9]+)$`).FindSubmatch(moduleBody)
	if len(match) != 2 {
		t.Fatal("go.mod must contain one exact Go version")
	}
	version := string(match[1])
	checks := map[string][]string{
		"scripts/windows-preflight.sh":         {"required=$(awk", "go env GOVERSION"},
		".github/workflows/windows-system.yml": {"go-version-file: tests/go.mod"},
	}
	for path, required := range checks {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(body)
		if strings.Contains(text, version) {
			t.Errorf("%s duplicates the canonical Go version", path)
		}
		for _, token := range required {
			if !strings.Contains(text, token) {
				t.Errorf("%s does not derive Go from go.mod through %q", path, token)
			}
		}
	}
}
