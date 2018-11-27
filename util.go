package vfs

import (
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
