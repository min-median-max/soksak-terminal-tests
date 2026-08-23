package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/min-median-max/soksak-terminal-tests/fleet"
	"github.com/min-median-max/soksak-terminal-tests/release"
)

func main() {
	platform := flag.String("platform", "", "runtime platform")
	target := flag.String("target", "", "sidecar artifact target")
	flag.Parse()
	profile, err := fleet.ForTarget(*platform, *target)
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		err = release.Verify(ctx, &http.Client{Timeout: 2 * time.Minute}, profile)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fingerprint, err := profile.Fingerprint()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("%s/%s release fleet verified: %s\n", *platform, *target, fingerprint)
}
