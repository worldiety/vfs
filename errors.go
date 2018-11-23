package vfs

import "fmt"

type MountPointNotFoundError struct {
	MountPoint string
}

func (e *MountPointNotFoundError) Error() string {
	return "not found: " + e.MountPoint
}

//

type OperationNotSupportedError struct {
	Message string
}

func (e *OperationNotSupportedError) Error() string {
	return e.Message
}

//

type ResourceNotFoundError struct {
	Path Path
}

func (e *ResourceNotFoundError) Error() string {
	return e.Path.String()
}

//

type UnsupportedAttributes struct {
	Data interface{}
}

func (e *UnsupportedAttributes) Error() string {
	return fmt.Sprintf("%v", e.Data)
}
