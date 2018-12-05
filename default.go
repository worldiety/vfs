package vfs

import (
	"bytes"
	"fmt"
	"io"
	"math"
)

var prov DataProvider = &FilesystemDataProvider{}

// Default returns the root data provider. By default it is a vfs.FilesystemDataProvider. Consider to reconfigure it to
// a vfs.MountableDataProvider which allows arbitrary prefixes (also called mount points). Use it to configure and setup
// a virtualized filesystem structure for your app.
//
// Best practice
//
//  * Mount your static app data into /assets
//  * Implement variant and localization data at mount time not runtime, e.g.
//    /assets contains the data for a specific client with a specific locale
//    instead of a manual lookup e.g. in /assets/customer/DE_de. Keep your code
//    clean.
//  * Mount your user specific data into something like /media/local and
//    /media/ftp and /media/gphotos etc.
//
func Default() DataProvider {
	return prov
}

// SetDefault updates the default data provider. See also #Default()
func SetDefault(provider DataProvider) {
	prov = provider
}

// Read opens the given resource for reading. May optionally also implement os.Seeker. If called on a directory
// UnsupportedOperationError is returned. Delegates to Default()#Read.
func Read(path Path) (io.ReadCloser, error) {
	return Default().Read(path)
}

// Write opens the given resource for writing. Removes and recreates the file. May optionally also implement os.Seeker.
// If elements of the path do not exist, they are created implicitly. Delegates to Default()#Write.
func Write(path Path) (io.WriteCloser, error) {
	return Default().Write(path)
}

// Delete a path entry and all contained children. It is not considered an error to delete a non-existing resource.
// Delegates to Default()#Delete.
func Delete(path Path) error {
	return Default().Delete(path)
}

// ReadAttrs reads Attributes. Every implementation must support ResourceInfo. Delegates to Default()#ReadAttrs.
func ReadAttrs(path Path, dest interface{}) error {
	return Default().ReadAttrs(path, dest)
}

// WriteAttrs writes Attributes. This is an optional implementation and may simply return UnsupportedOperationError.
// Delegates to Default()#WriteAttrs.
func WriteAttrs(path Path, src interface{}) error {
	return Default().WriteAttrs(path, src)
}

// ReadDir reads the contents of a directory. Delegates to Default()#ReadDir.
func ReadDir(path Path) (DirEntList, error) {
	return Default().ReadDir(path)
}

// MkDirs tries to create the given path hierarchy. If path already denotes a directory nothing happens. If any path
// segment already refers a file, an error must be returned. Delegates to Default()#MkDirs.
func MkDirs(path Path) error {
	return Default().MkDirs(path)
}

// Rename moves a file from the old to the new path. If oldPath does not exist, ResourceNotFoundError is returned.
// If newPath exists, it will be replaced. Delegates to Default()#Rename.
func Rename(oldPath Path, newPath Path) error {
	return Default().Rename(oldPath, newPath)
}

// ReadDirEnt is utility method to simply list a directory listing as ResourceInfo, which is supported by all
// DataProviders.
func ReadDirEnt(path Path) ([]*ResourceInfo, error) {
	res, err := Default().ReadDir(path)
	if err != nil {
		return nil, err
	}
	// a little bit of premature optimization
	expectedEntries := 0
	if res.Size() > 0 {
		if res.Size() > math.MaxInt32 {
			return nil, fmt.Errorf("to many entries: %v", res.Size())
		}
		expectedEntries = int(res.Size())
	}
	list := make([]*ResourceInfo, expectedEntries)[0:0]
	err = res.ForEach(func(scanner Scanner) error {
		row := &ResourceInfo{}
		err = scanner.Scan(row)
		if err != nil {
			return err
		}
		list = append(list, row)
		return nil
	})

	if err != nil {
		return list, err
	}
	return list, nil
}

// ReadDirEntRecur fully reads the given directory recursively and returns entries with full qualified paths.
func ReadDirEntRecur(path Path) ([]*PathEntry, error) {
	res := make([]*PathEntry, 0)
	err := Walk(path, func(path Path, info *ResourceInfo, err error) error {
		if err != nil {
			return err
		}
		res = append(res, &PathEntry{path, info})
		return nil
	})
	if err != nil {
		return res, err
	}
	return res, nil
}

// A WalkClosure is invoked for each entry in Walk, as long as no error is returned and entries are available.
type WalkClosure func(path Path, info *ResourceInfo, err error) error

// Walk recursively goes down the entire path hierarchy starting at the given path
func Walk(path Path, each WalkClosure) error {
	res, err := Default().ReadDir(path)
	if err != nil {
		return err
	}

	err = res.ForEach(func(scanner Scanner) error {
		tmp := &ResourceInfo{}
		err := scanner.Scan(tmp)
		if err != nil {
			// the dev may decide to ignore errors and continue walking, e.g. due to permission denied
			shouldBreak := each(path, nil, err)
			if shouldBreak != nil {
				return shouldBreak
			}
			return nil

		}

		//delegate call
		err = each(path.Child(tmp.Name), tmp, nil)

		if tmp.Mode.IsDir() {
			return Walk(path.Child(tmp.Name), each)
		}
		return nil
	})
	return nil
}

// A PathEntry simply provides a Path and the related ResourceInfo
type PathEntry struct {
	Path     Path
	Resource *ResourceInfo
}

// ReadAll loads the entire resource into memory. Only use it, if you know that it fits into memory
func ReadAll(path Path) ([]byte, error) {
	reader, err := Default().Read(path)
	if err != nil {
		return nil, err
	}
	defer silentClose(reader)

	buf := &bytes.Buffer{}
	_, err = io.Copy(buf, reader)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// WriteAll just puts the given data into the path
func WriteAll(path Path, data []byte) (int, error) {
	writer, err := Default().Write(path)
	if err != nil {
		return 0, err
	}
	defer silentClose(writer)

	n, err := writer.Write(data)
	if err != nil {
		return n, err
	}
	if n != len(data) {
		return n, fmt.Errorf("provider %v.Write has violated the Write contract", Default())
	}
	return n, nil
}

// Stat simply allocates a ResourceInfo and reads it, which must be supported by all implementations.
func Stat(path Path) (*ResourceInfo, error) {
	info := &ResourceInfo{}
	err := Default().ReadAttrs(path, info)
	if err != nil {
		return info, err
	}
	return info, nil
}
