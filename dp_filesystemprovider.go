package vfs

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
)

var _ DataProvider = (*FilesystemDataProvider)(nil)

// A FilesystemDataProvider just works with the local filesystem and optionally supports a local filename prefix, e.g.
// to just provide a subset instead of the entire root. See also Resolve.
type FilesystemDataProvider struct {
	// The Prefix is always added to any given path, so you can create artificial roots.
	Prefix string
}

// Resolve creates a platform specific filename from the given invariant path by adding the Prefix and using
// the platform specific name separator. If AllowRelativePaths is false (default), .. will be silently ignored.
func (p *FilesystemDataProvider) Resolve(path Path) string {
	if len(p.Prefix) == 0 {
		if runtime.GOOS == "windows" {
			return filepath.Join(path.Names()...)
		}
		return path.String()

	}
	// security feature: we normalize our path, before adding the prefix to avoid breaking out of our root
	path = path.Normalize()
	return filepath.Join(p.Prefix, filepath.Join(path.Names()...))
}

// Rename details: see DataProvider#Rename
func (p *FilesystemDataProvider) Rename(oldPath Path, newPath Path) error {
	err := os.Rename(p.Resolve(oldPath), p.Resolve(newPath))
	if err != nil {
		//perhaps the backend does not support the rename if target already exists
		err2 := p.Delete(newPath)
		if err2 != nil {
			//intentionally ignore err2 and return original failure
			return err
		}
		//retry again
		err3 := os.Rename(p.Resolve(oldPath), p.Resolve(newPath))
		if err3 != nil {
			//intentionally ignore err3 and return original failure
			return err
		}
	}
	return nil
}

// MkDirs details: see DataProvider#MkDirs
func (p *FilesystemDataProvider) MkDirs(path Path) error {
	return os.MkdirAll(p.Resolve(path), os.ModePerm)
}

// Read details: see DataProvider#Read
func (p *FilesystemDataProvider) Read(path Path) (io.ReadCloser, error) {
	return os.Open(p.Resolve(path))
}

// Write details: see DataProvider#Write
func (p *FilesystemDataProvider) Write(path Path) (io.WriteCloser, error) {
	file, err := os.Create(p.Resolve(path))
	if _, ok := err.(*os.PathError); ok {
		//try to recreate parent folder
		err2 := p.MkDirs(path.Parent())
		if err2 != nil {
			//suppress err2 intentionally and return the original failure
			return nil, err
		}
		// mkdir is fine, retry again
		file, err = os.Create(p.Resolve(path))
		if err != nil {
			return nil, err
		}
	}
	return file, nil
}

// Delete details: see DataProvider#Delete
func (p *FilesystemDataProvider) Delete(path Path) error {
	return os.RemoveAll(p.Resolve(path))
}

// ReadAttrs details: see DataProvider#ReadAttrs
func (p *FilesystemDataProvider) ReadAttrs(path Path, dest interface{}) error {
	if out, ok := dest.(*ResourceInfo); ok {
		info, err := os.Stat(p.Resolve(path))
		if err != nil {
			return err
		}
		out.Name = info.Name()
		out.Mode = info.Mode()
		out.ModTime = info.ModTime().UnixNano() / 1e6
		out.Size = info.Size()
		return nil
	}
	return &UnsupportedAttributesError{Data: dest}

}

// WriteAttrs details: see DataProvider#WriteAttrs
func (p *FilesystemDataProvider) WriteAttrs(path Path, src interface{}) error {
	return &UnsupportedOperationError{Message: "WriteAttrs"}
}

// ReadDir details: see DataProvider#ReadDir
func (p *FilesystemDataProvider) ReadDir(path Path) (DirEntList, error) {
	list, err := ioutil.ReadDir(p.Resolve(path))
	if err != nil {
		return nil, err
	}
	return &fileInfoDirEntList{list}, nil
}

// Close does nothing
func (p *FilesystemDataProvider) Close() error {
	return nil
}

//
type fileInfoDirEntList struct {
	list []os.FileInfo
}

func (l *fileInfoDirEntList) ForEach(each func(scanner Scanner) error) error {
	scanner := &fileScanner{}
	for _, info := range l.list {
		scanner.info = info
		err := each(scanner)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *fileInfoDirEntList) Size() int64 {
	return int64(len(l.list))
}

//does nothing
func (l *fileInfoDirEntList) Close() error {
	return nil
}

//
type fileScanner struct {
	info os.FileInfo
}

func (f *fileScanner) Scan(dest interface{}) error {
	if out, ok := dest.(*ResourceInfo); ok {
		out.Name = f.info.Name()
		out.Mode = f.info.Mode()
		out.ModTime = f.info.ModTime().UnixNano() / 1e6
		out.Size = f.info.Size()
		return nil
	}
	return &UnsupportedAttributesError{Data: dest}
}
