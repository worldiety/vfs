package vfs

// An optional DataProvider contract which offers the possibility of more efficient batch operations.
type BatchDataProvider interface {
	// Deletes all given path entries and all contained children. It is not considered an error to delete a
	// non-existing resource.
	BatchDelete(path ...Path) error

	// Reads all given attributes in a batch. Every implementation must support ResourceInfo
	BatchReadAttrs(attribs ...Attributes) error

	// Writes all given attributes. This is an optional implementation and may simply return OperationNotSupportedError
	BatchWriteAttrs(attribs ...Attributes) error

	DataProvider
}

// Attributes is just a simple holder to keep Path and unspecified data together
type Attributes struct {
	Path Path
	Data interface{}
}
