package vfs

import "fmt"

// A Wrapper is an error implementation
// wrapping context around another error.
// TODO remove in Go2 https://go.googlesource.com/proposal/+/master/design/go2draft-error-inspection.md
type Wrapper interface {
	// Unwrap returns the next error in the error chain.
	// If there is no next error, Unwrap returns nil.
	Unwrap() error
}

// A MountPointNotFoundError is only used by the MountableDataProvider to indicate that the given path cannot be
// associated with a mounted DataProvider. Check your prefix.
type MountPointNotFoundError struct {
	MountPoint string
	Cause      error
}

// Unwrap returns nil or the cause.
func (e *MountPointNotFoundError) Unwrap() error {
	return e.Cause
}

func (e *MountPointNotFoundError) Error() string {
	return "not found: " + e.MountPoint
}

// UnsupportedOperationError is always returned, if an implementation does not support a function in general.
type UnsupportedOperationError struct {
	Message string
	Cause   error
}

func (e *UnsupportedOperationError) Error() string {
	return e.Message
}

// Unwrap returns nil or the cause.
func (e *UnsupportedOperationError) Unwrap() error {
	return e.Cause
}

// ResourceNotFoundError is always returned if a path or file is required to complete an operation but no such resource
// is available.
type ResourceNotFoundError struct {
	Path  Path
	Cause error
}

func (e *ResourceNotFoundError) Error() string {
	return e.Path.String()
}

// Unwrap returns nil or the cause.
func (e *ResourceNotFoundError) Unwrap() error {
	return e.Cause
}

// UnsupportedAttributesError is returned by DataProvider#ReadAttrs and DataProvider#WriteAttrs whenever a type
// has been given which is not supported by the actual DataProvider implementation.
type UnsupportedAttributesError struct {
	Data  interface{}
	Cause error
}

func (e *UnsupportedAttributesError) Error() string {
	return fmt.Sprintf("%v", e.Data)
}

// Unwrap returns nil or the cause.
func (e *UnsupportedAttributesError) Unwrap() error {
	return e.Cause
}

// CancellationError is always used to indicate an implemented cancellation and is never returned by default.
type CancellationError struct {
	Cause error
}

func (e *CancellationError) Error() string {
	return "cancelled"
}

// Unwrap returns nil or the cause.
func (e *CancellationError) Unwrap() error {
	return e.Cause
}
