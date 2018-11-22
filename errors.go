package vfs

type MountPointNotFoundError struct {
	MountPoint string
}

func (e *MountPointNotFoundError) Error() string {
	return "not found: " + e.MountPoint
}

type OperationNotSupportedError struct {
	Message string
}

func (e *OperationNotSupportedError) Error() string {
	return e.Message
}
