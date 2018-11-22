//This package provides the API and basic tools for data providers, also known as virtual filesystem, in go.
package vfs

import (
	"io"
	"os"
	"time"
)

// A Path must be unique in it's context and has the role of a composite key. It's segments are always separated using
// a slash, even if they denote paths from windows.
//
// Example
//
// Valid example paths
//
//  * /my/path/may/denote/a/file/or/folder
//  * /c/my/windows/folder
//  * /
//
// Invalid example paths
//  * missing/slash
//  * /extra/slash/
//  * \using\backslashes
//  * /c:/using/punctuations
//  * ../../using/relative/paths
//
// Design decisions
//
// There are the following opinionated decisions:
//  * In the context of a filesystem, this is equal to the full qualified name of a file entry.
//
//  * It is a string, because defacto all modern APIs are UTF-8 and web based. However there are also a lot of Unix or
//    Linux types which have different local encodings or just have an undefined byte sequence. Providers with such
//    requirements must support the path API through some kind of conversion and normalization, but they should also
//    provide an exact API using byte slices then.
//    One could also argue, that a string is a bad choice for Go, because of these corner case, potential invalid utf-8
//    sequences and suboptimal string allocations. But using just byte-slices by default would make a lot of things even
//    worse:
//       * You cannot simply compare byte slices in Go. You need to compare and acknowledge about a new standard.
//       * It can be expected that the developer using this library will certainly need a string representation which
//         will practically always cause additional allocations.
//       * Because a path is naturally always a string, you certainly want to use all the provided and standard string
//         handling infrastructures instead of reinventing your own.
//
//  * There are studies which claim that the average filename is between 11 and 15 characters long. Because we
//    want to optimize use cases like keeping 1 million file names in memory, using a fixed size 256 byte array would result
//    in a 17x overhead of memory usage: e.g. 17GiB instead of 1GiB of main memory. To save even more memory and lower
//    GC pressure, we do not use a slice of strings but just a pure string providing helper methods.
type Path string

// A Query is a special struct to allow efficient batch queries e.g. of remote directory listings. Such
// scenarios cannot be modelled properly using a single os.Stat call.
//
// A query is very limited and only supports limited projection and filter capabilities.
//
type Query struct {
	Fields       []string
	MatchParents []Path
	MatchPaths   []Path
}

// NewQuery allocates a new Query instance for a fluent API. An empty query must be supported and returns everything.
func NewQuery() *Query {
	return &Query{}
}

// Select limits the available fields for the DataProvider which may use these information to optimize it's performance.
// All DataProviders must support an empty projection and must silently ignore unknown or unsupported fields.
func (q *Query) Select(fields ...string) *Query {
	q.Fields = fields
	return q
}

// MatchParent returns only those resources which have an exact matching parent parent.
func (q *Query) MatchParent(path Path) *Query {
	q.MatchParents = append(q.MatchParents, path)
	return q
}

// MatchPath returns only those resources which have an exact matching path.
func (q *Query) MatchPath(path Path) *Query {
	q.MatchPaths = append(q.MatchPaths, path)
	return q
}

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
//  * Most implementations do not provide a transactional contract, however abstraction which do so, should only provide
//    their VFS contract through the Transaction interface.
//
type DataProvider interface {
	// The query method is used to acquire meta data
	Query(query *Query) (Cursor, error)

	// Opens the given resource for reading. May optionally also implement os.Seeker
	Read(path Path) (io.ReadCloser, error)

	// Opens the given resource for writing.
	Write(path Path) (io.WriteCloser, error)
}



type AttributesReader interface {
	Attributes(data interface{}) error
}

// A Cursor currently only provides a ForEach logic, because this is what most of the use cases require.
// We don't want that all implementations require a seekable cursor. Most use cases will require a list anyway.
type Cursor interface {
	// to support GC f
	ForEach(func(reader AttributesReader) (next bool, err error)) error
	io.Closer
}



// A ResourceInfo represents the default meta data set which must be supported by all implementations Query method.
// However each implementation may also support other types as well.
type ResourceInfo struct {
	Path    string      // The full qualified path of the resource
	Size    int64       // length in bytes for regular files of the primary data stream; system-dependent for others
	Mode    os.FileMode // file mode bits
	ModTime time.Time   // modification time
}
