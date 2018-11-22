//This package provides the API and basic tools for data providers, also known as virtual filesystem, in go.
package vfs

import (
	"io"
	"os"
	"strings"
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
//  * c:/my/windows/folder
//  * https://mydomain.com:8080/myresource
//  * https://mydomain.com:8080/myresource?size=720p
//  * c:/my/ntfs/file:alternate-data-stream
//
// Invalid example paths
//  * missing/slash
//  * /extra/slash/
//  * \using\backslashes
//  * /c///using/slashes without content
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

// StartsWith tests whether the path begins with prefix.
func (p Path) StartsWith(prefix Path) bool {
	return strings.HasPrefix(string(p), string(prefix))
}

// EndsWith tests whether the path ends with prefix.
func (p Path) EndsWith(suffix Path) bool {
	return strings.HasSuffix(string(p), string(suffix))
}

// Splits the path by / and returns all segments as a simple string array.
func (p Path) Names() []string {
	tmp := strings.Split(string(p), "/")
	cleaned := make([]string, len(tmp))
	idx := 0
	for _, str := range tmp {
		str = strings.TrimSpace(str)
		if len(str) > 0 {
			cleaned[idx] = str
			idx++
		}
	}
	return cleaned[0:idx]
}

// Returns how many names are included in this path.
func (p Path) NameCount() int {
	return len(p.Names())
}

// Returns the name at the given index.
func (p Path) NameAt(idx int) string {
	return p.Names()[idx]
}

// Returns the last element in this path or the empty string if this path is empty.
func (p Path) Name() string {
	tmp := p.Names()
	if len(tmp) > 0 {
		return tmp[len(tmp)]
	}
	return ""
}

// Returns the parent path of this path.
func (p Path) Parent() Path {
	tmp := p.Names()
	if len(tmp) > 0 {
		return Path(strings.Join(tmp[:len(tmp)-1], "/"))
	}
	return ""
}

// String normalizes the slashes in Path
func (p Path) String() string {
	return "/" + strings.Join(p.Names(), "/")
}

// Returns a new Path append the child name
func (p Path) Child(name string) Path {
	if strings.HasPrefix(name, "/") {
		return Path(p.String() + name)
	}
	return Path(p.String() + "/" + name)
}

// Returns a path without the prefix
func (p Path) TrimPrefix(prefix Path) Path {
	tmp := "/" + strings.TrimPrefix(p.String(), prefix.String())
	return Path(tmp)
}

// Concates all paths together
func ConcatePaths(paths ...Path) Path {
	tmp := make([]string, 0)
	for _, path := range paths {
		for _, name := range path.Names() {
			tmp = append(tmp, name)
		}
	}
	return Path("/" + strings.Join(tmp, "/"))
}

// A Query is a special struct to allow efficient batch queries e.g. of remote directory listings. Such
// scenarios cannot be modelled properly using a single os.Stat call.
//
// A query is very limited and only supports limited projection and filter capabilities.
// The Match* criteria evaluates to a logical OR in the ResultSet. Multiple matches are explicitly allowed
// to support optimized queries of Attributes for multiple resources at once. This can avoid expensive remote lookups.
//
// Example
//
// An empty Query returns a ResultSet with access to the entire resource population. It is the logical equivalent of
// the SQL statement 'SELECT * FROM dataprovider'.
//
// A query with Fields returns a limited ResultSet, e.g. if a remote needs another call to get additional attributes.
// It is the logical equivalent of the SQL statement 'SELECT path FROM dataprovider'.
//
// A query with multiple paths only includes the affected elements. It is the logical equivalent of the SQL statement
// 'SELECT * FROM dataprovider WHERE path="/my/path" OR path="/my/other/path"'.
// A listing of contents matches against the parent path. It is the logical equivalent of the SQL statement
// 'SELECT * FROM dataprovider WHERE parent="/my/parent/path"'
//
type Query struct {
	// If empty, the projection (and later the Scanner) can always provide all available fields.
	Fields []string
	// All matches are evaluated using a logical OR
	MatchParents []Path
	// All matches are evaluated using a logical OR
	MatchPaths []Path
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
// Each call will add another Path with a logical OR
func (q *Query) MatchParent(path Path) *Query {
	q.MatchParents = append(q.MatchParents, path)
	return q
}

// MatchPath returns only those resources which have an exact matching path.
// Each call will add another Path with a logical OR
func (q *Query) MatchPath(path Path) *Query {
	q.MatchPaths = append(q.MatchPaths, path)
	return q
}

// Checks if any match path starts with the given prefix
func (q *Query) AnyMatchStartsWith(prefix Path) bool {
	for _, p := range q.MatchPaths {
		if p.StartsWith(prefix) {
			return true
		}
	}

	for _, p := range q.MatchParents {
		if p.StartsWith(prefix) {
			return true
		}
	}
	return false
}

// Checks if filter is empty
func (q *Query) IsFilterEmpty() bool {
	return len(q.MatchParents) == 0 && len(q.MatchPaths) == 0
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
	Query(query *Query) (ResultSet, error)

	// Opens the given resource for reading. May optionally also implement os.Seeker
	Read(path Path) (io.ReadCloser, error)

	// Opens the given resource for writing. Removes and recreates the file. May optionally also implement os.Seeker.
	Write(path Path) (io.WriteCloser, error)

	// Deletes a path entry and all contained children. It is not considered an error to delete a non-existing resource.
	Delete(path Path) error

	// Updates the attributes in a batch, if supported, otherwise returns an OperationNotSupportedError
	SetAttributes(attribs ...*Attributes) error
}

type Attributes struct {
	Path Path
	Data interface{}
}

// The AttributesScanner contract is used to populate a pointer to a struct to get specific meta data out of an
// entry.
type AttributesScanner interface {
	// Scan supports at least data into ResourceInfo
	Scan(dest interface{}) error
}

// A Cursor currently only provides a ForEach logic, because this is what most of the use cases require.
// We don't want that all implementations require a seekable cursor. Most use cases will require a list anyway.
// The ResultSet is always before the first entry.
type ResultSet interface {
	// Next prepares the next result row for reading with the Scan method.
	// It returns true on success, or false if there is no next result row or an error happened while preparing it.
	// Err should be consulted to distinguish between the two cases.
	//Every call to Scan, even the first one, must be preceded by a call to Next.
	Next() bool

	// Estimated amount of entries in the ResultSet
	Size() int64

	AttributesScanner
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
