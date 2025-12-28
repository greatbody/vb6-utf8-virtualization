package vfs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/greatbody/vb6-utf8-virtualization/internal/transcoder"
	"github.com/stirante/dokan-go"
	"github.com/stirante/dokan-go/winacl"
)

type ProxyFS struct {
	PhysicalPath string
	Filter       *Filter
	Mu           sync.Mutex
}

func NewProxyFS(physicalPath string, filter *Filter) *ProxyFS {
	return &ProxyFS{
		PhysicalPath: physicalPath,
		Filter:       filter,
	}
}

func (fs *ProxyFS) getPhysicalPath(path string) string {
	path = strings.TrimPrefix(path, "\\")
	return filepath.Join(fs.PhysicalPath, path)
}

// ProxyFS implementation

func (fs *ProxyFS) CreateFile(ctx context.Context, fi *dokan.FileInfo, cd *dokan.CreateData) (dokan.File, dokan.CreateStatus, error) {
	path := fi.Path()
	phys := fs.getPhysicalPath(path)

	st, err := os.Stat(phys)
	if err == nil && st.IsDir() {
		return &ProxyFile{fs: fs, path: path, isDir: true, physicalPath: phys}, 0, nil
	}

	processName, _ := getProcessName(uint32(fi.ProcessId()))
	shouldTranscode := fs.Filter.ShouldProcess(processName, path)

	file := &ProxyFile{
		fs:           fs,
		path:         path,
		physicalPath: phys,
		transcoding:  shouldTranscode,
	}

	if shouldTranscode {
		content, err := os.ReadFile(phys)
		if err == nil {
			utf8Content, _ := transcoder.NormalizeToUTF8(content)
			file.utf8Content = utf8Content
			file.originalSize = int64(len(content))
		}
	} else {
		// Passthrough
		h, err := os.Open(phys)
		if err == nil {
			file.handle = h
		}
	}

	return file, 0, nil
}

func (fs *ProxyFS) GetDiskFreeSpace(ctx context.Context, fi *dokan.FileInfo) (dokan.FreeSpace, error) {
	return dokan.FreeSpace{
		FreeBytesAvailable:     10 * 1024 * 1024 * 1024,
		TotalNumberOfBytes:     20 * 1024 * 1024 * 1024,
		TotalNumberOfFreeBytes: 10 * 1024 * 1024 * 1024,
	}, nil
}

func (fs *ProxyFS) GetVolumeInformation(ctx context.Context, fi *dokan.FileInfo) (dokan.VolumeInformation, error) {
	return dokan.VolumeInformation{
		VolumeName:             "UTF8Proxy",
		VolumeSerialNumber:     0x12345678,
		MaximumComponentLength: 255,
		FileSystemName:         "NTFS",
	}, nil
}

func (fs *ProxyFS) Mounted(ctx context.Context, fi *dokan.FileInfo) error   { return nil }
func (fs *ProxyFS) Unmounted(ctx context.Context, fi *dokan.FileInfo) error { return nil }

func (fs *ProxyFS) Print(s string)      {}
func (fs *ProxyFS) ErrorPrint(s string) {}

// ProxyFile implementation

type ProxyFile struct {
	fs           *ProxyFS
	path         string
	physicalPath string
	transcoding  bool
	utf8Content  []byte
	originalSize int64
	isDirty      bool
	handle       *os.File
	isDir        bool
}

func (f *ProxyFile) ReadFile(ctx context.Context, fi *dokan.FileInfo, bs []byte, offset int64) (int, error) {
	if f.transcoding {
		if offset >= int64(len(f.utf8Content)) {
			return 0, io.EOF
		}
		n := copy(bs, f.utf8Content[offset:])
		return n, nil
	}
	if f.handle != nil {
		return f.handle.ReadAt(bs, offset)
	}
	return 0, io.EOF
}

