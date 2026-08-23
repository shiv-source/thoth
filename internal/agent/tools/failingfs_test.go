package tools

import (
	"errors"
	"io/fs"

	agenttools "github.com/shiv-source/thoth/agent/tools"
)

// failingFS wraps an agenttools.FS and fails a selectable operation, so the
// internal ops tools' error branches are exercised. fail names the operation
// ("" fails nothing).
type failingFS struct {
	agenttools.FS
	fail string
}

func (f failingFS) ReadFile(name string) ([]byte, error) {
	if f.fail == "ReadFile" {
		return nil, errors.New("read: I/O error")
	}
	return f.FS.ReadFile(name)
}

func (f failingFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if f.fail == "ReadDir" {
		return nil, errors.New("read dir: I/O error")
	}
	return f.FS.ReadDir(name)
}

func (f failingFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	if f.fail == "WriteFile" {
		return errors.New("write: no space left on device")
	}
	return f.FS.WriteFile(name, data, perm)
}

func (f failingFS) MkdirAll(path string, perm fs.FileMode) error {
	if f.fail == "MkdirAll" {
		return errors.New("mkdir: permission denied")
	}
	return f.FS.MkdirAll(path, perm)
}

func (f failingFS) Rename(oldPath, newPath string) error {
	if f.fail == "Rename" {
		return errors.New("rename: cross-device link")
	}
	return f.FS.Rename(oldPath, newPath)
}
