//Package vfs provides the API and basic tools for virtual filesystems.
package vfs

import (
	"io"
	"os"
)

// A LinkMode determines at creation time the way how links are created.
type LinkMode int32

const (
	// SymLink writes the actual path into the file which is evaluated at runtime.
	SymLink LinkMode = 0
	// RefLink behaves like a file copy but shares underlying data structures to be more efficient.
	// While multiple hardlinks always point to the same file and changes are reflected vice versa, a reflink
	// really behaves like a copy using COW techniques.
	RefLink LinkMode = 1
	// HardLink is the entry point to a block of data. The data becomes inaccessible if there are no
	// more hard links.
	HardLink LinkMode = 2
)

// A Resource is an abstract accessor to read or write bytes. In general, not all methods are supported, either
// due to the way the resources have been opened or because of the underlying implementation. In such cases
// the affected method will always return an *UnsupportedOperationError.
type Resource interface {
	// ReadAt reads len(b) bytes from the File starting at byte offset off. It returns the number of bytes read and
	// the error, if any. ReadAt always returns a non-nil error when n < len(b). At end of file, that error is io.EOF.
	//
	// This method is explicitly thread safe, as long no overlapping
	// writes or reads to the given offset are happening. This is
	// especially useful in multi threaded scenarios to perform
	// concurrent changes on a single opened resource (see POSIX pread).
	ReadAt(b []byte, off int64) (n int, err error)
	io.Reader

	// WriteAt writes len(b) bytes to the File starting at byte offset off.
	// It returns the number of bytes written and an error, if any.
	// WriteAt returns a non-nil error when n != len(b).
	//
	// This method is explicitly thread safe, as long no overlapping
	// writes or reads to the given offset are happening. This is
	// especially useful in multi threaded scenarios to perform
	// concurrent changes on a single opened resource (see POSIX pwrite).
	WriteAt(b []byte, off int64) (n int, err error)
	io.Writer
	io.Seeker
	io.Closer
}

// The FileSystem interface is the core contract to provide access to hierarchical structures using a compound
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
//    TransactionableFileSystem.
//
//  * It is not specified, if a FileSystem is thread safe. However every
//    implementation should be as thread safe as possible, similar to the POSIX filesystem specification.
//
type FileSystem interface {
	// Open is the general read or write call. It opens the named resource with specified flags (O_RDONLY etc.)
	// and perm (before umask), if applicable.
	// If successful, methods on the returned File can be used for I/O.
	// If there is an error, the trace will also contain a *PathError.
	// Implementations have to create parent directories, if those do not exist. However if any existing
	// path segment denotes already a resource, the resource is not deleted and an error is returned instead.
	//
	// Resource Forks or Alternate Data Streams (or e.g. thumbnails from online resources) should be simply addressed
	// using a /. Example: /myfolder/test.png/thumb-jpg/720p. Note that this is path wise basically indistinguishable
	// from a folder and a regular file name (which is logically correct).
	// To make your lookups easier, you may use some kind of magic identifier
	// like rsc or $<some id> but you should generally avoid : as used by windows because it breaks
	// the entire path semantics and conflicts with the Unix path separator.
	Open(path Path, flag int, perm os.FileMode) (Resource, error)

	// Deletes a path entry and all contained children. It is not considered an error to delete a non-existing resource.
	Delete(path Path) error

	// Reads Attributes. Every implementation must support *ResourceInfo
	ReadAttrs(path Path, dest interface{}) error

	// Writes Attributes. This is an optional implementation and may simply return UnsupportedOperationError.
	WriteAttrs(path Path, src interface{}) error

	// ReadDir reads the contents of a directory. If path is not a directory, a ResourceNotFoundError is returned.
	// options can be arbitrary and at least nil options must be supported, otherwise unsupported abstraction will
	// cause an *UnsupportedAttributesError to be returned. Using the options, the query to
	// retrieve the directory contents can be optimized, like required fields, sorting, page size, filter etc. This
	// is especially important for online sources, because it also allows arbitrary queries, which are not
	// related to a path hierarchy at all.
	ReadDir(path Path, options interface{}) (DirEntList, error)

	// Tries to create the given path hierarchy. If path already denotes a directory nothing happens. If any path
	// segment already refers a resource, an error must be returned.
	MkDirs(path Path) error

	// Rename moves a file from the old to the new path. If oldPath does not exist, ResourceNotFoundError is returned.
	// If newPath exists, it will be replaced.
	Rename(oldPath Path, newPath Path) error

	// Link can create different kind of links for paths. The kind of links is specified by mode.
	// The parameter flags is reserved (and unspecified) and can
	// be used to narrow behavior e.g. for the reflink syscall.
	Link(oldPath Path, newPath Path, mode LinkMode, flags int32) error

	// Close when Done to release resources
	io.Closer
}

// A DirEntList is a collection of (potentially lazy loaded) directory entries.
// E.g. the entire query may be even delayed until the first next call.
type DirEntList interface {
	// Next prepares the next directory entry for reading with the Scan method.
	// It returns true on success, or false if there is no next entry or an error happened while preparing it.
	// Err should be consulted to distinguish between the two cases.

	// Every call to Scan, even the first one, must be preceded by a call to Next.
	Next() bool

	// Err returns the first error, if any, that was encountered during iteration.
	Err() error

	// Scan supports at least reading data into *ResourceInfo.
	// Especially it is not guaranteed to fill or map into unknown
	// structs even if the field structure is identical.
	Scan(dest interface{}) error

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

// Equals checks for equality with another PathEntry
func (e *ResourceInfo) Equals(other interface{}) bool {
	if e == nil || other == nil {
		return false
	}
	if o, ok := other.(*ResourceInfo); ok {
		return o.Name == e.Name && o.Size == e.Size && o.ModTime == e.ModTime && o.Mode == e.Mode
	}
	return false
}
