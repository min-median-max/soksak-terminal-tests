package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/min-median-max/soksak-terminal-tests/candidate"
)

func main() {
	sourcePlan := flag.String("source-plan", "", "absolute candidate source plan")
	artifacts := flag.String("artifacts", "", "absolute downloaded artifact root")
	output := flag.String("out", "", "absolute output candidate plan")
	flag.Parse()
	if flag.NArg() != 0 || *sourcePlan == "" || *artifacts == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "usage: compose-candidate-plan -source-plan <absolute> -artifacts <absolute> -out <absolute>")
		os.Exit(2)
	}
	if err := candidate.Compose(*sourcePlan, *artifacts, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
