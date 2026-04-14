package doubles

import (
	"io/fs"
	"os"
	"time"
)

// FileSystemStub is a test double for system.FileSystem with configurable behavior.
type FileSystemStub struct {
	RemoveFunc      func(path string) error
	RemoveAllFunc   func(path string) error
	LstatFunc       func(path string) (os.FileInfo, error)
	GlobFunc        func(pattern string) ([]string, error)
	UserHomeDirFunc func() (string, error)
	ReadDirFunc     func(dir string) ([]os.DirEntry, error)
	// RemovedAll records every path passed to RemoveAll, in call order.
	RemovedAll []string
}

func NewFileSystemStub() *FileSystemStub {
	stub := &FileSystemStub{
		RemoveFunc:      func(_ string) error { return nil },
		LstatFunc:       func(_ string) (os.FileInfo, error) { return nil, os.ErrNotExist },
		GlobFunc:        func(_ string) ([]string, error) { return nil, nil },
		UserHomeDirFunc: func() (string, error) { return "/home/testuser", nil },
		ReadDirFunc:     func(_ string) ([]os.DirEntry, error) { return nil, nil },
	}
	stub.RemoveAllFunc = func(path string) error {
		stub.RemovedAll = append(stub.RemovedAll, path)
		return nil
	}
	return stub
}

func (s *FileSystemStub) WithHomeDir(dir string) *FileSystemStub {
	s.UserHomeDirFunc = func() (string, error) { return dir, nil }
	return s
}

func (s *FileSystemStub) WithHomeDirError(err error) *FileSystemStub {
	s.UserHomeDirFunc = func() (string, error) { return "", err }
	return s
}

func (s *FileSystemStub) WithGlob(pattern string, matches []string) *FileSystemStub {
	prev := s.GlobFunc
	s.GlobFunc = func(p string) ([]string, error) {
		if p == pattern {
			return matches, nil
		}
		return prev(p)
	}
	return s
}

func (s *FileSystemStub) WithGlobError(pattern string, err error) *FileSystemStub {
	prev := s.GlobFunc
	s.GlobFunc = func(p string) ([]string, error) {
		if p == pattern {
			return nil, err
		}
		return prev(p)
	}
	return s
}

func (s *FileSystemStub) WithRemoveError(path string, err error) *FileSystemStub {
	prev := s.RemoveFunc
	s.RemoveFunc = func(p string) error {
		if p == path {
			return err
		}
		return prev(p)
	}
	return s
}

func (s *FileSystemStub) WithRemoveAllError(path string, err error) *FileSystemStub {
	prev := s.RemoveAllFunc
	s.RemoveAllFunc = func(p string) error {
		if p == path {
			return err
		}
		return prev(p)
	}
	return s
}

// WithPresentPath marks a path as present on the stub's Lstat calls. The
// returned FileInfo is minimal -- callers that need richer metadata should
// use [FileSystemStub.WithLstat] instead.
func (s *FileSystemStub) WithPresentPath(path string) *FileSystemStub {
	return s.WithLstat(path, &fakeFileInfo{name: path})
}

func (s *FileSystemStub) WithLstat(path string, info os.FileInfo) *FileSystemStub {
	prev := s.LstatFunc
	s.LstatFunc = func(p string) (os.FileInfo, error) {
		if p == path {
			return info, nil
		}
		return prev(p)
	}
	return s
}

func (s *FileSystemStub) WithLstatError(path string, err error) *FileSystemStub {
	prev := s.LstatFunc
	s.LstatFunc = func(p string) (os.FileInfo, error) {
		if p == path {
			return nil, err
		}
		return prev(p)
	}
	return s
}

func (s *FileSystemStub) WithReadDir(dir string, entries []os.DirEntry) *FileSystemStub {
	prev := s.ReadDirFunc
	s.ReadDirFunc = func(d string) ([]os.DirEntry, error) {
		if d == dir {
			return entries, nil
		}
		return prev(d)
	}
	return s
}

func (s *FileSystemStub) WithReadDirError(dir string, err error) *FileSystemStub {
	prev := s.ReadDirFunc
	s.ReadDirFunc = func(d string) ([]os.DirEntry, error) {
		if d == dir {
			return nil, err
		}
		return prev(d)
	}
	return s
}

func (s *FileSystemStub) Remove(path string) error                  { return s.RemoveFunc(path) }
func (s *FileSystemStub) RemoveAll(path string) error               { return s.RemoveAllFunc(path) }
func (s *FileSystemStub) Lstat(path string) (os.FileInfo, error)    { return s.LstatFunc(path) }
func (s *FileSystemStub) Glob(pattern string) ([]string, error)     { return s.GlobFunc(pattern) }
func (s *FileSystemStub) UserHomeDir() (string, error)              { return s.UserHomeDirFunc() }
func (s *FileSystemStub) ReadDir(dir string) ([]os.DirEntry, error) { return s.ReadDirFunc(dir) }

// FakeDirEntry implements [os.DirEntry] for testing.
type FakeDirEntry struct {
	EntryName  string
	EntryIsDir bool
}

func (e *FakeDirEntry) Name() string               { return e.EntryName }
func (e *FakeDirEntry) IsDir() bool                { return e.EntryIsDir }
func (e *FakeDirEntry) Type() fs.FileMode          { return 0 }
func (e *FakeDirEntry) Info() (fs.FileInfo, error) { return &fakeFileInfo{name: e.EntryName}, nil }

type fakeFileInfo struct{ name string }

func (f *fakeFileInfo) Name() string       { return f.name }
func (f *fakeFileInfo) Size() int64        { return 0 }
func (f *fakeFileInfo) Mode() fs.FileMode  { return 0 }
func (f *fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f *fakeFileInfo) IsDir() bool        { return false }
func (f *fakeFileInfo) Sys() any           { return nil }
