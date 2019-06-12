package vfs

import (
	"reflect"
	"strconv"
	"time"
)

// An Error incorporates the go 2 draft Wrapper interface and a status/error code and details. The error codes
// are largely inspired by the posix libc but are different in detail.
type Error interface {
	error

	// Unwrap returns the next error in the error chain.
	// If there is no next error, Unwrap returns nil.
	Unwrap() error

	// StatusCode returns a code which specifies the kind of error in more details. You may also want to
	// inspect #Details() to get more insight.
	StatusCode() int

	// Details may contain more information about
	Details() interface{}
}

// Reserved vfs status codes are between 0 and 255 (range of uint8). You can use any other values outside
// of this range for custom status codes. Many codes are intentionally equal to the posix specification
// and share most of semantics but not all, so be aware of the details.
// See also https://www.gnu.org/software/libc/manual/html_node/Error-Codes.html.
// Also a lot of codes have been tagged with unspecified, because the meaning is either not clear in the specification
// or because it does seem to bring any advantage to the vfs specification.
const (
	// Introduced for completeness. A non-nil VFSError is not allowed to return this. Details is always nil.
	EOK = 0

	// Operation not permitted
	//
	// Caution, definition changed from posix meaning: This error differs from EACCES (permission denied) in the way
	// that the the entire vfs is not accessible and needs to be reauthenticated. A typical situation is that
	// the realm of a user has changed, e.g. by revoking an access token.
	//
	// The details are implementation specific.
	EPERM = 1

	// No such file or directory / resource not found
	//
	// A resource like a folder or file may get removed at any time. Certain non-cached operations will therefore fail.
	//
	// The details contain a []string array which includes all affected path Entries.
	ENOENT = 2

	// No such process
	//
	// A useful indicator if an implementation needs a process as counterpart because of IPC communication.
	//
	// The details are implementation specific.
	ESRCH = 3

	// Interrupted system call
	//
	// Used to indicate that an operation has been cancelled.
	//
	// The details are implementation specific.
	EINTR = 4

	// I/O error
	//
	// Channels always have errors like timeouts, pulled ethernet cables, ssl failures etc. Usually these errors are related
	// to OSI-layer 1-4 but may also reach up to layer 5-6 (e.g. https and ssl failures). Note that protocols do not always fit unambiguously into the OSI model.
	// Interrupts are mapped to EINTR and timeouts ETIMEDOUT.
	//
	// The details are implementation specific.
	EIO = 5

	// No such device or address
	//
	// Usually indicates a configuration problem or a physical problem.
	//
	// The details are implementation specific.
	ENXIO = 6

	// Bad file number or descriptor
	//
	// Indicates that a resource has been closed but accessed or that an open mode (e.g. read) does not match an
	// operation (e.g. write)
	//
	// The details contain a []string array which includes all affected path Entries.
	EBADF = 9

	// No child processes
	//
	// If an implementation spawns new process and a child process manipulation fails.
	//
	// The details are implementation specific.
	ECHILD = 10

	// Try again
	//
	// The data service behind the channel (which is working) is temporarily unavailable.
	// You may want to retry it later again. Usually this may be used by servers which are currently in maintenance mode.
	// You may also want to map throttle errors (e.g. to many requests) to this.
	//
	// The details contain a non-nil UnavailableDetails interface.
	EAGAIN = 11

	// Out of memory
	//
	// A requested operation failed or would fail because of insufficient memory resources. This may also be due to
	// configured artificial limit.
	//
	// The details are implementation specific.
	ENOMEM = 12

	// Permission denied
	//
	// Right management is complex. There are usually different rights for the current user to read and write data in various locations.
	// This message usually signals that the general access to the datasource is possible, but not for the requested operation. Examples:
	//
	//    * A local filesystem has usually system files, which cannot be read or write
	//    * A remote filesystem has usually provides folders or files which are read only
	//
	// The details contain a []string array which includes all affected path Entries.
	EACCES = 13

	// Block device required
	//
	// An implementation may require a block device and not a regular file.
	//
	// The details are implementation specific.
	ENOTBLK = 15

	// Device or resource busy
	//
	// A resource is accessed which cannot be shared (or has to many shares) and the operation cannot succeed.
	// This usually happens also to regular filesystems when deleting a parent folder but you continue using a child.
	//
	// The details contain a []string array which includes all affected path Entries.
	EBUSY = 16

	// File exists
	//
	// An operation expected that it has to create a new file and that it is definitely an error that it already exists.
	//
	// The details contain a []string array which includes all affected path Entries.
	EEXIST = 17

	// Cross-device link
	//
	// This is usually used in two situation: A local file system with different mount points and creating links
	// between file systems or doing the same across vfs mounted filesystems.
	//
	// The details contain a []string array which includes all affected path Entries.
	EXDEV = 18

	// No such device
	//
	// A specific device is required, which is not available.
	//
	// The details are implementation specific.
	ENODEV = 19

	// Not a directory
	//
	// An operation was requested which required a directory but found that it's not.
	//
	// The details contain a []string array which includes all affected path Entries.
	ENOTDIR = 20

	// Is a directory
	//
	// An operation was requested which required a file but found that it's a directory.
	//
	// The details contain a []string array which includes all affected path Entries.
	EISDIR = 21

	// Invalid argument
	//
	// A generic code to indicate one or more invalid parameters.
	//
	// The details are implementation specific.
	EINVAL = 22

	// File table overflow / to many open files
	//
	// The resources of the entire system to keep files open, are depleted. See also EMFILE, which is more common.
	//
	// The details contain a []string array which includes all affected path Entries.
	ENFILE = 23

	// Too many open files
	//
	// The configured limit of your process or filesystem has been reached. See also ENFILE.
	//
	// The details contain a []string array which includes all affected path Entries.
	EMFILE = 24

	// File too large
	//
	// The system may impose limits to the maximum supported file size.
	//
	// The details contain LimitDetails
	EFBIG = 27

	// No space left on device
	//
	// There is not enough free space. See also EDQUOT or EFBIG to distinguish other related conditions.
	//
	// The details contain LimitDetails
	ENOSPC = 28

	// Read-only file system
	//
	// Returned to distinguish intrinsic read-only property from a permission problem.
	//
	// The details are implementation specific.
	EROFS = 30

	// Resource deadlock would occur
	//
	// You are lucky and the system detected that your operation would result in a deadlock, like using folded
	// transactions.
	// The details are implementation specific.
	EDEADLK = 35

	// File name too long
	//
	// The identifier for a file, directory, hostname or similar is too long.
	//
	// The details contain a []string array which includes all affected path Entries.
	ENAMETOOLONG = 36

	// No record locks available
	//
	// A resource may be locked, so that others are not allowed to read, write, delete or move a resource. It depends on the
	// data source which operations are not allowed. For example it may still be allowed to read, but not to delete it.
	// To continue, you need to acquire a lock properly.
	//
	// The details are implementation specific.
	ENOLCK = 37

	// Function not implemented / operation not supported
	//
	// A persistent implementation detail of a filesystem, that it does not implement a specific operation.
	//
	// The details are implementation specific.
	ENOSYS = 38

	// Directory not empty
	//
	// An operation expected an empty directory but found it's not. Caution: in contrast to the posix specification,
	// deleting a non-empty directory is not allowed to fail because it is empty.
	//
	//
	ENOTEMPTY = 39

	// Too many symbolic links encountered / cycle detected
	//
	// When using links, a cycle can be constructed which may cause infinite loops.
	//
	// The details are implementation specific.
	ELOOP = 40

	// No data available
	//
	// The server responded but contained no data
	//
	// The details are implementation specific.
	ENODATA = 61

	// Object is remote
	//
	// Caution, this is defined entirely different: The object cannot be accessed because the system works in offline
	// mode and the object is only available on remote.
	//
	// The details are implementation specific.
	EREMOTE = 66

	// Communication error on send
	//
	// A corruption or error has been detected while sending or receiving, e.g. because something went wrong on the
	// line or has been tampered.
	//
	// The details are implementation specific.
	ECOMM = 70

	// Protocol error
	//
	// The implementation expected a certain behavior of the backend (e.g. a network server response) but the protocol failed. This is likely an implementation failure at the client side (assuming that the server is always right).
	// One could argue that it would be better to throw and die but the experience shows that assertions in backends are usually violated in corner cases, so dieing is not really helpful for the user.
	//
	// Inspect the details, which are implementation specific.
	EPROTO = 71

	// Value too large for defined data type
	//
	// In various situations, an implementation may force a maximum size of a data structure. This usually happens in
	// case of corruptions or attacks.
	//
	// Inspect the details, which are implementation specific.
	EOVERFLOW = 75

	// Id not unique on network
	//
	// There are several operations, especially for creating resources or renaming them which may fail due to uniqueness
	// constraints.
	//
	// The details contain a []string array which includes all affected path Entries.
	ENOTUNIQ = 76

	// Illegal byte sequence
	//
	// This may be returned, if there is a format error, e.g. because an UTF-8 sequence is not UTF-8 or a string
	// contains a null byte or a jpeg is truncated.
	//
	// Inspect the details, which are implementation specific.
	EILSEQ = 84

	// Protocol not available
	//
	// The client requested a specific protocol or version but the server rejects the connection because it won't support
	// that protocol anymore. Usually this indicates, that the client must be updated.
	//
	// The details are implementation specific.
	ENOPROTOOPT = 92

	// Protocol not supported
	//
	// Especially for network bindings (or also container formats). The channel works, however the protocol implementation
	// detected that it is incompatible with the service (e.g. the remote side is newer than the client side and is not
	// backwards compatible).
	//
	// The details are implementation specific.
	EPROTONOSUPPORT = 93

	// Address already in use
	//
	// Usually returned, if a socket server is spawned but another server is already running on that port.
	//
	// The details are implementation specific.
	EADDRINUSE = 98

	// Cannot assign requested address
	//
	// Usually returned, if a socket server should be bound to a host name or ip, which is not available.
	//
	// The details are implementation specific.
	EADDRNOTAVAIL = 99

	// Network is down
	//
	// Your network is gone and your user should plugin the cable or disable airplane mode.
	//
	// The details are implementation specific.
	ENETDOWN = 100

	// Network is unreachable
	//
	// A part of your network is gone and your user should plugin the cable to the switch.
	//
	// The details are implementation specific.
	ENETUNREACH = 101

	// Network dropped connection because of reset
	//
	// The network has a hickup, probably just try again
	//
	// The details are implementation specific.
	ENETRESET = 102

	// Software caused connection abort
	//
	// The connection was aborted by intention. You likely want to check your implementation. A reason may be
	// that a server wants to force some properties like SSL on you.
	//
	// The details are implementation specific.
	ECONNABORTED = 103

	// Connection reset by peer
	//
	// Your connection died, e.g. because the remote is rebooting, probably just try again later.
	ECONNRESET = 104

	// Connection timed out
	//
	// There was some kind of a time out.
	//
	// The details are implementation specific.
	ETIMEDOUT = 110

	// Connection refused
	//
	// Typically a host rejected the connection, because the service is not available.
	//
	// The details are implementation specific.
	ECONNREFUSED = 111

	// Host is down
	//
	// The remote host is down.
	//
	// The details are implementation specific.
	EHOSTDOWN = 112

	// No route to host
	//
	// The entire remote host is not reachable.
	//
	// The details are implementation specific.
	EHOSTUNREACH = 113

	// Operation already in progress
	//
	// An implementation may be clever and reject operations which are already in progress.
	//
	// The details are implementation specific.
	EALREADY = 114

	// Remote I/O error
	//
	// Used to indicate that the remote server detected an internal problem, like a crash (e.g. http 500)
	EREMOTEIO = 121

	// Quota exceeded
	//
	// A users storage quota has been exceeded. This may be caused by other resources, like transfer volume or throughput.
	//
	// Details contain LimitDetails
	EDQUOT = 122

	// === non posix error codes below === //

	// End Of File
	EOF = 248

	// Invalid transaction
	//
	// The transaction is invalid, e.g. because it has been closed already.
	ETXINVALID = 249

	// Invalid iterator
	//
	// This error is returned as soon as the underlying iterator has been invalidated, which means that it has been
	// corrupted and can no longer be used. Iterators without snapshot support have to detect situations where the
	// consistency of already returned Entries is violated, however in such cases you may try to revalidate it to
	// get a new instance (not specified). Once EITINVALID has been returned, the iterator cannot be used
	// anymore. It signals an unfixable error which cannot be resolved by repetition.
	// Example situations:
	//
	//   * read A, B, C then adding D, E, F : getSize increases from 3 to 6 while iterating => all is good,
	//     everything is still consistent
	//   * read A, B, C then removing A, B, C and adding D, E, F: getSize is identical, but all items are entirely
	//     different. Because the consumer of the iterator cannot detect this situation, the iterator is => invalid
	//   * read A, B, C then removing A: getSize is different, but the consumer cannot know what happened, the iterator
	//     is => invalid
	//   * read A, jump, read C then removing B: getSize is different and jumping to B would return C again, the
	//     iterator is => invalid
	//   * tricky: jump, read Z, Y, X, remove Y => invalid
	//
	// => The generic rule is: deletion, insertion or changing order will always result in an invalid iterator. The only allowed modification is appending at the end. Jumping to the end may result in an inefficient entire deserialization of the list (depends on the implementation). If this is an issue, consider to reformulate your query using sorting criteria instead.
	EITINVALID = 250

	// Invalid isolation level
	//
	// Everything is executed in a transaction. If the level is unsupported this kind is returned, instead of executing it using the wrong level. The only exception of this rule is the usage of IsolationLevel#None
	// where the implementation is free to choose.
	//
	// The details are implementation specific.
	EINISOL = 251

	// Account expired
	//
	// The account used for login was marked as expired by the backend. This generally is used
	// to indicate some kind of user action outside the scope is required to fix this error,
	// like visiting the service's website or contacting it's admin.
	//
	// Inspect the details, which are implementation specific.
	EAEXP = 252

	// Mount point not found
	//
	// A MountPointNotFoundError is only used by the MountableFileSystem to indicate that the given path cannot be
	// associated with a mounted FileSystem. Check your prefix.
	//
	// The details contain a []string array which includes all affected path Entries.
	ENOMP = 253

	// Unsupported attributes
	//
	// Typically returned by FileSystem#ReadAttrs and FileSystem#WriteAttrs whenever a type
	// has been given which is not supported by the actual FileSystem implementation.
	//
	// The details are implementation specific.
	EUNATTR = 254

	// unknown error
	//
	// The unknown error must be used, if a return value with colliding or unknown meaning is transported, e.g. if
	// an implementation simply returns an unchecked http status code.
	//
	// The details contain an int of the unchecked original error code.
	EUNKOWN = 255
)

