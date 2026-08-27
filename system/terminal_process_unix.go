//go:build !windows

package system

import (
	"errors"
	"syscall"
)

func processGone(pid uint32) (bool, error) {
	err := syscall.Kill(int(pid), 0)
	if err == nil {
		return false, nil
	}
	if errors.Is(err, syscall.ESRCH) {
		return true, nil
	}
	if errors.Is(err, syscall.EPERM) {
		return false, nil
	}
	return false, err
}

func terminateProcess(pid uint32) error {
	return syscall.Kill(int(pid), syscall.SIGKILL)
}
