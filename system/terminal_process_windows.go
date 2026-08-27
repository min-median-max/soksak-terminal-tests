//go:build windows

package system

import (
	"encoding/csv"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func processGone(pid uint32) (bool, error) {
	output, err := exec.Command("tasklist.exe", "/FO", "CSV", "/NH", "/FI", fmt.Sprintf("PID eq %d", pid)).Output()
	if err != nil {
		return false, err
	}
	records, err := csv.NewReader(strings.NewReader(string(output))).ReadAll()
	if err != nil {
		return false, err
	}
	for _, record := range records {
		if len(record) < 2 {
			continue
		}
		found, parseErr := strconv.ParseUint(strings.ReplaceAll(record[1], ",", ""), 10, 32)
		if parseErr == nil && uint32(found) == pid {
			return false, nil
		}
	}
	return true, nil
}

func terminateProcess(pid uint32) error {
	return exec.Command("taskkill.exe", "/PID", strconv.FormatUint(uint64(pid), 10), "/F").Run()
}
