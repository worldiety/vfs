package vfs

import (
	"context"
)

var _ FileSystem = (*AbstractFileSystem)(nil)

type AbstractFileSystem struct {
	FConnect func(ctx context.Context, options interface{}) error

	FDisconnect func(ctx context.Context) error

	FAddListener func(ctx context.Context, path string, listener ResourceListener) (int, error)

	FRemoveListener func(ctx context.Context, handle int) error

	FBegin func(ctx context.Context, options interface{}) (context.Context, error)

	FCommit func(ctx context.Context) error

	FRollback func(ctx context.Context) error

	FOpen func(ctx context.Context, path string, flag int, options interface{}) (Blob, error)

	FDelete func(ctx context.Context, path string) error

	FReadAttrs func(ctx context.Context, path string, options interface{}) (Entry, error)

	FReadForks func(ctx context.Context, path string) ([]string, error)

	FWriteAttrs func(ctx context.Context, path string, src interface{}) error

	FReadBucket func(ctx context.Context, path string, options interface{}) (ResultSet, error)

	FInvoke func(ctx context.Context, endpoint string, args ...interface{}) (interface{}, error)

	FMkBucket func(ctx context.Context, path string, options interface{}) error

	FRename func(ctx context.Context, oldPath string, newPath string) error

	FSymLink func(ctx context.Context, oldPath string, newPath string) error

	FHardLink func(ctx context.Context, oldPath string, newPath string) error

	FCopy func(ctx context.Context, oldPath string, newPath string) error

	FClose func() error

	FString func() string

	FFireEvent func(ctx context.Context, path string, event interface{}) error
}

func (v *AbstractFileSystem) FireEvent(ctx context.Context, path string, event interface{}) error {
	return v.FFireEvent(ctx, path, event)
}

func (v *AbstractFileSystem) Connect(ctx context.Context, path string, options interface{}) error {
	return v.FConnect(ctx, options)
}

func (v *AbstractFileSystem) Disconnect(ctx context.Context, path string) error {
	return v.FDisconnect(ctx)
}

func (v *AbstractFileSystem) AddListener(ctx context.Context, path string, listener ResourceListener) (int, error) {
	return v.FAddListener(ctx, path, listener)
}

func (v *AbstractFileSystem) RemoveListener(ctx context.Context, handle int) error {
	return v.FRemoveListener(ctx, handle)
}

func (v *AbstractFileSystem) Begin(ctx context.Context, path string, options interface{}) (context.Context, error) {
	return v.FBegin(ctx, options)
}

func (v *AbstractFileSystem) Commit(ctx context.Context) error {
	return v.FCommit(ctx)
}

func (v *AbstractFileSystem) Rollback(ctx context.Context) error {
	return v.FRollback(ctx)
}

func (v *AbstractFileSystem) Open(ctx context.Context, path string, flag int, options interface{}) (Blob, error) {
	return v.FOpen(ctx, path, flag, options)
}

func (v *AbstractFileSystem) Delete(ctx context.Context, path string) error {
	return v.FDelete(ctx, path)
}

func (v *AbstractFileSystem) ReadAttrs(ctx context.Context, path string, options interface{}) (Entry, error) {
	return v.FReadAttrs(ctx, path, options)
}

func (v *AbstractFileSystem) ReadForks(ctx context.Context, path string) ([]string, error) {
	return v.FReadForks(ctx, path)
}

func (v *AbstractFileSystem) WriteAttrs(ctx context.Context, path string, src interface{}) error {
	return v.FWriteAttrs(ctx, path, src)
}

func (v *AbstractFileSystem) ReadBucket(ctx context.Context, path string, options interface{}) (ResultSet, error) {
	return v.FReadBucket(ctx, path, options)
}

func (v *AbstractFileSystem) Invoke(ctx context.Context, endpoint string, args ...interface{}) (interface{}, error) {
	return v.FInvoke(ctx, endpoint, args)
}

func (v *AbstractFileSystem) MkBucket(ctx context.Context, path string, options interface{}) error {
	return v.FMkBucket(ctx, path, options)
}

func (v *AbstractFileSystem) Rename(ctx context.Context, oldPath string, newPath string) error {
	return v.FRename(ctx, oldPath, newPath)
}

func (v *AbstractFileSystem) SymLink(ctx context.Context, oldPath string, newPath string) error {
	return v.FSymLink(ctx, oldPath, newPath)
}

func (v *AbstractFileSystem) HardLink(ctx context.Context, oldPath string, newPath string) error {
	return v.FHardLink(ctx, oldPath, newPath)
}

func (v *AbstractFileSystem) Copy(ctx context.Context, oldPath string, newPath string) error {
	return v.FCopy(ctx, oldPath, newPath)
}

func (v *AbstractFileSystem) Close() error {
	return v.FClose()
}

func (v *AbstractFileSystem) String() string {
	return v.FString()
}
