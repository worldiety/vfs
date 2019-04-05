package vfs

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
)

var _ FileSystem = (*LocalFileSystem)(nil)

// A LocalFileSystem just works with the local filesystem.
type LocalFileSystem struct {
}

// Link details: see FileSystem#Link
func (p *LocalFileSystem) Link(oldPath string, newPath string, mode LinkMode, flags int32) error {
	switch mode {
	case SymLink:
		return os.Symlink(p.Resolve(Path(oldPath)), p.Resolve(Path(newPath)))
	case HardLink:
		return os.Link(p.Resolve(Path(oldPath)), p.Resolve(Path(newPath)))
	default:
		return NewErr().UnsupportedOperation("Mode is unsupported: " + strconv.Itoa(int(mode)))

	}
}

// Resolve creates a platform specific filename from the given invariant path by adding the Prefix and using
// the platform specific name separator. If AllowRelativePaths is false (default), .. will be silently ignored.
func (p *LocalFileSystem) Resolve(path Path) string {
	//TODO what about windows? Does \c:\a\b work?
	return string(filepath.Separator) + filepath.Join(path.Names()...)
}

// Rename details: see FileSystem#Rename
func (p *LocalFileSystem) Rename(oldPath string, newPath string) error {
	err := os.Rename(p.Resolve(Path(oldPath)), p.Resolve(Path(newPath)))
	if err != nil {
		//perhaps the backend does not support the rename if target already exists
		err2 := p.Delete(newPath)
		if err2 != nil {
			//intentionally ignore err2 and return original failure
			return err
		}
		//retry again
		err3 := os.Rename(p.Resolve(Path(oldPath)), p.Resolve(Path(newPath)))
		if err3 != nil {
			//intentionally ignore err3 and return original failure
			return err
		}
	}
	return nil
}

// MkDirs details: see FileSystem#MkDirs
func (p *LocalFileSystem) MkDirs(path string) error {
	return os.MkdirAll(p.Resolve(Path(path)), os.ModePerm)
}

// Open details: see FileSystem#Open
func (p *LocalFileSystem) Open(ctx context.Context, flag int, perm os.FileMode, path string) (Resource, error) {
	if flag == os.O_RDONLY {
		return os.OpenFile(p.Resolve(Path(path)), flag, 0)
	}
	file, err := os.OpenFile(p.Resolve(Path(path)), flag, perm)
	if _, ok := err.(*os.PathError); ok {
		//try to recreate parent folder
		err2 := p.MkDirs(Path(path).Parent().String())
		if err2 != nil {
			//suppress err2 intentionally and return the original failure
			return nil, err
		}
		// mkdir is fine, retry again
		file, err = os.OpenFile(p.Resolve(Path(path)), flag, perm)
		if err != nil {
			return nil, err
		}
	}
	return file, nil

}

// Delete details: see FileSystem#Delete
func (p *LocalFileSystem) Delete(path string) error {
	return os.RemoveAll(p.Resolve(Path(path)))
}

// ReadAttrs details: see FileSystem#ReadAttrs
func (p *LocalFileSystem) ReadAttrs(path string, dest interface{}) error {
	if out, ok := dest.(ResourceInfo); ok {
		info, err := os.Stat(p.Resolve(Path(path)))
		if err != nil {
			return err
		}
		out.SetName(info.Name())
		out.SetMode(info.Mode())
		out.SetModTime(info.ModTime().UnixNano() / 1e6)
		out.SetSize(info.Size())
		return nil
	}
	return NewErr().UnsupportedAttributes("ReadAttrs", dest)

}

// WriteAttrs details: see FileSystem#WriteAttrs
func (p *LocalFileSystem) WriteAttrs(path string, src interface{}) error {
	return NewErr().UnsupportedOperation("WriteAttrs")
}

// ReadDir details: see FileSystem#ReadDir
func (p *LocalFileSystem) ReadDir(path string, options interface{}) (DirEntList, error) {
	list, err := ioutil.ReadDir(p.Resolve(Path(path)))
	if err != nil {
		return nil, err
	}
	return NewDirEntList(int64(len(list)), func(idx int64, out ResourceInfo) error {
		out.SetName(list[int(idx)].Name())
		out.SetMode(list[int(idx)].Mode())
		out.SetModTime(list[int(idx)].ModTime().UnixNano() / 1e6)
		out.SetSize(list[int(idx)].Size())
		return nil
	}), nil

}

// Close does nothing.
func (p *LocalFileSystem) Close() error {
	return nil
}
