package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/min-median-max/soksak-terminal-tests/development"
	"github.com/min-median-max/soksak-terminal-tests/release"
)

func main() {
	stage := flag.String("stage", "", "absolute staging directory")
	flag.Parse()
	if !filepath.IsAbs(*stage) {
		fail(fmt.Errorf("--stage must be absolute"))
	}
	fleet, err := release.ReadFleet("release/windows-fleet.json")
	if err != nil {
		fail(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	staged, err := release.VerifyAndStage(ctx, &http.Client{Timeout: 2 * time.Minute}, fleet, *stage)
	if err != nil {
		fail(err)
	}
	input := development.Input{Platform: "windows", Target: fleet.Target, Home: filepath.Join(*stage, "composition-home"), Plugins: map[string]development.ArtifactInput{}, Sidecars: map[string]development.ArtifactInput{}}
	for id, artifact := range staged.Plugins {
		input.Plugins[id] = development.ArtifactInput{Path: artifact.Path, Repository: artifact.Repository, Commit: artifact.Commit, ArtifactSHA256: artifact.ArtifactSHA256}
	}
	for id, artifact := range staged.Sidecars {
		input.Sidecars[id] = development.ArtifactInput{Path: artifact.Path, Repository: artifact.Repository, Commit: artifact.Commit, ArtifactSHA256: artifact.ArtifactSHA256, Target: artifact.Target}
	}
	paths, err := development.PrepareState(input)
	if err != nil {
		fail(err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(paths); err != nil {
		fail(err)
	}
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
