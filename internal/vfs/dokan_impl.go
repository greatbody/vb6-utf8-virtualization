package vfs

import (
	"context"
	"fmt"
	"io"
	"log"
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
	processName, _ := getProcessName(uint32(fi.ProcessId()))
	if processName == "" {
		processName = fmt.Sprintf("PID:%d", fi.ProcessId())
	}

	log.Printf("[%s] CreateFile: %s (IsDir: %v)", processName, path, fi.IsDirectory())

	// Always allow root directory
	if path == "\\" {
		return &ProxyFile{fs: fs, path: path, isDir: true, physicalPath: phys}, 0, nil
	}

	st, err := os.Stat(phys)

	// Handle directory open/create
	if fi.IsDirectory() {
		if err != nil {
			if os.IsNotExist(err) {
				return nil, 0, os.ErrNotExist
			}
			return nil, 0, err
		}
		if !st.IsDir() {
			return nil, 0, fmt.Errorf("not a directory")
		}
		return &ProxyFile{fs: fs, path: path, isDir: true, physicalPath: phys}, 0, nil
	}

	// If it's a file but exists as a directory
	if err == nil && st.IsDir() {
		// Return as directory anyway if requested as file but is a directory
		// Some apps do this to check existence
		return &ProxyFile{fs: fs, path: path, isDir: true, physicalPath: phys}, 0, nil
	}

	shouldTranscode := fs.Filter.ShouldProcess(processName, path)
	if shouldTranscode {
		log.Printf("[%s] Transcoding enabled for: %s", processName, path)
	}

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
		} else if !os.IsNotExist(err) {
			return nil, 0, err
		}
	} else {
		// Passthrough
		h, err := os.OpenFile(phys, os.O_RDWR, 0)
		if err != nil {
			h, err = os.Open(phys)
		}
		if err == nil {
			file.handle = h
		} else if !os.IsNotExist(err) {
			return nil, 0, err
		}
	}

	return file, 0, nil
}

func (fs *ProxyFS) GetDiskFreeSpace(ctx context.Context) (dokan.FreeSpace, error) {
	return dokan.FreeSpace{
		FreeBytesAvailable:     10 * 1024 * 1024 * 1024,
		TotalNumberOfBytes:     20 * 1024 * 1024 * 1024,
		TotalNumberOfFreeBytes: 10 * 1024 * 1024 * 1024,
	}, nil
}

func (fs *ProxyFS) GetVolumeInformation(ctx context.Context) (dokan.VolumeInformation, error) {
	return dokan.VolumeInformation{
		VolumeName:             "UTF8Proxy",
		VolumeSerialNumber:     0x12345678,
		MaximumComponentLength: 255,
		FileSystemName:         "NTFS",
	}, nil
}

func (fs *ProxyFS) Mounted(ctx context.Context) error   { return nil }
func (fs *ProxyFS) Unmounted(ctx context.Context) error { return nil }

func (fs *ProxyFS) WithContext(c context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(c)
}

func (fs *ProxyFS) ErrorPrint(err error) {
	log.Printf("Dokan Error: %v", err)
}

func (fs *ProxyFS) Printf(format string, v ...interface{}) {
	log.Printf("Dokan: "+format, v...)
}

func (fs *ProxyFS) MoveFile(ctx context.Context, sourceHandle dokan.File, sourceFileInfo *dokan.FileInfo, targetPath string, replaceExisting bool) error {
	return nil
}

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
	processName, _ := getProcessName(uint32(fi.ProcessId()))
	if processName == "" {
		processName = fmt.Sprintf("PID:%d", fi.ProcessId())
	}
	log.Printf("[%s] GetFileInformation: %s", processName, f.path)
	st, err := os.Stat(f.physicalPath)
	if err != nil {
		log.Printf("GetFileInformation Error: %s: %v", f.path, err)
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
	processName, _ := getProcessName(uint32(fi.ProcessId()))
	if processName == "" {
		processName = fmt.Sprintf("PID:%d", fi.ProcessId())
	}
	log.Printf("[%s] FindFiles: %s (pattern: %s)", processName, f.path, pattern)
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
		if err := fill(ns); err != nil {
			return err
		}
	}
	return nil
}

func (f *ProxyFile) Cleanup(ctx context.Context, fi *dokan.FileInfo) {
	processName, _ := getProcessName(uint32(fi.ProcessId()))
	if processName == "" {
		processName = fmt.Sprintf("PID:%d", fi.ProcessId())
	}
	log.Printf("[%s] Cleanup: %s", processName, f.path)
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
func (f *ProxyFile) SetFileAttributes(ctx context.Context, fi *dokan.FileInfo, attr dokan.FileAttribute) error {
	return nil
}
func (f *ProxyFile) SetFileTime(ctx context.Context, fi *dokan.FileInfo, ctime, atime, mtime time.Time) error {
	return nil
}
