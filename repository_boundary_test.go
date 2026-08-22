package terminaltests

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepositoryDoesNotExecuteOwnerSources(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tokens := []string{"soksak-" + "plugins", "soksak-" + "sidecars", "soksak-" + "kits"}
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if path != root && (info.Name() == ".git" || info.Name() == ".task") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" && filepath.Ext(path) != ".sh" && filepath.Ext(path) != ".yml" {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		line := 0
		for scanner.Scan() {
			line++
			for _, token := range tokens {
				if strings.Contains(scanner.Text(), token) {
					t.Errorf("%s:%d references sibling source %s", path, line, token)
				}
			}
		}
		return scanner.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"plugin.json", "soksak-unit.json", "release.json"} {
		if _, err := os.Stat(forbidden); !os.IsNotExist(err) {
			t.Errorf("test repository contains install manifest %s", forbidden)
		}
	}
}

func TestRepositoryUsesCanonicalSettingsAndInstalledState(t *testing.T) {
	for _, path := range []string{"development/settings.go", "system/inventory.go"} {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		source := string(body)
		if !strings.Contains(source, "github.com/soksak-ai/soksak-spec/go/platformspec") {
			t.Errorf("%s does not use the canonical platform state parser", path)
		}
		for _, obsolete := range []string{"soksak-spec-composition", "type Component struct", "SettingsSpec"} {
			if strings.Contains(source, obsolete) {
				t.Errorf("%s contains obsolete platform state %q", path, obsolete)
			}
		}
	}
}
