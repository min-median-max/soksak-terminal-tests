package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/min-median-max/soksak-terminal-tests/candidate"
)

func main() {
	plan := flag.String("plan", "", "absolute local release selection plan")
	store := flag.String("store", "", "absolute immutable local release store")
	output := flag.String("out", "", "absolute candidate plan output")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "only named options are accepted")
		os.Exit(2)
	}
	state, err := candidate.ComposeLocal(candidate.LocalComposeOptions{
		Plan: *plan, Store: *store, Output: *output,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_ = json.NewEncoder(os.Stdout).Encode(map[string]string{"state": state, "output": *output})
}