func (f *ProxyFile) WriteFile(ctx context.Context, fi *dokan.FileInfo, bs []byte, offset int64) (int, error) {
	if f.transcoding {
		end := offset + int64(len(bs))
		if end > int64(len(f.utf8Content)) {
			newBuf := make([]byte, end)
			copy(newBuf, f.utf8Content)
			f.utf8Content = newBuf
		}
		copy(f.utf8Content[offset:], bs)
		f.isDirty = true
		return len(bs), nil
	}
	return 0, fmt.Errorf("write not supported")
}

func (f *ProxyFile) GetFileInformation(ctx context.Context, fi *dokan.FileInfo) (*dokan.Stat, error) {
	st, err := os.Stat(f.physicalPath)
	if err != nil {
		return &dokan.Stat{FileAttributes: 128}, nil
	}
	s := &dokan.Stat{
		LastWrite:  st.ModTime(),
		LastAccess: st.ModTime(),
		Creation:   st.ModTime(),
		FileSize:   st.Size(),
	}
	if st.IsDir() {
		s.FileAttributes = 16
	} else {
		s.FileAttributes = 128
		if f.transcoding {
			s.FileSize = int64(len(f.utf8Content))
		}
	}
	return s, nil
}

func (f *ProxyFile) FindFiles(ctx context.Context, fi *dokan.FileInfo, pattern string, fill func(*dokan.NamedStat) error) error {
	phys := f.physicalPath
	entries, err := os.ReadDir(phys)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		info, _ := entry.Info()
		ns := &dokan.NamedStat{
			Name: entry.Name(),
			Stat: dokan.Stat{
				FileSize:   info.Size(),
				LastWrite:  info.ModTime(),
				LastAccess: info.ModTime(),
				Creation:   info.ModTime(),
			},
		}
		if entry.IsDir() {
			ns.Stat.FileAttributes = 16
		} else {
			ns.Stat.FileAttributes = 128
		}
		fill(ns)
	}
	return nil
}

func (f *ProxyFile) Cleanup(ctx context.Context, fi *dokan.FileInfo) {
	if f.isDirty && f.transcoding {
		gbData, err := transcoder.ConvertToGB18030(f.utf8Content)
		if err == nil {
			os.WriteFile(f.physicalPath, gbData, 0644)
		}
	}
	if f.handle != nil {
		f.handle.Close()
	}
}

func (f *ProxyFile) CloseFile(ctx context.Context, fi *dokan.FileInfo) {}

func (f *ProxyFile) FlushFileBuffers(ctx context.Context, fi *dokan.FileInfo) error { return nil }
func (f *ProxyFile) SetEndOfFile(ctx context.Context, fi *dokan.FileInfo, length int64) error {
	return nil
}
func (f *ProxyFile) SetAllocationSize(ctx context.Context, fi *dokan.FileInfo, length int64) error {
	return nil
}
func (f *ProxyFile) LockFile(ctx context.Context, fi *dokan.FileInfo, offset, length int64) error {
	return nil
}
func (f *ProxyFile) UnlockFile(ctx context.Context, fi *dokan.FileInfo, offset, length int64) error {
	return nil
}
func (f *ProxyFile) CanDeleteFile(ctx context.Context, fi *dokan.FileInfo) error      { return nil }
func (f *ProxyFile) CanDeleteDirectory(ctx context.Context, fi *dokan.FileInfo) error { return nil }

func (f *ProxyFile) GetFileSecurity(ctx context.Context, fi *dokan.FileInfo, si winacl.SecurityInformation, sd *winacl.SecurityDescriptor) error {
	return nil
}
func (f *ProxyFile) SetFileSecurity(ctx context.Context, fi *dokan.FileInfo, si winacl.SecurityInformation, sd *winacl.SecurityDescriptor) error {
	return nil
}
func (f *ProxyFile) SetFileAttributes(ctx context.Context, fi *dokan.FileInfo, attr uint32) error {
	return nil
}
func (f *ProxyFile) SetFileTime(ctx context.Context, fi *dokan.FileInfo, ctime, atime, mtime time.Time) error {
	return nil
}
