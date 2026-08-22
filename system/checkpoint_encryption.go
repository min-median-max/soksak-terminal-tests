package system

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var checkpointMagic = []byte("SKTERM01")

const checkpointMinimumBytes = 8 + 1 + 8 + 8 + 12 + 16

func verifyEncryptedCheckpoints(home string, providers, plaintext []string, expectedFiles int) error {
	root := filepath.Join(home, "terminal-checkpoints")
	count := 0
	for _, provider := range providers {
		directory := filepath.Join(root, provider)
		entries, err := os.ReadDir(directory)
		if err != nil {
			return fmt.Errorf("%s checkpoints: %w", provider, err)
		}
		providerFiles := 0
		for _, entry := range entries {
			path := filepath.Join(directory, entry.Name())
			info, err := os.Lstat(path)
			if err != nil || !info.Mode().IsRegular() {
				return fmt.Errorf("%s is not a regular checkpoint file: %v", path, err)
			}
			final := strings.HasSuffix(entry.Name(), ".checkpoint")
			temporary := strings.HasPrefix(entry.Name(), ".") && strings.Contains(entry.Name(), ".checkpoint.") && strings.HasSuffix(entry.Name(), ".tmp")
			if !final && !temporary {
				return fmt.Errorf("%s is not a checkpoint path", path)
			}
			body, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if len(body) < checkpointMinimumBytes || !bytes.Equal(body[:8], checkpointMagic) || body[8] != 1 {
				return fmt.Errorf("%s has an invalid encrypted checkpoint envelope", path)
			}
			for _, marker := range plaintext {
				if marker != "" && bytes.Contains(body, []byte(marker)) {
					return fmt.Errorf("%s contains plaintext marker %q", path, marker)
				}
			}
			if final {
				providerFiles++
				count++
			}
		}
		if providerFiles == 0 {
			return fmt.Errorf("%s has no encrypted checkpoints", provider)
		}
	}
	if count != expectedFiles {
		return fmt.Errorf("encrypted checkpoint files=%d, want %d", count, expectedFiles)
	}
	return nil
}
