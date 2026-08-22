package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/min-median-max/soksak-terminal-tests/development"
)

func main() {
	input, err := development.DecodeInput(os.Stdin)
	if err == nil {
		var paths development.StatePaths
		paths, err = development.PrepareState(input)
		if err == nil {
			err = json.NewEncoder(os.Stdout).Encode(paths)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
