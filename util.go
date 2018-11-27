package vfs

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"math"
)

// tries to close and prints silently the closer in case of an error
func silentClose(closer io.Closer) {
	err := closer.Close()
	if err != nil {
		log.Printf("failed to close: %v\n", err)
	}
}

// ReadDir is utility method to simply list a directory listing as ResourceInfo, which is supported by all DataProviders
func ReadDir(provider DataProvider, path Path) ([]*ResourceInfo, error) {
	res, err := provider.ReadDir(path)
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

// A WalkClosure is invoked for each entry in Walk, as long as no error is returned and entries are available.
type WalkClosure func(path Path, info *ResourceInfo, err error) error

// Walk recursively goes down the entire path hierarchy starting at the given path
func Walk(provider DataProvider, path Path, each WalkClosure) error {
	res, err := provider.ReadDir(path)
	if err != nil {
		return err
	}

	err = res.ForEach(func(scanner Scanner) error {
		tmp := &ResourceInfo{}
		err := scanner.Scan(tmp)
		if err != nil {
			each(path, nil, err)
			return err
		}

		//delegate call
		err = each(path.Child(tmp.Name), tmp, nil)

		if tmp.Mode.IsDir() {
			return Walk(provider, path.Child(tmp.Name), each)
		}
		return nil
	})
	return nil
}

// ReadDirs fully reads the given directory recursively
func ReadDirs(provider DataProvider, path Path) ([]*PathEntry, error) {
	res := make([]*PathEntry, 0)
	err := Walk(provider, path, func(path Path, info *ResourceInfo, err error) error {
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

// A PathEntry simply provides a Path and the related ResourceInfo
type PathEntry struct {
	Path     Path
	Resource *ResourceInfo
}

// ReadFully loads the entire resource into memory
func ReadFully(provider DataProvider, path Path) ([]byte, error) {
	reader, err := provider.Read(path)
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
