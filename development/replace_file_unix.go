//go:build !windows

package development

import "os"

func replaceFile(source, destination string) error {
	return os.Rename(source, destination)
}
