//go:build !windows

package system

import (
	"fmt"
	"path/filepath"
)

const portableUnixSocketPathLimit = 104

func validateControlAddress(path string) error {
	if len(path) >= portableUnixSocketPathLimit {
		return fmt.Errorf("control socket path is %d bytes; maximum is %d: %s", len(path), portableUnixSocketPathLimit-1, path)
	}
	return nil
}

func controlAddress(runtime, identifier string) string {
	return filepath.Join(runtime, identifier+".sock")
}
