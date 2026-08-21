package git

import (
	"errors"
	"io"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

// worktreeChange builds the object.Change describing one changed path between
// the HEAD tree and the working tree. It returns nil for a change that
// produces no diff (a file that vanished without ever being tracked).
func worktreeChange(wt *git.Worktree, headTree *object.Tree, path string, fs *git.FileStatus, s storer.EncodedObjectStorer) (*object.Change, error) {
	oldPath := path
	if fs.Staging == git.Renamed && fs.Extra != "" {
		oldPath = fs.Extra
	}
	from := object.ChangeEntry{Name: oldPath, Tree: headTree}
	if entry, err := headTree.FindEntry(oldPath); err == nil {
		from.TreeEntry = *entry
	}

	content, mode, err := readWorktreeFile(wt, path)
	if errors.Is(err, os.ErrNotExist) {
		if from.TreeEntry.Hash.IsZero() {
			return nil, nil
		}
		return &object.Change{From: from, To: object.ChangeEntry{}}, nil
	}
	if err != nil {
		return nil, err
	}
	hash, err := writeBlob(s, content)
	if err != nil {
		return nil, err
	}
	to := object.ChangeEntry{Name: path, Tree: headTree, TreeEntry: object.TreeEntry{Name: path, Mode: mode, Hash: hash}}
	if from.TreeEntry.Hash.IsZero() {
		return &object.Change{From: object.ChangeEntry{}, To: to}, nil
	}
	return &object.Change{From: from, To: to}, nil
}

// readWorktreeFile reads path from the worktree, reporting the file mode for
// the diff (regular, executable or symlink).
func readWorktreeFile(wt *git.Worktree, path string) ([]byte, filemode.FileMode, error) {
	fi, err := wt.Filesystem.Stat(path)
	if err != nil {
		return nil, 0, err
	}
	f, err := wt.Filesystem.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = f.Close() }()
	mode := filemode.Regular
	if fi.Mode()&os.ModeSymlink != 0 {
		mode = filemode.Symlink
	} else if fi.Mode().Perm()&0o111 != 0 {
		mode = filemode.Executable
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, 0, err
	}
	return data, mode, nil
}

// writeBlob stores data in the repository object store and returns its hash,
// so the diff's "to" side resolves like any tracked blob.
func writeBlob(s storer.EncodedObjectStorer, data []byte) (plumbing.Hash, error) {
	o := s.NewEncodedObject()
	o.SetType(plumbing.BlobObject)
	o.SetSize(int64(len(data)))
	w, err := o.Writer()
	if err != nil {
		return plumbing.ZeroHash, err
	}
	if _, err := w.Write(data); err != nil {
		return plumbing.ZeroHash, err
	}
	return s.SetEncodedObject(o)
}
