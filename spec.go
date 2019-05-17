//Package vfs provides the API specification and basic tools for virtual filesystems.
package vfs

import (
	"context"
	"io"
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
// clash with the path and fork separator. If the string is not a valid utf8 sequence, 0 is returned.
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

// A Blob is an abstract accessor to read or write bytes. In general, not all methods are supported, either
// due to the way the resources have been opened or because of the underlying implementation. In such cases
// the affected method will always return an *UnsupportedOperationError.
type Blob interface {
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

// A ResourceListener is used to get notified about changes
type ResourceListener interface {
	// OnStatusChanged is called e.g. when a file has been added, removed or changed. event contains
	// implementation specific information. There are many examples of detailed changed events, which
	// we don't like to specify, like onRead, before read, after read, renaming, meta data, quota, lock status,
	// target link, ownership, truncated, ...
	//
	// It is also not defined, when an event is fired and received. However there may be implementation which
	// provide a "before" semantic and may evaluate any returned error, so that a ResourceListener can be used
	// as an interceptor or in a kind of aspect oriented programming.
	OnEvent(path string, event interface{}) error
}

// The FileSystem interface is the core contract to provide access to hierarchical structures using a compound
// key logic. This is an abstract of way of the design thinking behind a filesystem.
//
// Design decisions
//
// There are the following opinionated decisions:
//
//  * It is an Interface, because it cannot be expected to have a reasonable code reusage between implementations but we
//    need a common behavior. There is a builder to create implementation in a simple way.
//
//  * It contains both read and write contracts, because a distinction between read-only and write-only and filesystems
//    with both capabilities are edge cases. Mostly there will be implementations which provides each combination due to
//    their permission handling.
//
//  * It is not specified, if a FileSystem is thread safe. However every
//    implementation should be as thread safe as possible, similar to the POSIX filesystem specification.
//
//  * The entire VFS specification is completely based on (non recursive) interfaces
//    and can be implemented without any dependencies. The builder, the package level functions and
//    default implementations do not belong to the specification, however you are encouraged to use them.
//    If you use the builder, you can be sure to get at least a commonly supported contract.
//
//  * It makes heavy usage of interface{}, which always requires heap allocations and type assertions. However
//    it is not possible to define a common contract for all. Even the os.Filemode makes no sense for most
//    online implementations. Using map[string]interface{} is even worse
//    because it prevents the usage of real types and would undermine the type system entirely. So at the end
//    using interface{} is the most go-ish we could do here. This will probably also not change with the introduction
//    of generics, because they cannot help us with a generic (and perhaps "meaningless" contract).
//
type FileSystem interface {
	// Connect may perform an authentication and authorization based on the given properties.
	// This method changes the internal state of the file system. Implementations may change the properties, so
	// that a refresh token can be returned. Some implementations may support distinct connections
	// per bucket. So a workflow may be as follows:
	//
	//  props := onLoadInstanceState() // load properties from somewhere
	//  err := vfs.Connect(context.Background(), "/", props) // try to connect
	//  if err == nil {
	//     return vfs // everything was fine, exit
	//  }
	//
	//  // connection failed, so fill in custom credentials (you need to read the documentation)
	//  props["user"] = "john.doe@mycompany.com"
	//  props["pwd"] = "1234"
	//  err = vfs.Connect(context.Background(), "/", props)  // reconnect
	//  if err == nil { // props may contain refresh token, a session id or anything arbitrary
	//	    delete(props, "user")  // you may want to keep the user to autofill
	//      delete(props, "pwd")  // but remove credentials which are worth protecting
	//      onSaveInstanceState(props) // save whatever else has been inserted into the properties
	//  }
	//
	// Implementations may reject this operation permanently with ENOSYS error.
	Connect(ctx context.Context, path string, options interface{}) error

	// Disconnect terminates the internal state of authentication and authorization, e.g. by destroying a refresh
	// token or a session at the remote side.
	// Implementations may reject this operation permanently with ENOSYS error.
	Disconnect(ctx context.Context, path string) error

	// FireEvent may notify all registered listeners. Depending on the implementation, the behavior in case of
	// errors is undefined. However it is recommended, that an error of a listener will short circuit the invocations
	// of other listeners and return the error early. Also it is recommended that an implementation should evaluate
	// events according to their pre/post semantic and respect that error, so that a listener can cancel an entire
	// operation which is just returned by the calling method.
	//
	// Implementations may reject this operation permanently with ENOSYS error.
	FireEvent(ctx context.Context, path string, event interface{}) error

	// AddListener a ResourceListener to get notified about changes to resources. It returns a handle to unregister.
	// We use a handle to keep the api sleek and to avoid a discussion about the equality of interfaces.
	// Implementations may reject this operation permanently with ENOSYS error.
	AddListener(ctx context.Context, path string, listener ResourceListener) (handle int, err error)

	// RemoveListener removes an already registered listener. It is not an error to remove an unregistered
	// or invalid listener.
	// Implementations may reject this operation permanently with ENOSYS error.
	RemoveListener(ctx context.Context, handle int) error

	// Begin starts a transaction, so that all following method calls are interpreted in the context of the running
	// transaction. The options argument is a map of key/value primitives suitable for json serialization.
	// An implementation shall be as thread safe as possible and should support concurrent read/write
	// on any operation. The context is modified WithValue and used to track the transaction state.
	// If an implementation supports transactions and begin/commit/rollback cycle is not used, the transactional
	// behavior is not defined, which may be e.g. none at all, every operation in a single transaction or committed
	// within a time slot or anything else. However it is guaranteed that an implementation which supports
	// transaction must not fail because the transactional api has not been used. Some implementations
	// may support transaction for sub buckets, otherwise the path must be root (/).
	//
	// Implementations may reject this operation permanently with ENOSYS error. When used with the wrong arguments,
	// e.g. with an unsupported isolation level EINISOL is returned.
	Begin(ctx context.Context, path string, options interface{}) (context.Context, error)

	// Commit applies a running transaction. See also #Begin() for details.
	// Implementations may reject this operation permanently with ENOSYS error. Returns ETXINVALID if no transaction
	// is pending.
	Commit(ctx context.Context) error

	// Rollback does not apply the current state of the transaction and reverts all changes.
	// Implementations may reject this operation permanently with ENOSYS error. Returns ETXINVALID if no transaction
	// is pending.
	Rollback(ctx context.Context) error

	// Open is the general read or write call. It opens the named resource with specified flags (O_RDONLY etc.).
	// The type of options is implementation specific and may be e.g. something like os.FileMode to declare permissions.
	// If successful, methods on the returned File can be used for I/O.
	// If there is an error, the trace will also contain a *PathError.
	// Implementations have to create parent directories, if those do not exist. However if any existing
	// path segment denotes already a resource, the resource is not deleted and an error is returned instead.
	//
	// Resource Forks or Alternate Data Streams (or e.g. thumbnails from online resources) shall be addressed
	// using a colon (:). Example: /myfolder/test.png/thumb-jpg:720p. Note that the : is also considered to be an unportable
	// and unsafe character but indeed it should never make it into a real local path. See also #ReadForks().
	//
	// Do not forget to close the resource, to avoid any leak.
	// Implementations may reject this operation permanently with ENOSYS error.
	Open(ctx context.Context, path string, flag int, options interface{}) (Blob, error)

	// Deletes a path entry and all contained children. It is not considered an error to delete a non-existing resource.
	// This non-posix behavior is introduced to guarantee two things:
	//   * the implementation shall ensure that races have more consistent effects
	//   * descending the tree and collecting all children is an expensive procedure and often unnecessary, especially
	//     in relational databases with foreign key constraints.
	// Implementations may reject this operation permanently with ENOSYS error.
	Delete(ctx context.Context, path string) error

	// ReadAttrs reads arbitrary or extended attributes into an implementation specific destination.
	// This method may returns additional meta data
	// about the resource like size, last modified, permissions, ACLs or even EXIF data. This allows structured
	// information to pass out without going through a serialization process using the fork logic.
	// args may contain query options (like selected fields) but is also used to recycle objects to
	// avoid heap allocations, but this depends on the actual implementation.
	// Implementations may reject this operation permanently with ENOSYS error.
	ReadAttrs(ctx context.Context, path string, args interface{}) (Entry, error)

	// ReadForks reads all available named resource forks or alternate data streams. Any path object may have
	// an arbitrary amount of forks. A file object always has the unnamed fork, which is not included in the
	// returned list. To access a fork, concat the fork name to the regular file name with a colon.
	//
	// Why using the colon?
	//  * the MacOS convention (../forkName/rsrc) cannot distinguish between a relative path and a named fork
	//  * at least a single platform uses this convention (windows), Posix does not support it. Small things are represented
	//    in extended attributes (ReadAttrs)
	//  * the colon is not allowed on MacOS and Windows and also discouraged on Linux for filenames, because it conflicts
	//    e.g. with the path separator. Also most online sources do not allow it for compatibility reasons, besides
	//    google drive, which literally allows anything.
	//  * easy to be read by humans
	//  * seems to be the less of two evils in anticipated use cases within this api
	//
	// Example:
	//
	//   forks, err := vfs.ReadForks(context.Background(), "image.jpg") // forks may contain things like thumbnails/720p
	//   _, _ = vfs.Open(context.Background, os.O_RDONLY, 0, "image.jpg") // opens the unamed (original) data stream
	//   _, _ = vfs.Open(context.Background, os.O_RDONLY, 0, "image.jpg:thumbnails/720p") // opens a thumbnail by a named stream
	//  _ = vfs.Delete(context.Background, "image.jpg:
	//
	// Implementations may reject this operation permanently with ENOSYS error.
	ReadForks(ctx context.Context, path string) ([]string, error)

	// WriteAttrs inserts or updates properties or extended attributes with key/value primitives,
	// suitable for json serialization.
	// Implementations may reject this operation permanently with ENOSYS error.
	WriteAttrs(ctx context.Context, path string, src interface{}) error

	// ReadBucket reads the contents of a directory. If path is not a bucket, an ENOENT is returned.
	// options can be arbitrary primitives and should be json serializable.
	// At least nil and empty options must be supported, otherwise an EUNATTR can be returned.
	// Using the options, the query to retrieve the directory contents can be optimized,
	// like required fields, sorting or page size, if not
	// already passed through the path and its potential fork or query path.
	// Implementations may support additional parameters like sorting or page sizes but should not be appended
	// to the path (uri style), as long as they do not change the actual result set. Options which act like
	// a filter should always map to a distinct path, to avoid confusion or merge conflicts of caching layers on top.
	//
	// Conventionally the colon path /: has a special meaning, because it lists hidden endpoints, which
	// are not otherwise reachable. These endpoints do not make sense to be inspected in a hierarchy. One reason
	// could be, that they always require a set of options as method arguments. There is currently no way
	// of inspecting the arguments programmatically but if you know what you are doing (read the documentation)
	// you can peek through the abstraction but avoid to publish a concrete contract (which may be either something
	// you want to avoid or which you are favoring). See also #Invoke()
	//
	// Implementations may reject this operation permanently with ENOSYS error.
	ReadBucket(ctx context.Context, path string, options interface{}) (ResultSet, error)

	// Invoke is a peephole in the specification to call arbitrary endpoints which are not related to a filesystem
	// but share the same internal state, like the authorization. You need to use type assertions and to
	// consult the documentation to access the concrete
	// type, because it may be a json object, even a http response or an io.Reader, just anything.
	//
	// To inspect the available endpoints you can use /: with #ReadBucket(). Example:
	//
	//   // endpoints contains things like "fullTextSearch"
	//   endpoints, _ := vfs.ReadBucket(context.Background(), "/:", nil)
	//
	//   // perform a custom endpoint query with a json like argument object
	//   args := Options{}
	//   args["text"] = "hello world"
	//   args["sortBy"] = "asc"
	//   args["since"] = "2018.05.14"
	//   args["pageSize"] = 1000
	//   res, err := vfs.Invoke(context.Background(), "fullTextSearch", args) // res may contain a map[string]interface{}
	//
	//   // something which is not serializable at all
	//   reader := getSomeReader()
	//   writer, err := vfs.Invoke(context.Background(), "transfer", reader) // res contains a writer to append stuff
	//   writer.(io.Writer).Write([]byte("EOF"))
	//
	//   // or even more method like
	//   isShared := true
	//   listOfImages := getImageList()
	//   _, err := vfs.Invoke(context.Background(), "createAlbum", "my album title", isShared, listOfImages)
	//
	// Implementations may reject this operation permanently with ENOSYS error.
	Invoke(ctx context.Context, endpoint string, args ...interface{}) (interface{}, error)

	// MkBucket tries to create the given path hierarchy. If path already denotes a directory nothing happens. If any path
	// segment already refers a resource, an error must be returned. The type of options is implementation specific
	// and may be e.g. something like os.FileMode to declare permissions
	MkBucket(ctx context.Context, path string, options interface{}) error

	// Rename moves a file from the old to the new path. If oldPath does not exist, ResourceNotFoundError is returned.
	// If newPath exists, it will be replaced.
	Rename(ctx context.Context, oldPath string, newPath string) error

	// SymLink tries to create a soft link or an alias for an existing resource. This is usually just a special
	// reference which contains the oldPath entry which is resolved if required. The actual resource becomes unavailable
	// as soon as the original path is removed, which will cause the symbolic link to become invalid or to disappear.
	// Implementations may support this for buckets or blobs differently and will return ENOTDIR or EISDIR to
	// give insight. If newPath already exists EEXIST is returned.
	// Implementations may reject this operation permanently with ENOSYS error.
	SymLink(ctx context.Context, oldPath string, newPath string) error

	// HardLink tries to create a new named entry for an existing resource. Changes to one of both named entries
	// are reflected vice versa, however due to eventual consistency it may take some time to become visible.
	// To remove the resource, one has to remove all named entries.
	// Implementations may support this for buckets or blobs differently and will return ENOTDIR or EISDIR to
	// give insight. If newPath already exists EEXIST is returned.
	// Implementations may reject this operation permanently with ENOSYS error.
	HardLink(ctx context.Context, oldPath string, newPath string) error

	// Copy tries to perform a copy from old to new using the most efficient possible way, e.g. by using
	// reference links. Implementations may reject this operation permanently with ENOSYS error. If oldPath and/or
	// newPath refer to buckets and the backend does not support that operations for buckets, EISDIR error is returned
	// or vice versa ENOTDIR if only buckets are supported.
	Copy(ctx context.Context, oldPath string, newPath string) error

	// Close when Done to release resources. Subsequent calls have no effect and do not report additional errors.
	// An implementation may reject closing while still in process, so future calls may be necessary.
	Close() error

	// String returns a name or description of this VFS
	String() string
}

// An EntryList is a collection of loaded bucket entries, whose entries are typically the names of other
// Buckets or entries. We do not include the ReadForks method, because the determination of available forks
// may be expensive and usually requires additional I/O. Logically it belongs to the FileSystem#Open() call
// and is therefore not related to the ResultSet.
type ResultSet interface {
	// ReadAttrs returns the entry for the index >= 0 and < Len() of an already queried response, which
	// is why there is no context here. However args is still in the contract to allows implementation
	// specific allocation free data transformation.
	//
	// See also FileSystem#ReadAttrs()
	ReadAttrs(idx int, args interface{}) Entry

	// Len returns the amount of entries in the entire result set, which are available without any further I/O
	Len() int

	// Total is the estimated amount of all entries when all pages have been requested.
	// Is -1 if unknown and you definitely have to loop over to count.
	// However in the meantime, Size may deliver a more correct estimation.
	Total() int64

	// Pages is the estimated amount of all available pages, including the current one. Is -1 if unknown.
	Pages() int64

	// Next loads the next page of results or EOF if no more results are available. The ResultSet is
	// undefined, if an error has been returned. Otherwise you have to evaluate #Len() again and loop over
	// the set again to grab the next results.
	Next(ctx context.Context) error

	// Data returns the actual model behind this list. Implementations which wrap REST APIs typically return
	// a map[string]interface{} or []interface{}. Can return nil. It is called Data to be compatible with os.FileInfo.Data
	Sys() interface{}
}

// Entry is the contract which each implementation needs to support. It actually allows a named navigation.
// Intentionally it is a subset of os.FileInfo
type Entry interface {
	// Id returns the unique (at least per bucket) id of an entry
	Name() string

	// IsDir is the folder or bucket flag. This flag is an indicator if it makes sense to query contents
	// with #ReadBucket()
	IsDir() bool

	// Data returns the implementation specific payload. This can be anything, e.g. a map[string]interface{} or
	// a distinct struct.
	Sys() interface{}
}
