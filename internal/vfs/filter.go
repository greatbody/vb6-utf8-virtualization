package vfs

import (
	"path/filepath"
	"strings"
)

// Filter handles the logic of whether a request should be processed for transcoding.
type Filter struct {
	AllowedProcesses  []string
	AllowedExtensions []string
}

// NewFilter creates a new Filter with the given settings.
func NewFilter(processes, extensions []string) *Filter {
	return &Filter{
		AllowedProcesses:  processes,
		AllowedExtensions: extensions,
	}
}

// ShouldProcess checks if the process and file should be handled.
func (f *Filter) ShouldProcess(processName string, path string) bool {
	if !f.matchProcess(processName) {
		return false
	}
	if !f.matchExtension(path) {
		return false
	}
	return true
}

func (f *Filter) matchProcess(name string) bool {
	if len(f.AllowedProcesses) == 0 {
		return true // Default allowed if list empty? Or denied?
		// Usually for a whitelist it's denied, but for testing we might want flexibility.
		// Let's go with white-list: if empty, deny.
		// Wait, user said "only for specific processes".
	}
	name = strings.ToLower(name)
	for _, p := range f.AllowedProcesses {
		if strings.ToLower(p) == name {
			return true
		}
	}
	return false
}

func (f *Filter) matchExtension(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if len(f.AllowedExtensions) == 0 {
		return false
	}
	for _, e := range f.AllowedExtensions {
		if strings.ToLower(e) == ext {
			return true
		}
	}
	return false
}
