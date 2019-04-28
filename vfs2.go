package vfs

import (
	"github.com/worldiety/xobj"
	"io"
	"os"
)

type DataDriver interface {
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
	Open(ctx Cancelable, path string, flag int, perm os.FileMode) (Resource, error)

	// Delete removes a batch of path entries and all contained children.
	// It is not considered an error to delete a non-existing resource. If deletion of a single entry fails,
	// the result is undefined, e.g. n-1 entries may be already removed or none at all. However the implementation
	// must return an error in any case.
	Delete(ctx Cancelable, paths *StrList) error

	// ReadAttrs reads a batch of path entries. If a single entry cannot be read, the result is undefined. Some
	// implementations may provide all other entries, however the implementation must return an error in any case.
	ReadAttrs(ctx Cancelable, paths *StrList) (xobj.Arr, error)

	// Writes Attributes. This is an optional implementation and may simply return UnsupportedOperationError.
	WriteAttrs(ctx Cancelable, paths *StrList, attrs xobj.Arr) error

	// ReadDir reads the contents of a directory. If path is not a directory, a ResourceNotFoundError is returned.
	// options can be arbitrary and at least nil options must be supported, otherwise unsupported abstraction will
	// cause an *UnsupportedAttributesError to be returned. Using the options, the query to
	// retrieve the directory contents can be optimized, like required fields, sorting, page size, filter etc. This
	// is especially important for online sources, because it also allows arbitrary queries, which are not
	// related to a path hierarchy at all.
	ReadBucket(ctx Cancelable, path string, options xobj.Obj) (Page, error)

	// Tries to create the given path hierarchy. If path already denotes a directory nothing happens. If any path
	// segment already refers a resource, an error must be returned.
	MkBucket(ctx Cancelable, path string) error

	// Rename moves a file from the old to the new path. If oldPath does not exist, ResourceNotFoundError is returned.
	// If newPath exists, it will be replaced.
	Rename(ctx Cancelable, oldPath string, newPath string) error

	// Link can create different kind of links for paths. The kind of links is specified by mode (symbolic, reference
	// or hard). Symbolic is usually an os specific file containing the actual path to follow. Reference is like
	// a real copy but for copy-on-write systems this is nearly a no-op but changes will not be reflected on the source
	// file. In contrast to that, a hardlink points to the original source file, so two (or more) names for the same
	// resource.
	// The parameter flags is reserved (and unspecified) and can
	// be used to narrow behavior e.g. for the reflink syscall.
	Link(ctx Cancelable, oldPath string, newPath string, mode int32, flags int32) error

	// Close when Done to release resources
	io.Closer
}

// Page represents a bunch of results. Some implementations may always return all results in a single page, others
// may even impose limits on the maximum page size. There may be even implementations which
// perform delta updates through paging, so you need to know internals of Attr to e.g. evaluate if an
// entry of a prior page has been deleted.
type Page interface {
	// Total is the estimated total amount of entries across all pages
	Total() int64

	// Size is the amount of entries in this page
	Size() int

	// EntryAt returns the Attributes of the indexed entry
	EntryAt(idx int) Attr

	// NextPage loads the next page of entries. If there are currently no more pages, returns
	// a valid page with zero entries. It depends on the implementation if an empty Page may return
	// a non empty Page later again.
	NextPage() (Page, error)
}

// An Entry is a typed accessor for an Attr
type Entry interface {
	// IsDir returns true, if this entry is a directory and can be used to query contents. (the flag '.d')
	IsDir() bool

	// Name returns the (unique) name or id of the entry (the string '.n')
	Name() string

	// Size is the estimated blob size in bytes of the primary data stream, if available at all.
	// There are many implementations which cannot provide any meaningful value (the number '.s'). If the meaning
	// is imprecise or not available, a negative number should be returned.
	Size() int64

	// Version denotes a discriminator to distinguish different versions of the same resource.
	// Not all implementations can provide this and the meaning and format is different across
	// implementations, e.g. it may be a counter, a cryptographic hash sum or the time of the last modification.
	// (the string '.v')
	Version() string

	// Unwrap returns the original attribute set
	Unwrap() xobj.Obj
}

// AsEntry wraps the given attr as an Entry.
// It interprets the attribute .d as directory flag and .n as the name.
func AsEntry(attr Attr) Entry {
	return nil
}
