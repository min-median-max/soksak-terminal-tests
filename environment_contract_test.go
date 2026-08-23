package terminaltests

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAcceptanceUsesOneEnvironmentAndCurrentSidecarRoles(t *testing.T) {
	forbidden := []string{
		"settings.json", "installed.json", "SOKSAK_TEST_SETTINGS",
		"platformspec.Settings", "platformspec.Installed", "Providers",
	}
	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "local" || entry.Name() == ".task" {
				return filepath.SkipDir
			}
			return nil
		}
		if path == "environment_contract_test.go" {
			return nil
		}
		if filepath.Ext(path) != ".go" && filepath.Ext(path) != ".md" && filepath.Ext(path) != ".sh" && filepath.Ext(path) != ".yml" {
			return nil
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, token := range forbidden {
			if strings.Contains(string(body), token) {
				t.Errorf("%s contains retired environment token %q", path, token)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
