package vfs

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
)

var _ DataProvider = (*FilesystemDataProvider)(nil)

// A Provider which works with the local filesystem and optionally supports a local filename prefix, e.g.
// to just provide a local folder instead of the entire root. See also Resolve.
type FilesystemDataProvider struct {
	Prefix string
}

// Resolve creates a platform specific filename from the given invariant path by adding the Prefix and using
// the platform specific name separator
func (p *FilesystemDataProvider) Resolve(path Path) string {
	if len(p.Prefix) == 0 {
		if runtime.GOOS == "windows" {
			return filepath.Join(path.Names()...)
		} else {
			return path.String()
		}

	} else {
		return filepath.Join(p.Prefix, filepath.Join(path.Names()...))
	}
}

func (p *FilesystemDataProvider) MkDirs(path Path) error {
	return os.MkdirAll(p.Resolve(path), os.ModePerm)
}

func (p *FilesystemDataProvider) Read(path Path) (io.ReadCloser, error) {
	return os.Open(p.Resolve(path))
}

func (p *FilesystemDataProvider) Write(path Path) (io.WriteCloser, error) {
	return os.Create(p.Resolve(path))
}

func (p *FilesystemDataProvider) Delete(path Path) error {
	return os.RemoveAll(p.Resolve(path))
}

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
	} else {
		return &UnsupportedAttributesError{Data: dest}
	}
}

func (p *FilesystemDataProvider) WriteAttrs(path Path, src interface{}) error {
	return &UnsupportedOperationError{Message: "WriteAttrs"}
}

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
	} else {
		return &UnsupportedAttributesError{Data: dest}
	}
}
