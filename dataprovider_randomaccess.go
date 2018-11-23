package vfs

import "io"

// ReadWriteSeeker is the interface that groups the basic Read, Write, Seek and Close methods.
type RandomAccessor interface {
	io.Reader
	io.Writer
	io.Seeker
	io.Closer
}

// A RandomAccessDataProvider is an optional DataProvider which allows efficient in-place modification and delta
// updates for a Resource.
type RandomAccessDataProvider interface {
	// Opens the resource without truncation. Initial position is at offset 0. If the resource can be opened
	// concurrently or if the modifications are immediately visible is implementation specific.
	Modify(path Path) (RandomAccessor, error)

	DataProvider
}
