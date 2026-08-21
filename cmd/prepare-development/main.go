package main

import (
	"fmt"
	"os"

	"github.com/min-median-max/soksak-terminal-tests/development"
)

func main() {
	input, err := development.DecodeInput(os.Stdin)
	if err == nil {
		var path string
		path, err = development.WriteSettings(input)
		if err == nil {
			fmt.Println(path)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
