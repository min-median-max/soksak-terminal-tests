package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/min-median-max/soksak-terminal-tests/release"
)

func main() {
	fleet, err := release.ReadFleet("release/windows-fleet.json")
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		err = release.Verify(ctx, &http.Client{Timeout: 2 * time.Minute}, fleet)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("Windows release fleet verified")
}
