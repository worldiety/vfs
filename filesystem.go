//Package vfs provides the API and basic tools for virtual filesystems.
package vfs

import (
	"context"
	"io"
	"os"
	"unicode/utf8"
)

// The PathSeparator is always / and platform independent
const PathSeparator = "/"

// The ForkSeparator is always ? and platform independent
const ForkSeparator = "?"

// The QuerySeparator is always ? and platform independent. Intentionally this is the same as the ForkSeparator.
const QuerySeparator = ForkSeparator

var unportableCharacters = []uint8{'*', '?', ':', '[', ']', '"', '<', '>', '|', '(', ')', '{', '}', '&', '\'', '!', '\\', ';', '$', 0x0}

// UnportableCharacter checks the given string for unsafe characters and returns the first index of occurrence or -1.
// This is important to exchange file names across different implementations, like windows, macos or linux.
// In general the following characters are considered unsafe *?:[]"<>|(){}&'!\;$ and chars <= 0x1F. As a developer
// you should check and avoid file path segments to contain any of these characters, especially because / or ? would
// clash with the path and fork separator. If the string is found not to be a valid utf8 sequence, 0 is returned.
func UnportableCharacter(str string) int {
	for i := 0; i < len(str); i++ {
		c := str[i]
		for _, avoid := range unportableCharacters {
			if c == avoid {
				return i
			}
		}
		if c <= 0x1F {
			return i
		}
	}
	if !utf8.ValidString(str) {
		return 0
	}
	return -1
}

// A LinkMode determines at creation time the way how links are created.
type LinkMode = int32



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
	// Resource Forks or Alternate Data Streams (or e.g. thumbnails from online resources) shall be addressed
	// using a ?. Example: /myfolder/test.png/thumb-jpg?720p. Note that the ? is also considered to be an unportable
	// and unsafe character but indeed it should never make it into a real local path. Instead the semantic is the
	// same as specified by an URI and represents the query and fragment part.
	//
	// Do not forget to close the resource, to avoid any leak.
	Open(ctx context.Context, flag int, perm os.FileMode, path string) (Resource, error)

	// Deletes a path entry and all contained children. It is not considered an error to delete a non-existing resource.
	// This non-posix behavior is introduced to guarantee two things:
	//   * the implementation shall ensure that races have more consistent effects
	//   * descending the tree and collecting all children is an expensive procedure and often unnecessary, especially
	//     in relational databases with foreign key constraints.
	Delete(path string) error

	// Reads Attributes. Every implementation must support the ResourceInfo interface. This allows structured
	// information to pass out without going through a serialization process using the fork logic.
	// Use cases which reads millions of attributes can be realized without any pressure on the memory subsystem.
	ReadAttrs(path string, dest interface{}) error

	// Writes Attributes. This is an optional implementation and may simply return UnsupportedOperationError.
	WriteAttrs(path string, src interface{}) error

	// ReadDir reads the contents of a directory. If path is not a directory, a ENOENT is returned.
	// options can be arbitrary and at least nil options must be supported, otherwise unsupported abstraction will
	// cause an EUNATTR to be returned. Using the options, the query to
	// retrieve the directory contents can be optimized, like required fields, sorting, page size, filter etc, if not
	// already passed through the path and its potential fork or query path. This
	// is especially important for online sources, because it also allows arbitrary queries, which are not
	// related to a path hierarchy at all, like tokens or post data.
	//
	// Implementations may support additional parameters like sorting or page sizes. These parameters should be
	// appended to the path with the QuerySeparator (URI-Style), e.g. /my/folder?type=jpg&sort=asc.
	ReadDir(path string, options interface{}) (DirEntList, error)

	// Tries to create the given path hierarchy. If path already denotes a directory nothing happens. If any path
	// segment already refers a resource, an error must be returned.
	MkDirs(path string) error

	// Rename moves a file from the old to the new path. If oldPath does not exist, ResourceNotFoundError is returned.
	// If newPath exists, it will be replaced.
	Rename(oldPath string, newPath string) error

	// Link can create different kind of links for paths. The kind of links is specified by mode (symbolic, reference
	// or hard). Symbolic is usually an os specific file containing the actual path to follow. Reference is like
	// a real copy but for copy-on-write systems this is nearly a no-op but changes will not be reflected on the source
	// file. In contrast to that, a hardlink points to the original source file, so two (or more) names for the same
	// resource.
	// The parameter flags is reserved (and unspecified) and can
	// be used to narrow behavior e.g. for the reflink syscall.
	Link(oldPath string, newPath string, mode int32, flags int32) error

	// Close when Done to release resources
	io.Closer
}

// A DirEntList is a collection of (potentially lazy loaded) directory entries.
// E.g. the entire query may be even delayed until the first next call.
type DirEntList interface {
	// Next prepares the next directory entry for reading with the Scan method.
	// It returns true on success, or false if there is no next entry or an error happened while preparing it.
	// Err should be consulted to distinguish between the two cases.
	//
	// Every call to Scan, even the first one, must be preceded by a call to Next.
	Next() bool

	// Err returns the first error, if any, that was encountered during iteration.
	Err() error

	// Scan supports at least reading data into a ResourceInfo interface.
	// Especially it is not guaranteed to fill or map into unknown
	// structs even if the field structure is identical. It is equivalent to FileSystem#ReadAttrs but instead
	// of performing an extra lookup, it shall use the already queried data from the iterator. This may also mean,
	// that depending on the query options (e.g. for performance reasons) some values are missing.
	Scan(dest interface{}) error

	// Estimated amount of entries. Is -1 if unknown and you definitely have to loop over to count.
	// However in the meantime, Size may deliver a more correct estimation.
	Size() int64

	// Please close when done
	io.Closer
}

// A ResourceInfo represents the default meta data set which must be supported by all implementations.
// However each implementation may also support other metadata as well.
type ResourceInfo interface {
	// SetName sets the local name of this resource
	SetName(name string)
	// Name returns the name of the resource
	Name() string
	// SetSize sets the length in bytes for regular files of the primary data stream; system-dependent for others
	SetSize(size int64)
	// Size returns the length in bytes for regular files of the primary data stream; system-dependent for others
	Size() int64
	// SetMode sets the file mode bits
	SetMode(mode os.FileMode)
	// Mode returns the file mode bits. Mode.IsDir and Mode.IsRegular are your friends.
	Mode() os.FileMode
	// SetModTime modification time in milliseconds since epoch 1970.
	SetModTime(time int64)
	// ModTime returns modification time in milliseconds since epoch 1970.
	ModTime() int64
}


//TODO how to perform cancellation and timeouts? context?