var statusText = map[int]string{
	EOK: "OK",
}

// StatusText returns th
func StatusText(code int) string {
	val, ok := statusText[code]
	if !ok {
		return "Status-" + strconv.Itoa(code)
	}
	return val
}

// UnavailableDetails indicates a temporary downtime and communicates a retry time.
type UnavailableDetails interface {
	// A specific message, useful for the user
	UserMessage() string
	// RetryAfter returns the duration to wait before
	RetryAfter() time.Duration
}

// LimitDetails represents a contract to access a min, current and max occupation of a resource.
type LimitDetails interface {
	// A specific message, useful for the user
	UserMessage() string
	// the required minimal value
	Min() int64
	// the actual used resources
	Used() int64
	// the maximum available resources
	Max() int64
}

type wrapper interface {
	Unwrap() error
}

// IsErr inspects the wrapped hierarchy for a specific statusCode
func IsErr(err error, statusCode int) bool {
	if e, ok := err.(Error); ok {
		if e.StatusCode() == statusCode {
			return true
		}

	}

	if e, ok := err.(wrapper); ok {
		// recursive search
		return IsErr(e.Unwrap(), statusCode)
	}

	return false
}

// DefaultError implements the Error interface
var _ Error = (*DefaultError)(nil)

type DefaultError struct {
	Message        string
	Code           int
	CausedBy       error
	DetailsPayload interface{}
}

func (e *DefaultError) Error() string {
	if len(e.Message) > 0 {
		return e.Message + ": " + StatusText(e.Code)
	}
	return StatusText(e.Code)
}

func (e *DefaultError) Unwrap() error {
	return e.CausedBy
}

func (e *DefaultError) StatusCode() int {
	return e.Code
}

func (e *DefaultError) Details() interface{} {
	return e.DetailsPayload
}

func NewErr() errBuilder {
	return errBuilder{}
}

type errBuilder struct {
}

// NewUnsupportedOperation creates an ENOSYS
func (b errBuilder) UnsupportedOperation(msg string) *DefaultError {
	return &DefaultError{msg, ENOSYS, nil, nil}
}

// NewUnsupportedAttributes creates an EUNATTR
func (b errBuilder) UnsupportedAttributes(msg string, what interface{}) *DefaultError {
	return &DefaultError{msg + ": " + reflect.TypeOf(what).String(), EUNATTR, nil, what}
}

//
var eof = &DefaultError{Code: EOF}
