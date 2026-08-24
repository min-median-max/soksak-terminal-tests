package system

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type candidateCaller struct {
	calls []recordedCall
}

func TestCandidatePlanRequiresThePortablePresentationContract(t *testing.T) {
	planPath := filepath.Join(t.TempDir(), "candidate-plan.json")
	body := `{"budgets":{"renderMs":16.666666666666668,"inputToPtyWriteMs":50},"components":[{"kind":"plugin"}]}`
	if err := os.WriteFile(planPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readCandidatePlan(planPath); err == nil {
		t.Fatal("candidate plan without its presentation contract was accepted")
	}
}

func (caller *candidateCaller) Call(command string, params map[string]any) (map[string]any, error) {
	caller.calls = append(caller.calls, recordedCall{command: command, params: params})
	switch command {
	case "artifact_install_begin":
		return map[string]any{"transactionId": "candidate-transaction"}, nil
	case "artifact_install_stage":
		identity := params["identity"].(map[string]any)
		return map[string]any{
			"handle":         identity["id"].(string) + "-handle",
			"sha256":         params["artifact"].(map[string]any)["sha256"],
			"size":           params["artifact"].(map[string]any)["size"],
			"manifestSha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"extraction":     "regular-files-only", "verifiedEntrypoints": []any{params["artifact"].(map[string]any)["manifest"]},
		}, nil
	case "artifact_install_read_utf8":
		return map[string]any{"text": "unused"}, nil
	case "artifact_install_commit":
		return map[string]any{"revision": float64(1)}, nil
	default:
		return map[string]any{}, nil
	}
}

func (caller *candidateCaller) CallValue(command string, params map[string]any) (any, error) {
	caller.calls = append(caller.calls, recordedCall{command: command, params: params})
	if command == "environment_get" {
		return map[string]any{"revision": 0, "plugins": map[string]any{}, "sidecars": map[string]any{}, "kits": map[string]any{}, "contracts": map[string]any{}, "specs": map[string]any{}}, nil
	}
	if command == "artifact_install_read_utf8" {
		handle, _ := params["handle"].(string)
		if strings.Contains(handle, "sidecar") {
			return `{"id":"soksak-sidecar-terminal-vt100","version":"0.0.12"}`, nil
		}
		return `{"id":"soksak-plugin-terminal-vt100","version":"0.0.15"}`, nil
	}
	return nil, nil
}

func TestCandidateFleetUsesOneAtomicPublicInstallTransaction(t *testing.T) {
	root := t.TempDir()
	components := []map[string]string{
		{"kind": "sidecar", "id": "soksak-sidecar-terminal-vt100", "version": "0.0.12", "manifest": "sidecar.json"},
		{"kind": "plugin", "id": "soksak-plugin-terminal-vt100", "version": "0.0.15", "manifest": "plugin.json"},
	}
	for _, component := range components {
		archive := filepath.Join(root, component["id"]+".tgz")
		writeCandidateArchive(t, archive, component["manifest"], map[string]any{
			"id": component["id"], "version": component["version"],
		})
	}
	plan := map[string]any{"budgets": map[string]any{"renderMs": 1000.0 / 60.0, "inputToPtyWriteMs": 50.0}, "components": []any{
		map[string]any{"kind": "sidecar", "id": "soksak-sidecar-terminal-vt100", "version": "0.0.12", "artifact": "soksak-sidecar-terminal-vt100.tgz", "manifest": "sidecar.json", "target": "aarch64-apple-darwin", "sourceRepository": "https://github.com/soksak-ai/soksak-sidecar-terminal-vt100", "sourceCommit": "1111111111111111111111111111111111111111"},
		map[string]any{"kind": "plugin", "id": "soksak-plugin-terminal-vt100", "version": "0.0.15", "artifact": "soksak-plugin-terminal-vt100.tgz", "manifest": "plugin.json", "sourceRepository": "https://github.com/soksak-ai/soksak-plugin-terminal-vt100", "sourceCommit": "2222222222222222222222222222222222222222"},
	}}
	body, _ := json.Marshal(plan)
	planPath := filepath.Join(root, "candidate-plan.json")
	if err := os.WriteFile(planPath, body, 0o600); err != nil {
		t.Fatal(err)
	}
	caller := &candidateCaller{}
	if err := InstallCandidateFleet(planPath, caller); err != nil {
		t.Fatal(err)
	}
	commands := []string{}
	for _, call := range caller.calls {
		commands = append(commands, call.command)
	}
	want := []string{"environment_get", "artifact_install_begin", "artifact_install_stage", "artifact_install_read_utf8", "artifact_install_stage", "artifact_install_read_utf8", "artifact_install_commit"}
	if len(commands) != len(want) {
		t.Fatalf("commands=%v", commands)
	}
	for index := range want {
		if commands[index] != want[index] {
			t.Fatalf("commands=%v want=%v", commands, want)
		}
	}
}

func writeCandidateArchive(t *testing.T, path, manifest string, value map[string]any) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)
	body, _ := json.Marshal(value)
	if err := tarWriter.WriteHeader(&tar.Header{Name: manifest, Mode: 0o600, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := tarWriter.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}
