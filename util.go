package vfs

import (
	"io"
	"log"
)

// tries to close and prints silently the closer
func silentClose(closer io.Closer) {
	err := closer.Close()
	if err != nil {
		log.Printf("failed to close: %v\n", err)
	}
}

// A utility method to simply list a Query Result as ResourceInfo, which is supported by all DataProviders
func ListOf(provider DataProvider, path Path) ([]*ResourceInfo, error) {
	list := make([]*ResourceInfo, 0)
	res, err := provider.ReadDir(path)
	if err != nil {
		return list, err
	}
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
