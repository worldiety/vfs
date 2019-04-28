package vfs

import (
	"github.com/worldiety/xobj"
	"io"
)

// ReadSeekCloser is the interface that groups the basic Read, Seek and Close methods.
// The Seek contract is optional and implementations may reject this operation permanently with ENOSYS error.
type ReadSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

// WriteSeekCloser is the interface that groups the basic Write, Seek and Close methods.
// The Seek contract is optional and implementations may reject this operation permanently with ENOSYS error.
type WriteSeekCloser interface {
	io.Writer
	io.Seeker
	io.Closer
}

// A DataDriver abstracts a kind of hierarchical concept of accessing blobs, like cloud storage platforms or
// local filesystems. Implementations may always provide more interface contracts, so check each implementations
// for details, especially for things like authentication etc.
type DataDriver interface {
	// Read opens a blob for read-only. There are no consistency guarantees. Resource forks like thumbnails should be
	// either addressed with path or uri style:
	//  Path style: /my/folder/test.png/thumb-jpg/720p
	//  URI style: /my/folder/test.png?format=jpg&res=720p
	Read(ctx Cancelable, path string) (ReadSeekCloser, error)

	// Write opens a blob to read and write. There are no consistency guarantees. Existing files are not removed, but
	// non-existing files are automatically created. The seeker position is always at offset 0.
	Write(ctx Cancelable, path string) (WriteSeekCloser, error)

	// Delete removes a batch of path entries and all contained children.
	// It is not considered an error to delete a non-existing resource. If deletion of a single entry fails,
	// the result is undefined, e.g. n-1 entries may be already removed or none at all. However the implementation
	// must return an error in any case.
	Delete(ctx Cancelable, paths *StrList) error

	// ReadAttrs reads a batch of path entries. If a single entry in a batch cannot be read, the result is undefined.
	// Some implementations may provide all other entries, with a nil entry at the affected position,
	// however the implementation must return an ENOENT error in any case.
	// Every implementation needs to support this method for exactly those entries returned by #ReadBucket() (excluding
	// the .$ query folders, if any).
	ReadAttrs(ctx Cancelable, paths *StrList) (Entries, error)

	// WriteAttrs inserts or updates attributes of resources. Implementations may provide specific semantic behavior to
	// certain resources. Implementations should return all modified entries in their full set.
	// There are no consistency guarantees.
	// Some implementations may provide all other entries when returning, with a nil entry at the affected position,
	// however the implementation must return an ENOENT error in any case.
	// Implementations may reject this operation permanently with ENOSYS error.
	WriteAttrs(ctx Cancelable, paths *StrList, attrs xobj.Arr) (Entries, error)

	// ReadBucket reads the contents of a directory. A bucket may contain other buckets and blobs.
	// If path does not exist ENOENT is returned. If path is not a directory, an ENOTDIR error is returned.
	// Any options should be either expressed through the hidden navigable bucket hierarchy '.$'
	// or by using URI style parameters.
	//
	// Example
	//
	//   URI style: /my/bucket?sortBy=name&order=asc (truncate path for child names, e.g. /my/bucket/myblob.bin instead of /my/bucket/myblob.bin?sortBy=name&order=asc)
	//   Path style: /my/bucket/.$/sortBy/name/asc (truncate path for child names, e.g. /my/bucket/myblob.bin instead of /my/bucket/.$/sortBy/name/asc/myblob.bin)
	//
	// Every implementation needs to support this method at least with the root path "/" to list all available
	// buckets or blobs.
	ReadBucket(ctx Cancelable, path string) (Entries, error)

	// Tries to create the given path hierarchy. If path already denotes a bucket, nothing happens (it is not removed).
	// If any path segment already refers a blob, an ENOTDIR error is returned.
	// Implementations may reject this operation permanently with ENOSYS error.
	MkBucket(ctx Cancelable, path string) error

	// Move renames a blob or bucket from the old to the new path.
	// If oldPath does not exist, ENOENT is returned. If newPath exists, it will be replaced. If newPath
	// denotes a Bucket, it is also removed first. There are no consistency guarantees, but an implementation should
	// ensure that oldPath is never lost, so either being still at oldPath and/or already at newPath,
	// however the data at newPath may have been lost so that the data of oldPath is at both places.
	// Implementations may reject this operation permanently with ENOSYS error.
	Move(ctx Cancelable, oldPath string, newPath string) error

	// SymLink tries to create a soft link or an alias for an existing resource. This is usually just a special
	// reference which contains the oldPath entry which is resolved if required. The actual resource becomes unavailable
	// as soon as the original path is removed, which will cause the symbolic link to become invalid or to disappear.
	// Implementations may support this for buckets or blobs differently and will return ENOTDIR or EISDIR to
	// give insight. If newPath already exists EEXIST is returned.
	// Implementations may reject this operation permanently with ENOSYS error.
	SymLink(ctx Cancelable, oldPath string, newPath string) error

	// HardLink tries to create a new named entry for an existing resource. Changes to one of both named entries
	// are reflected vice versa, however due to eventual consistency it may take some time to become visible.
	// To remove the resource, one has to remove all named entries.
	// Implementations may support this for buckets or blobs differently and will return ENOTDIR or EISDIR to
	// give insight. If newPath already exists EEXIST is returned.
	// Implementations may reject this operation permanently with ENOSYS error.
	HardLink(ctx Cancelable, oldPath string, newPath string) error

	// Copy tries to perform a copy from old to new using the most efficient possible way, e.g. by using
	// reference links. Implementations may reject this operation permanently with ENOSYS error. If oldPath and/or
	// newPath refer to buckets and the backend does not support that operations for buckets, EISDIR error is returned
	// or vice versa ENOTDIR if only buckets are supported.
	Copy(ctx Cancelable, oldPath string, newPath string) error

	// Close when Done to release resources
	io.Closer
}

