package terminaltests

import (
	"os"
	"strings"
	"testing"
)

func TestEveryBuildSurfacePinsGo1263(t *testing.T) {
	checks := map[string]string{
		"go.mod":                               "go 1.26.3",
		"scripts/windows-preflight.sh":         "go1.26.3",
		".github/workflows/windows-system.yml": `go-version: "1.26.3"`,
	}
	for path, required := range checks {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(body), required) {
			t.Errorf("%s does not pin %q", path, required)
		}
	}
}
