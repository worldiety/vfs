package vfs

import "fmt"

// TODO remove in Go2 https://go.googlesource.com/proposal/+/master/design/go2draft-error-inspection.md
// A Wrapper is an error implementation
// wrapping context around another error.
type Wrapper interface {
	// Unwrap returns the next error in the error chain.
	// If there is no next error, Unwrap returns nil.
	Unwrap() error
}

//

type MountPointNotFoundError struct {
	MountPoint string
	Cause      error
}

func (e *MountPointNotFoundError) Unwrap() error {
	return e.Cause
}

func (e *MountPointNotFoundError) Error() string {
	return "not found: " + e.MountPoint
}

//

type UnsupportedOperationError struct {
	Message string
	Cause   error
}

func (e *UnsupportedOperationError) Error() string {
	return e.Message
}

func (e *UnsupportedOperationError) Unwrap() error {
	return e.Cause
}

//

type ResourceNotFoundError struct {
	Path  Path
	Cause error
}

func (e *ResourceNotFoundError) Error() string {
	return e.Path.String()
}

func (e *ResourceNotFoundError) Unwrap() error {
	return e.Cause
}

//

type UnsupportedAttributesError struct {
	Data  interface{}
	Cause error
}

func (e *UnsupportedAttributesError) Error() string {
	return fmt.Sprintf("%v", e.Data)
}

func (e *UnsupportedAttributesError) Unwrap() error {
	return e.Cause
}

//
type CancellationError struct {
	Cause error
}

func (e *CancellationError) Error() string {
	return "cancelled"
}

func (e *CancellationError) Unwrap() error {
	return e.Cause
}
