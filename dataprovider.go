//This package provides the API and basic tools for data providers, also known as virtual filesystem, in go.
package vfs

import (
	"io"
	"os"
)

// The DataProvider interface is the core contract to provide access to hierarchical structures using a compound
// key logic. This is an abstract of way of the design thinking behind a filesystem.
//
// Design decisions
//
// There are the following opinionated decisions:
//
//  * It is an Interface, because it cannot be expected to have a reasonable code reusage between implementations but we
//    need a common behavior.
//
//  * It contains both read and write contracts, because a distinction between read-only and write-only and filesystems
//    with both capabilities are edge cases. Mostly there will be implementations which provides each combination due to
//    their permission handling.
//
//  * Most implementations do not provide a transactional contract, which is represented through the optional
//    TransactionableDataProvider.
//
//  * It is not specified, if a DataProvider is thread safe and entirely implementation specific.
//
type DataProvider interface {
	// Opens the given resource for reading. May optionally also implement os.Seeker
	Read(path Path) (io.ReadCloser, error)

	// Opens the given resource for writing. Removes and recreates the file. May optionally also implement os.Seeker.
	Write(path Path) (io.WriteCloser, error)

	// Deletes a path entry and all contained children. It is not considered an error to delete a non-existing resource.
	Delete(path Path) error

	// Reads Attributes. Every implementation must support ResourceInfo
	ReadAttrs(path Path, dest interface{}) error

	// Writes Attributes. This is an optional implementation and may simply return OperationNotSupportedError.
	WriteAttrs(path Path, src interface{}) error

	// Reads the contents of a directory.
	ReadDir(path Path) (DirEntList, error)

	// Please close when Done
	io.Closer
}

// The Scanner contract is used to populate a pointer to a struct to get specific meta data out of an
// entry.
type Scanner interface {
	// Scan supports at least data into *ResourceInfo
	Scan(dest interface{}) error
}

// A DirEntList is a collection of (potentially lazy loaded) directory entries.
// E.g. the entire query may be even delayed until the first ForEach query, so that the scanner takes the given
// type and optimizes its remote query (e.g. by including or excluding required fields).
type DirEntList interface {
	// Loops over the result set and is invoked for each entry.
	ForEach(each func(scanner Scanner) error) error

	// Estimated amount of entries. Is -1 if unknown and you definitely have to use ForEach to collect the result.
	// However in the meantime, Size may deliver a more correct estimation.
	Size() int64

	// Please close when done
	io.Closer
}

// A ResourceInfo represents the default meta data set which must be supported by all implementations.
// However each implementation may also support other metadata as well.
type ResourceInfo struct {
	Name    string      // The local name of this resource
	Size    int64       // length in bytes for regular files of the primary data stream; system-dependent for others
	Mode    os.FileMode // file mode bits. Mode.IsDir and Mode.IsRegular are your friends.
	ModTime int64       // modification time in milliseconds since epoch 1970.
}
