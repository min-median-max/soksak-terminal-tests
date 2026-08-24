package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/min-median-max/soksak-terminal-tests/fleet"
	"github.com/min-median-max/soksak-terminal-tests/releaseaudit"
)

func main() {
	platform := flag.String("platform", "", "fleet platform")
	target := flag.String("target", "", "fleet artifact target")
	specPackage := flag.String("spec-package", "", "extracted immutable plugin-spec package")
	out := flag.String("out", "", "audit evidence JSON")
	flag.Parse()
	if err := run(*platform, *target, *specPackage, *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(platform, target, specPackage, out string) error {
	profile, err := fleet.ForTarget(platform, target)
	if err != nil {
		return err
	}
	if err := requireNativeRuntime(profile); err != nil {
		return err
	}
	root, err := filepath.Abs(specPackage)
	if err != nil || root == "" {
		return fmt.Errorf("absolute spec package path required")
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil || resolved != root {
		return fmt.Errorf("spec package path must contain no symbolic link: %s", root)
	}
	tool := filepath.Join(root, "release-template", "sidecar", "audit-releases.mjs")
	if info, err := os.Stat(tool); err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("spec package has no sidecar release auditor")
	}
	if err := requireNode(root, profile); err != nil {
		return err
	}
	runner := func(ctx context.Context, invocation releaseaudit.Invocation) ([]byte, error) {
		command := exec.CommandContext(ctx, "node", tool,
			"--repository", invocation.Repository, "--tag", invocation.Tag)
		command.Dir = root
		var stdout, stderr bytes.Buffer
		command.Stdout, command.Stderr = &stdout, &stderr
		err := command.Run()
		if err != nil && stderr.Len() > 0 {
			err = fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return stdout.Bytes(), err
	}
	report, err := releaseaudit.AuditCurrentFleet(context.Background(), profile, root, runner)
	encoded, encodeErr := json.MarshalIndent(report, "", "  ")
	if encodeErr != nil {
		return encodeErr
	}
	encoded = append(encoded, '\n')
	if writeErr := releaseaudit.WriteEvidence(out, encoded); writeErr != nil {
		return writeErr
	}
	return err
}

func requireNativeRuntime(profile fleet.Profile) error {
	wantOS, wantArch := profile.Platform, "amd64"
	if strings.HasPrefix(profile.Target, "aarch64-") {
		wantArch = "arm64"
	}
	if runtime.GOOS != wantOS || runtime.GOARCH != wantArch {
		return fmt.Errorf("audit runtime is %s/%s, want %s/%s", runtime.GOOS, runtime.GOARCH, wantOS, wantArch)
	}
	return nil
}

func requireNode(specPackage string, profile fleet.Profile) error {
	body, err := os.ReadFile(filepath.Join(specPackage, "package.json"))
	if err != nil {
		return err
	}
	var manifest struct {
		Engines struct {
			Node string `json:"node"`
		} `json:"engines"`
	}
	if err := json.Unmarshal(body, &manifest); err != nil || manifest.Engines.Node == "" {
		return fmt.Errorf("spec package has no exact Node engine")
	}
	version, err := exec.Command("node", "--version").Output()
	if err != nil || strings.TrimSpace(string(version)) != "v"+manifest.Engines.Node {
		return fmt.Errorf("Node %s is required", manifest.Engines.Node)
	}
	actual, err := exec.Command("node", "-p", `process.platform+"/"+process.arch`).Output()
	if err != nil {
		return err
	}
	wantArch := "x64"
	if strings.HasPrefix(profile.Target, "aarch64-") {
		wantArch = "arm64"
	}
	if strings.TrimSpace(string(actual)) != profile.Platform+"/"+wantArch {
		return fmt.Errorf("Node runtime is %s, want %s/%s", strings.TrimSpace(string(actual)), profile.Platform, wantArch)
	}
	return nil
}
