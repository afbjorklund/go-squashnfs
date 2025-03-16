// squashfs exposes a squashfs as a read-only billy.Filesystem.
package squashfs

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"

	"github.com/CalebQ42/squashfs"
)

type Squash struct {
	underlying *squashfs.Reader
}

func New(f io.ReaderAt, o int64) billy.Filesystem {
	r, err := squashfs.NewReaderAtOffset(f, o)
	if err != nil {
		log.Printf("failed to read squashfs: %v", err)
	}

	fs := &Squash{
		underlying: r,
	}

	return fs
}

func (fs *Squash) Root() string {
	return ""
}

func (fs *Squash) Stat(filename string) (os.FileInfo, error) {
	f, err := fs.underlying.Open(filename)
	if err != nil {
		return nil, err
	}

	sf := f.(*squashfs.File)
	if !sf.IsSymlink() {
		return f.Stat()
	}
	f = sf.GetSymlinkFile()
	if f == nil {
		return nil, fmt.Errorf("Failed to get symlink's file: %s", filename)
	}
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	// the name of the file should always the name of the stated file, so we
	// overwrite the Stat returned from the storage with it, since the
	// filename may belong to a link.
	fi.(*fileInfo).name = filepath.Base(filename)
	return fi, nil
}

func (fs *Squash) Open(filename string) (billy.File, error) {
	return fs.OpenFile(filename, os.O_RDONLY, 0)
}

func (fs *Squash) OpenFile(filename string, flag int, _ os.FileMode) (billy.File, error) {
	if flag&(os.O_CREATE|os.O_WRONLY|os.O_APPEND|os.O_RDWR|os.O_EXCL|os.O_TRUNC) != 0 {
		return nil, billy.ErrReadOnly
	}

	f, err := fs.underlying.Open(filename)
	if err != nil {
		return nil, err
	}

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	if fi.IsDir() {
		return nil, fmt.Errorf("cannot open directory: %s", filename)
	}

	data, err := fs.underlying.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Only load the bytes to memory if the files is needed.
	lazyFunc := func() *bytes.Reader { return bytes.NewReader(data) }
	return toFile(lazyFunc, fi), nil
}

// Join return a path with all elements joined by forward slashes.
//
// This behaviour is OS-agnostic.
func (fs *Squash) Join(elem ...string) string {
	for i, el := range elem {
		if el != "" {
			clean := filepath.Clean(strings.Join(elem[i:], "/"))
			return filepath.ToSlash(clean)
		}
	}
	return ""
}

func (fs *Squash) ReadDir(path string) ([]os.FileInfo, error) {
	e, err := fs.underlying.ReadDir(path)
	if err != nil {
		return nil, err
	}

	entries := make([]os.FileInfo, 0, len(e))
	for _, f := range e {
		fi, _ := f.Info()
		entries = append(entries, fi)
	}

	sort.Sort(memfs.ByName(entries))

	return entries, nil
}

// Chroot is not supported.
//
// Calls will always return billy.ErrNotSupported.
func (fs *Squash) Chroot(_ string) (billy.Filesystem, error) {
	return nil, billy.ErrNotSupported
}

func (fs *Squash) Lstat(filename string) (os.FileInfo, error) {
	f, err := fs.underlying.Open(filename)
	if err != nil {
		return nil, err
	}
	return f.Stat()
}

func (fs *Squash) Readlink(link string) (string, error) {
	f, err := fs.underlying.Open(link)
	if err != nil {
		return "", err
	}

	sf := f.(*squashfs.File)
	if !sf.IsSymlink() {
		return "", &os.PathError{
			Op:   "readlink",
			Path: link,
			Err:  fmt.Errorf("not a symlink"),
		}
	}

	return sf.SymlinkPath(), nil
}

// TempFile is not supported.
//
// Calls will always return billy.ErrNotSupported.
func (fs *Squash) TempFile(_, _ string) (billy.File, error) {
	return nil, billy.ErrNotSupported
}

// Symlink is not supported.
//
// Calls will always return billy.ErrReadOnly.
func (fs *Squash) Symlink(_, _ string) error {
	return billy.ErrReadOnly
}

// Create is not supported.
//
// Calls will always return billy.ErrReadOnly.
func (fs *Squash) Create(_ string) (billy.File, error) {
	return nil, billy.ErrReadOnly
}

// Rename is not supported.
//
// Calls will always return billy.ErrReadOnly.
func (fs *Squash) Rename(_, _ string) error {
	return billy.ErrReadOnly
}

// Remove is not supported.
//
// Calls will always return billy.ErrReadOnly.
func (fs *Squash) Remove(_ string) error {
	return billy.ErrReadOnly
}

// MkdirAll is not supported.
//
// Calls will always return billy.ErrReadOnly.
func (fs *Squash) MkdirAll(_ string, _ os.FileMode) error {
	return billy.ErrReadOnly
}

func toFile(lazy func() *bytes.Reader, fi fs.FileInfo) billy.File {
	return &file{
		lazy: lazy,
		fi:   fi,
	}
}

type file struct {
	lazy   func() *bytes.Reader
	reader *bytes.Reader
	fi     fs.FileInfo
	once   sync.Once
}

func (f *file) loadReader() {
	f.reader = f.lazy()
}

func (f *file) Name() string {
	return f.fi.Name()
}

func (f *file) Read(b []byte) (int, error) {
	f.once.Do(f.loadReader)

	return f.reader.Read(b)
}

func (f *file) ReadAt(b []byte, off int64) (int, error) {
	f.once.Do(f.loadReader)

	return f.reader.ReadAt(b, off)
}

func (f *file) Seek(offset int64, whence int) (int64, error) {
	f.once.Do(f.loadReader)

	return f.reader.Seek(offset, whence)
}

func (f *file) Stat() (os.FileInfo, error) {
	return &fileInfo{
		name:  f.fi.Name(),
		size:  f.fi.Size(),
		mode:  f.fi.Mode(),
		mtime: f.fi.ModTime(),
	}, nil
}

// Close for squashfs file is a no-op.
func (f *file) Close() error {
	return nil
}

// Lock for squashfs file is a no-op.
func (f *file) Lock() error {
	return nil
}

// Unlock for squashfs file is a no-op.
func (f *file) Unlock() error {
	return nil
}

type fileInfo struct {
	name  string
	size  int64
	mode  os.FileMode
	mtime time.Time
}

func (fi *fileInfo) Name() string {
	return fi.name
}

func (fi *fileInfo) Size() int64 {
	return fi.size
}

func (fi *fileInfo) Mode() os.FileMode {
	return fi.mode
}

func (fi *fileInfo) ModTime() time.Time {
	return fi.mtime
}

func (fi *fileInfo) IsDir() bool {
	return fi.mode.IsDir()
}

func (*fileInfo) Sys() interface{} {
	return nil
}

// Truncate is not supported.
//
// Calls will always return billy.ErrReadOnly.
func (f *file) Truncate(_ int64) error {
	return billy.ErrReadOnly
}

// Write is not supported.
//
// Calls will always return billy.ErrReadOnly.
func (f *file) Write(_ []byte) (int, error) {
	return 0, billy.ErrReadOnly
}

// WriteAt is not supported.
//
// Calls will always return billy.ErrReadOnly.
func (f *file) WriteAt([]byte, int64) (int, error) {
	return 0, billy.ErrReadOnly
}
