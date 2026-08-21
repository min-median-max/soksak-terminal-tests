//go:build windows

package system

import (
	"fmt"
	"strings"
)

func validateControlAddress(path string) error {
	if !strings.HasPrefix(path, `\\.\pipe\`) {
		return fmt.Errorf("control address must be a Windows named pipe: %s", path)
	}
	return nil
}
