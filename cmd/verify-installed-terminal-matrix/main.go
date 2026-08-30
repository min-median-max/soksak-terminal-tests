package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/min-median-max/soksak-terminal-tests/installedmatrix"
)

type expectationFlags []installedmatrix.EngineExpectation

func (values *expectationFlags) String() string {
	return fmt.Sprint([]installedmatrix.EngineExpectation(*values))
}

func (values *expectationFlags) Set(raw string) error {
	engine, identity, ok := strings.Cut(raw, "=")
	version, digest, digestOK := strings.Cut(identity, "@")
	decoded, decodeErr := hex.DecodeString(digest)
	if !ok || !digestOK || engine == "" || version == "" || decodeErr != nil || len(decoded) != 32 {
		return fmt.Errorf("expect must be engine=version@64-char-sha256: %s", raw)
	}
	*values = append(*values, installedmatrix.EngineExpectation{
		Engine: engine, Version: version, SHA256: strings.ToLower(digest),
	})
	return nil
}

func main() {
	cliPath := flag.String("cli", "", "absolute non-symlink sok executable")
	socket := flag.String("socket", "", "optional absolute public control socket")
	window := flag.String("window", "", "workspace window label")
	pane := flag.String("pane", "", "existing host pane id for fresh Vision tabs")
	prompt := flag.String("prompt-marker", "", "marker expected in every full fresh-pane read")
	snapshotDir := flag.String("snapshot-dir", "", "absolute existing directory for human PNG evidence")
	var expectations expectationFlags
	flag.Var(&expectations, "expect", "engine=version@sha256; repeat for all six engines")
	flag.Parse()

	report, err := installedmatrix.Verify(installedmatrix.Config{
		Window: *window, HostPane: *pane, PromptMarker: *prompt,
		SnapshotDir: *snapshotDir, Engines: expectations,
	}, installedmatrix.ExecCaller{Path: *cliPath, Socket: *socket}, installedmatrix.DigestRegularFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(string(body))
}
