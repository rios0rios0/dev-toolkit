package doubles

import (
	"io/fs"
	"os"
	"time"
)

// FileSystemStub is a test double for system.FileSystem with configurable behavior.
type FileSystemStub struct {
	RemoveFunc      func(path string) error
	GlobFunc        func(pattern string) ([]string, error)
	UserHomeDirFunc func() (string, error)
	ReadDirFunc     func(dir string) ([]os.DirEntry, error)
}

func NewFileSystemStub() *FileSystemStub {
	return &FileSystemStub{
		RemoveFunc:      func(_ string) error { return nil },
		GlobFunc:        func(_ string) ([]string, error) { return nil, nil },
		UserHomeDirFunc: func() (string, error) { return "/home/testuser", nil },
		ReadDirFunc:     func(_ string) ([]os.DirEntry, error) { return nil, nil },
	}
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

func (s *FileSystemStub) Remove(path string) error                    { return s.RemoveFunc(path) }
func (s *FileSystemStub) Glob(pattern string) ([]string, error)       { return s.GlobFunc(pattern) }
func (s *FileSystemStub) UserHomeDir() (string, error)                { return s.UserHomeDirFunc() }
func (s *FileSystemStub) ReadDir(dir string) ([]os.DirEntry, error)   { return s.ReadDirFunc(dir) }

// FakeDirEntry implements os.DirEntry for testing.
type FakeDirEntry struct {
	EntryName  string
	EntryIsDir bool
}

func (e *FakeDirEntry) Name() string               { return e.EntryName }
func (e *FakeDirEntry) IsDir() bool                { return e.EntryIsDir }
func (e *FakeDirEntry) Type() fs.FileMode          { return 0 }
func (e *FakeDirEntry) Info() (fs.FileInfo, error)  { return &fakeFileInfo{name: e.EntryName}, nil }

type fakeFileInfo struct{ name string }

func (f *fakeFileInfo) Name() string      { return f.name }
func (f *fakeFileInfo) Size() int64       { return 0 }
func (f *fakeFileInfo) Mode() fs.FileMode { return 0 }
func (f *fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f *fakeFileInfo) IsDir() bool       { return false }
func (f *fakeFileInfo) Sys() any          { return nil }
