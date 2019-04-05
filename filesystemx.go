package vfs


// A ResourceName is just an alias to avoid an unneeded dependency
type ResourceName = string

// NamedResources is an optional attribute interface and a standardized way of inspecting available Resource Forks for
// a path entry.
//
// Usage example
//   var images vfs.FileSystem
//   //...
//   attrs := &DefaultNamedResources{}
//   images.ReadAttrs("/my/folder/image.jpg", attrs)
//   // attrs will contain now thumb-512, thumb-720p, thumb-1080p, exif-json and xmp
//   res := images.Open("/my/folder/image.jpg"+vfs.ForkSeparator+attrs.Names()[0], os.O_RDONLY, 0)
type NamedResources interface {
	// Names returns a slice of names
	Names() []ResourceName

	// SetNames updates the slice list
	SetNames(names []ResourceName)
}

// SortOrder is either ASC (true) or DESC (false)
type SortOrder = bool

const (
	// Sort ascending
	ASC SortOrder = true
	// Sort descending
	DESC SortOrder = false
)

// QueryOptions is an optional query options interface and a standardized way to perform a projection of meta data
// and signal sort orders.
//
// Usage example
//
//   var images vfs.FileSystem
//   //...
//   it, err := images.ReadDir("/my/folder", &DefaultQueryOptions{Select:"caption", Order:vfs.ASC, By:[]string{"takenAt"}}
type QueryOptions interface {
	// Projection returns the required field names, e.g. caption, sha256, size, width, takenAt
	Projection() (fieldNames []string)

	// OrderBy returns a boolean flag to either sort ascending or descending. Has no effect if fieldNames is empty.
	OrderBy() (asc SortOrder, fieldNames []string)
}



// BatchFileSystem is an optional contract which offers the possibility of more efficient batch operations.
// This can be very important for remote services, where the call overhead is enormous.
type BatchFileSystem interface {
	// Deletes all given path entries and all contained children. It is not considered an error to delete a
	// non-existing resource.
	BatchDelete(path ...string) error

	// Reads all given attributes in a batch. Every implementation must support ResourceInfo
	BatchReadAttrs(attribs ...Attributes) error

	// Writes all given attributes. This is an optional implementation and may simply return UnsupportedOperationError
	BatchWriteAttrs(attribs ...Attributes) error

	FileSystem
}

// Attributes is just a simple holder to keep Path and unspecified data together
type Attributes interface {
	Path() string
	Data() interface{}
}

//===================

// An IsolationLevel determines the isolation between concurrent transactions.
type IsolationLevel = int



// See https://en.wikipedia.org/wiki/Isolation_(database_systems)#Isolation_levels to learn more about isolation levels.
const (
	LevelDefault IsolationLevel = iota
	LevelReadUncommitted
	LevelReadCommitted
	LevelWriteCommitted
	LevelRepeatableRead
	LevelSnapshot
	LevelSerializable
	LevelLinearizable
)

// TxOptions are used to configure and spawn a new concurrent transaction.
type TxOptions interface {
	// Isolation is the transaction isolation level.
	Isolation() IsolationLevel
	// If true, a transaction must deny all modification attempts.
	ReadOnly() bool
}

// A TransactionableFileSystem supports also the usual style. If it implicitly creates a transaction per
// operation or in time slices or other criteria is implementation specific.
type TransactionableFileSystem interface {
	// Begins either a ReadOnly or ReadWrite transaction. ReadOnly may be ignored and used for optimizations only.
	// The returned Transaction must be closed by either committing or by rollback.
	Begin(opts TxOptions) (Tx, error)
	FileSystem
}

// A Tx is the FileSystem contract providing commit and rollback methods but also is a normal FileSystem.
// An implementation should rollback, if a transaction has not been explicitly closed by a
// Commit or Rollback.
type Tx interface {
	Commit() error
	Rollback() error
	// A simple close of the FileSystem without a commit will perform a Rollback.
	FileSystem
}

