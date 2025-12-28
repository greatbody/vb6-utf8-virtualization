//go:build !windows

package vfs

func getProcessName(pid uint32) (string, error) {
	return "unknown.exe", nil
}
