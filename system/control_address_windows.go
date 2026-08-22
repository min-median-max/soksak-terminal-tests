//go:build windows

package system

import (
	"fmt"
	"strings"

	controlwire "github.com/soksak-ai/soksak-contract-control"
)

func validateControlAddress(path string) error {
	if !strings.HasPrefix(path, `\\.\pipe\`) {
		return fmt.Errorf("control address must be a Windows named pipe: %s", path)
	}
	return nil
}

func controlAddress(runtime, identifier string) string {
	return controlwire.Address(runtime, identifier, true)
}
