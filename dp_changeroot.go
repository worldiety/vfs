package vfs

import (
	"os"
)

var _ FileSystem = (*ChRoot)(nil)

// A ChRoot is a filesystem which is basically a poor man's chroot which just adds a prefix to all endpoints and
// delegates all calls. A security note: the path is normalized before prefixing, so that path based attacks
// using .. are not possible.
type ChRoot struct {
	// The Prefix which is added before delegating
	Prefix Path
	// The Delegate to call with the prefixed path
	Delegate FileSystem
}

// Link details: see FileSystem#Link
func (f *ChRoot) Link(oldPath Path, newPath Path, mode LinkMode, flags int32) error {
	return f.Delegate.Link(f.Resolve(oldPath), f.Resolve(newPath), mode, flags)
}

// Resolve normalizes the given Path and inserts the prefix.
// We normalize our path, before adding the prefix to avoid breaking out of our root
func (f *ChRoot) Resolve(path Path) Path {
	return f.Prefix.Add(path.Normalize())
}

// Open details: see FileSystem#Open
func (f *ChRoot) Open(path Path, flag int, perm os.FileMode) (Resource, error) {
	return f.Delegate.Open(f.Resolve(path), flag, perm)
}

// Delete details: see FileSystem#Delete
func (f *ChRoot) Delete(path Path) error {
	return f.Delegate.Delete(f.Resolve(path))
}

// ReadAttrs details: see FileSystem#ReadAttrs
func (f *ChRoot) ReadAttrs(path Path, dest interface{}) error {
	return f.Delegate.ReadAttrs(f.Resolve(path), dest)
}

// WriteAttrs details: see FileSystem#WriteAttrs
func (f *ChRoot) WriteAttrs(path Path, src interface{}) error {
	return f.Delegate.WriteAttrs(f.Resolve(path), src)
}

// ReadDir details: see FileSystem#ReadDir
func (f *ChRoot) ReadDir(path Path, options interface{}) (DirEntList, error) {
	return f.Delegate.ReadDir(f.Resolve(path), options)
}

// MkDirs details: see FileSystem#MkDirs
func (f *ChRoot) MkDirs(path Path) error {
	return f.Delegate.MkDirs(f.Resolve(path))
}

// Rename details: see FileSystem#Rename
func (f *ChRoot) Rename(oldPath Path, newPath Path) error {
	return f.Delegate.Rename(f.Resolve(oldPath), f.Resolve(newPath))
}

// Close details: see FileSystem#Close
func (f *ChRoot) Close() error {
	return f.Delegate.Close()
}
