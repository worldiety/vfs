package vfs

// A Wrapper is an error implementation
// wrapping context around another error.
// TODO remove in Go2 https://go.googlesource.com/proposal/+/master/design/go2draft-error-inspection.md
type Wrapper interface {
	// Unwrap returns the next error in the error chain.
	// If there is no next error, Unwrap returns nil.
	Unwrap() error
}

// AsValue is a Draft method to unwrap an error from a chain of errors
func AsValue(e interface{}) {

}
