package installedmatrix

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// ExecCaller invokes one absolute sok executable without changing PATH.
type ExecCaller struct {
	Path   string
	Socket string
}

func (caller ExecCaller) Call(command string, params map[string]any) (map[string]any, error) {
	if err := validateRegularFile(caller.Path, "CLI"); err != nil {
		return nil, err
	}
	args := make([]string, 0, 4)
	if caller.Socket != "" {
		if !filepath.IsAbs(caller.Socket) {
			return nil, fmt.Errorf("socket path must be absolute: %s", caller.Socket)
		}
		if info, err := os.Lstat(caller.Socket); err != nil {
			return nil, fmt.Errorf("socket: %w", err)
		} else if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("socket path is a symbolic link: %s", caller.Socket)
		}
		args = append(args, "--socket", caller.Socket)
	}
	body, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	args = append(args, command, string(body))
	output, runErr := exec.Command(caller.Path, args...).CombinedOutput()
	return decodePublicEnvelope(command, output, runErr)
}

type publicEnvelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Code  string          `json:"code"`
	Error string          `json:"error"`
}

func decodePublicEnvelope(command string, output []byte, runErr error) (map[string]any, error) {
	if runErr != nil {
		return nil, fmt.Errorf("%s failed: %w: %s", command, runErr, output)
	}
	var envelope publicEnvelope
	if err := json.Unmarshal(output, &envelope); err != nil {
		return nil, fmt.Errorf("%s returned invalid JSON: %w: %s", command, err, output)
	}
	if !envelope.OK || len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return nil, fmt.Errorf("%s returned %s/%s: %s", command, envelope.Code, envelope.Error, output)
	}
	var data map[string]any
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return nil, fmt.Errorf("%s data is not an object: %w", command, err)
	}
	return data, nil
}

// DigestRegularFile hashes only an absolute, regular, non-symlink sidecar executable.
func DigestRegularFile(path string) (string, error) {
	if err := validateRegularFile(path, "sidecar process"); err != nil {
		return "", err
	}
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func validateRegularFile(path, label string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("%s path must be absolute: %s", label, path)
	}
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular non-symlink file: %s", label, path)
	}
	return nil
}
