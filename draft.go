package vfs

// A Wrapper is an error implementation
// wrapping context around another error.
// TODO remove in Go2 https://go.googlesource.com/proposal/+/master/design/go2draft-error-inspection.md
type Wrapper interface {
	// Unwrap returns the next error in the error chain.
	// If there is no next error, Unwrap returns nil.
	Unwrap() error
}

//ForEachErr loops each error started at root
func ForEachErr(root error, closure func(err error) bool) {
	for root != nil {
		if !closure(root) {
			return
		}
		if causer, ok := root.(Wrapper); ok {
			root = causer.Unwrap()
		}
	}
}

// UnwrapUnsupportedAttributesError either returns the first occurrence of UnsupportedAttributesError or nil
func UnwrapUnsupportedAttributesError(root error) *UnsupportedAttributesError {
	var tmp *UnsupportedAttributesError
	ForEachErr(root, func(err error) bool {
		if myErr, ok := err.(*UnsupportedAttributesError); ok {
			tmp = myErr
			return true
		}
		return false
	})

	return tmp
}

// UnsupportedOperationError either returns the first occurrence of UnsupportedAttributesError or nil
func UnwrapUnsupportedOperationError(root error) *UnsupportedOperationError {
	var tmp *UnsupportedOperationError
	ForEachErr(root, func(err error) bool {
		if myErr, ok := err.(*UnsupportedOperationError); ok {
			tmp = myErr
			return true
		}
		return false
	})

	return tmp
}
