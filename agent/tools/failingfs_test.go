package tools

import (
	"errors"
	"io/fs"
)

// failingFS wraps a FS and fails a selectable mutating operation, so the file
// tools' write-path error branches are exercised without touching real files.
// fail names the operation that must fail ("" fails nothing; anything else
// fails every call of that operation).
type failingFS struct {
	FS
	fail string
}

func (f failingFS) ReadFile(name string) ([]byte, error) {
	if f.fail == "ReadFile" {
		return nil, errors.New("read: I/O error")
	}
	return f.FS.ReadFile(name)
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

func (f failingFS) Remove(name string) error {
	if f.fail == "Remove" {
		return errors.New("remove: permission denied")
	}
	return f.FS.Remove(name)
}
