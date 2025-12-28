//go:build windows

package vfs

import (
	"path/filepath"

	"golang.org/x/sys/windows"
)

// getProcessName gets the process name (basename) from a given PID.
func getProcessName(pid uint32) (string, error) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(h)

	var buf [windows.MAX_PATH]uint16
	size := uint32(len(buf))
	err = windows.QueryFullProcessImageName(h, 0, &buf[0], &size)
	if err != nil {
		return "", err
	}
	name := windows.UTF16ToString(buf[:size])
	return filepath.Base(name), nil
}
