package system

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEncryptedCheckpointFilesRejectPlaintextAndInvalidEnvelopes(t *testing.T) {
	home := t.TempDir()
	provider := "soksak-sidecar-terminal-alacritty"
	directory := filepath.Join(home, "terminal-checkpoints", provider)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	marker := "SOKSAK_PRIVATE_MARKER"
	valid := append([]byte("SKTERM01\x01"), make([]byte, 8+8+12+16)...)
	if err := os.WriteFile(filepath.Join(directory, "one.checkpoint"), valid, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyEncryptedCheckpoints(home, []string{provider}, []string{marker}, 1); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, ".one.checkpoint.7.tmp"), valid, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyEncryptedCheckpoints(home, []string{provider}, []string{marker}, 1); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, ".one.checkpoint.7.tmp"), []byte(marker), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyEncryptedCheckpoints(home, []string{provider}, []string{marker}, 1); err == nil {
		t.Fatal("accepted plaintext in an in-progress checkpoint")
	}
	if err := os.Remove(filepath.Join(directory, ".one.checkpoint.7.tmp")); err != nil {
		t.Fatal(err)
	}

	for name, body := range map[string][]byte{
		"plaintext": append(valid, marker...),
		"header":    append([]byte("NOTCRYPT\x01"), valid[9:]...),
		"short":     []byte("SKTERM01\x01"),
	} {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(filepath.Join(directory, "one.checkpoint"), body, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := verifyEncryptedCheckpoints(home, []string{provider}, []string{marker}, 1); err == nil {
				t.Fatal("accepted invalid checkpoint bytes")
			}
		})
	}
}

func TestEncryptedCheckpointFilesRequireEveryProviderAndView(t *testing.T) {
	home := t.TempDir()
	err := verifyEncryptedCheckpoints(home, []string{"provider-a", "provider-b"}, []string{"marker"}, 2)
	if err == nil || !strings.Contains(err.Error(), "provider-a") {
		t.Fatalf("error=%v", err)
	}
}