// Entries represents a bunch of entries, either buckets or blobs.
// Some implementations may always return all results in a single page, others
// may always require at least two page accesses to determines if all data has been sent.
// There may be even implementations which perform delta updates through paging,
// so you need to know internals of Attr to e.g. evaluate if an
// entry of a prior page has been deleted.
//
// There are no consistency guarantees or order options guaranteed and all of them are implementation specific.
type Entries interface {
	// Total is the estimated total amount of entries across all pages. Implementations may guess or simply return
	// negative numbers if they don't know.
	Total() int64

	// Size is the amount of entries in this page, which is always known and can be accessed without further
	// I/O operations.
	Size() int

	// EntryAt returns the Entry of the indexed entry at the given position. Panics if out of bounds.
	EntryAt(idx int) Entry

	// NextPage loads the next page of entries. If there are currently no more pages, returns
	// a valid page with zero entries. It depends on the implementation if an empty Page may return
	// a non empty Page later again in the future.
	Next() (Entries, error)
}

// An Entry is a typed accessor for an Attr
//TODO there will be always ever a single implementation, so should be a struct
type Entry interface {
	// IsDir returns true, if this entry is a directory and can be used to query contents. (the number attribute '.d')
	IsDir() bool

	// Name returns the (unique) name or id of the entry (the string attribute '.n')
	Name() string

	// Size is the estimated blob size in bytes of the primary data stream, if available at all.
	// There are many implementations which cannot provide any meaningful value (the number attribute '.s').
	// If the meaning is imprecise or not available, a negative number should be returned.
	Size() int64

	// Version denotes a discriminator to distinguish different versions of the same resource.
	// Not all implementations can provide this and the meaning and format is different across
	// implementations, e.g. it may be a counter, a cryptographic hash sum or the time of the last modification.
	// (the string '.v'). Returns the empty string, if not supported.
	Version() string

	// Unwrap returns the original attribute set for later inspection. Also contains the .d/.n/.s/.v attributes.
	// Note that these values may not be a defensive copy due to performance aspects, so do not modify them blindly.
	// For REST based implementations this is usually the original response object, for XML based protocols
	// a jsonml transformation should have been applied.
	Unwrap() xobj.Obj
}
