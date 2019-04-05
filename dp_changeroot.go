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
func (f *ChRoot) Link(oldPath string, newPath string, mode LinkMode, flags int32) error {
	return f.Delegate.Link(f.Resolve(Path(oldPath)).String(), f.Resolve(Path(newPath)).String(), mode, flags)
}

// Resolve normalizes the given Path and inserts the prefix.
// We normalize our path, before adding the prefix to avoid breaking out of our root
func (f *ChRoot) Resolve(path Path) Path {
	return f.Prefix.Add(path.Normalize())
}

// Open details: see FileSystem#Open
func (f *ChRoot) Open(ctx context.Context, flag int, perm os.FileMode, path string) (Resource, error) {
	return f.Delegate.Open(
}

// Delete details: see FileSystem#Delete
func (f *ChRoot) Delete(path string) error {
	return f.Delegate.Delete(f.Resolve(Path(path)).String())
}

// ReadAttrs details: see FileSystem#ReadAttrs
func (f *ChRoot) ReadAttrs(path string, dest interface{}) error {
	return f.Delegate.ReadAttrs(f.Resolve(Path(path)).String(), dest)
}

// WriteAttrs details: see FileSystem#WriteAttrs
func (f *ChRoot) WriteAttrs(path string, src interface{}) error {
	return f.Delegate.WriteAttrs(f.Resolve(Path(path)).String(), src)
}

// ReadDir details: see FileSystem#ReadDir
func (f *ChRoot) ReadDir(path string, options interface{}) (DirEntList, error) {
	return f.Delegate.ReadDir(f.Resolve(Path(path)).String(), options)
}

// MkDirs details: see FileSystem#MkDirs
func (f *ChRoot) MkDirs(path string) error {
	return f.Delegate.MkDirs(f.Resolve(Path(path)).String())
}

// Rename details: see FileSystem#Rename
func (f *ChRoot) Rename(oldPath string, newPath string) error {
	return f.Delegate.Rename(f.Resolve(Path(oldPath)).String(), f.Resolve(Path(newPath)).String())
}

// Close details: see FileSystem#Close
func (f *ChRoot) Close() error {
	return f.Delegate.Close()
}
