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

// A utility method which just loops over a ResultSet
func ForEach(provider DataProvider, query *Query, apply func(scanner AttributesScanner) (next bool, err error)) error {
	res, err := provider.Query(query)
	if err != nil {
		return err
	}
	defer silentClose(res)
	for res.Next() {
		callNext, err := apply(res)
		if err != nil {
			return err
		}
		if !callNext {
			return nil
		}
	}
	return nil
}

// A utility method to simply list a Query Result as ResourceInfo, which is supported by all DataProviders
func List(provider DataProvider, query *Query) ([]*ResourceInfo, error) {
	list := make([]*ResourceInfo, 0)
	err := ForEach(provider, query, func(scanner AttributesScanner) (next bool, err error) {
		row := &ResourceInfo{}
		err = scanner.Scan(row)
		if err != nil {
			return
		}
		list = append(list, row)
		next = true
		return
	})
	if err != nil {
		return list, err
	}
	return list, nil
}
