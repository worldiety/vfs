package vfs

import (
	"context"
	"io"
	"reflect"
)

var _ FileSystem = (*AbstractFileSystem)(nil)

const absVFSName = "AbstractFileSystem"

// An AbstractFileSystem can be embedded to bootstrap a new implementation faster. One can even
// embed an uninitialized pointer type, which will return ENOSYS on each method call.
type AbstractFileSystem struct {
	FConnect func(ctx context.Context, options interface{}) (interface{}, error)

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

	FWriteAttrs func(ctx context.Context, path string, src interface{}) (Entry, error)

	FReadBucket func(ctx context.Context, path string, options interface{}) (ResultSet, error)

	FInvoke func(ctx context.Context, endpoint string, args ...interface{}) (interface{}, error)

	FMkBucket func(ctx context.Context, path string, options interface{}) error

	FRename func(ctx context.Context, oldPath string, newPath string) error

	FSymLink func(ctx context.Context, oldPath string, newPath string) error

	FHardLink func(ctx context.Context, oldPath string, newPath string) error

	FRefLink func(ctx context.Context, oldPath string, newPath string) error

	FClose func() error

	FString func() string

	FFireEvent func(ctx context.Context, path string, event interface{}) error
}

func (v *AbstractFileSystem) FireEvent(ctx context.Context, path string, event interface{}) error {
	if v == nil || v.FFireEvent == nil {
		return NewENOSYS("FireEvent not supported", absVFSName)
	}
	return v.FFireEvent(ctx, path, event)
}

func (v *AbstractFileSystem) Connect(ctx context.Context, path string, options interface{}) (interface{}, error) {
	if v == nil || v.FConnect == nil {
		return nil, NewENOSYS("Connect not supported", absVFSName)
	}
	return v.FConnect(ctx, options)
}

func (v *AbstractFileSystem) Disconnect(ctx context.Context, path string) error {
	if v == nil || v.FDisconnect == nil {
		return NewENOSYS("Disconnect not supported", absVFSName)
	}
	return v.FDisconnect(ctx)
}

func (v *AbstractFileSystem) AddListener(ctx context.Context, path string, listener ResourceListener) (int, error) {
	if v == nil || v.FAddListener == nil {
		return 0, NewENOSYS("AddListener not supported", v)
	}
	return v.FAddListener(ctx, path, listener)
}

func (v *AbstractFileSystem) RemoveListener(ctx context.Context, handle int) error {
	if v == nil || v.FRemoveListener == nil {
		return NewENOSYS("RemoveListener not supported", v)
	}
	return v.FRemoveListener(ctx, handle)
}

func (v *AbstractFileSystem) Begin(ctx context.Context, path string, options interface{}) (context.Context, error) {
	if v == nil || v.FBegin == nil {
		return nil, NewENOSYS("Begin not supported", v)
	}
	return v.FBegin(ctx, options)
}

func (v *AbstractFileSystem) Commit(ctx context.Context) error {
	if v == nil || v.FCommit == nil {
		return NewENOSYS("Commit not supported", v)
	}
	return v.FCommit(ctx)
}

func (v *AbstractFileSystem) Rollback(ctx context.Context) error {
	if v == nil || v.FRollback == nil {
		return NewENOSYS("Rollback not supported", v)
	}
	return v.FRollback(ctx)
}

func (v *AbstractFileSystem) Open(ctx context.Context, path string, flag int, options interface{}) (Blob, error) {
	if v == nil || v.FOpen == nil {
		return nil, NewENOSYS("Open not supported", v)
	}
	return v.FOpen(ctx, path, flag, options)
}

func (v *AbstractFileSystem) Delete(ctx context.Context, path string) error {
	if v == nil || v.FDelete == nil {
		return NewENOSYS("Delete not supported", v)
	}
	return v.FDelete(ctx, path)
}

func (v *AbstractFileSystem) ReadAttrs(ctx context.Context, path string, options interface{}) (Entry, error) {
	if v == nil || v.FReadForks == nil {
		return nil, NewENOSYS("ReadAttrs not supported", v)
	}
	return v.FReadAttrs(ctx, path, options)
}

func (v *AbstractFileSystem) ReadForks(ctx context.Context, path string) ([]string, error) {
	if v == nil || v.FReadForks == nil {
		return nil, NewENOSYS("ReadForks not supported", v)
	}
	return v.FReadForks(ctx, path)
}

func (v *AbstractFileSystem) WriteAttrs(ctx context.Context, path string, src interface{}) (Entry, error) {
	if v == nil || v.FWriteAttrs == nil {
		return nil, NewENOSYS("WriteAttrs not supported", v)
	}
	return v.FWriteAttrs(ctx, path, src)
}

func (v *AbstractFileSystem) ReadBucket(ctx context.Context, path string, options interface{}) (ResultSet, error) {
	if v == nil || v.FReadBucket == nil {
		return nil, NewENOSYS("ReadBucket not supported", v)
	}
	return v.FReadBucket(ctx, path, options)
}

func (v *AbstractFileSystem) Invoke(ctx context.Context, endpoint string, args ...interface{}) (interface{}, error) {
	if v == nil || v.FInvoke == nil {
		return nil, NewENOSYS("Invoke not supported", v)
	}
	return v.FInvoke(ctx, endpoint, args)
}

func (v *AbstractFileSystem) MkBucket(ctx context.Context, path string, options interface{}) error {
	if v == nil || v.FMkBucket == nil {
		return NewENOSYS("MkBucket not supported", v)
	}
	return v.FMkBucket(ctx, path, options)
}

func (v *AbstractFileSystem) Rename(ctx context.Context, oldPath string, newPath string) error {
	if v == nil || v.FRename == nil {
		return NewENOSYS("Rename not supported", v)
	}
	return v.FRename(ctx, oldPath, newPath)
}

func (v *AbstractFileSystem) SymLink(ctx context.Context, oldPath string, newPath string) error {
	if v == nil || v.FSymLink == nil {
		return NewENOSYS("SymLink not supported", v)
	}
	return v.FSymLink(ctx, oldPath, newPath)
}

func (v *AbstractFileSystem) HardLink(ctx context.Context, oldPath string, newPath string) error {
	if v == nil || v.FHardLink == nil {
		return NewENOSYS("HardLink not supported", v)
	}
	return v.FHardLink(ctx, oldPath, newPath)
}

func (v *AbstractFileSystem) RefLink(ctx context.Context, oldPath string, newPath string) error {
	if v == nil || v.FRefLink == nil {
		return NewENOSYS("RefLink not supported", v)
	}
	return v.FRefLink(ctx, oldPath, newPath)
}

func (v *AbstractFileSystem) Close() error {
	if v == nil || v.FClose == nil {
		return NewENOSYS("Close not supported", v)
	}
	return v.FClose()
}

func (v *AbstractFileSystem) String() string {
	if v == nil || v.FString == nil {
		return "AbstractFileSystem"
	}
	return v.FString()
}

// DefaultEntry is a minimal type, useful to create simple VFS implementations. It may make sense to
// create a custom type which just fulfills the contract and avoids some GC pressure.
type DefaultEntry struct {
	Id       string      // Id must be at least unique per bucket
	IsBucket bool        // IsBucket denotes the directory or folder flag
	Length   int64       // Length in bytes, if unknown set to -1
	Data     interface{} // Data is the original payload, if any, otherwise nil
}

// Name returns the (unique) Id
func (a *DefaultEntry) Name() string {
	return a.Id
}

// IsDir returns the IsBucket flag
func (a *DefaultEntry) IsDir() bool {
	return a.IsBucket
}

// Sys returns any internal or implementation specific payload, which may be nil.
func (a *DefaultEntry) Sys() interface{} {
	return a.Data
}

// Size returns the Length value
func (a *DefaultEntry) Size() int64 {
	return a.Length
}

// DefaultResultSet is a minimal type, useful to create simple VFS implementation. However you should usually
// provide a custom implementation to give access to the raw data (see #Sys()), e.g. the original parsed
// JSON data structures.
type DefaultResultSet struct {
	Entries []*DefaultEntry
}

func (r *DefaultResultSet) ReadAttrs(idx int, args interface{}) Entry {
	entry := r.Entries[idx]
	switch t := args.(type) {
	case map[string]interface{}:
		t[mapEntryName] = entry.Id
		t[mapEntrySize] = entry.Size
		t[mapEntryIsDir] = entry.IsBucket
		t[mapEntrySys] = entry.Data
		return AbsMapEntry(t)
	case *DefaultEntry:
		t.Id = entry.Id
		t.Length = entry.Length
		t.IsBucket = entry.IsBucket
		t.Data = entry.Data
		return t
	default:
		return r.ReadAttrs(idx, make(map[string]interface{}))
	}
}

// Len always returns len(Entries)
func (r *DefaultResultSet) Len() int {
	return len(r.Entries)
}

// Total always returns Len
func (r *DefaultResultSet) Total() int64 {
	return int64(r.Len())
}

// Pages always returns 1
func (r *DefaultResultSet) Pages() int64 {
	return 1
}

// Next always returns EOF
func (r *DefaultResultSet) Next(ctx context.Context) error {
	return eof
}

// Sys always returns []*DefaultEntry
func (r *DefaultResultSet) Sys() interface{} {
	return r.Entries
}

//==

// A BlobAdapter is used to wrap something like io.Reader into a Blob, whose other methods (e.g. other than Read)
// will simply return ENOSYS.
type BlobAdapter struct {
	// Delegate can be anything like io.ReaderAt, io.WriterAt, io.Writer, io.Closer, io.Reader and io.Seeker
	// in all combinations.
	Delegate interface{}
}

func (d *BlobAdapter) ReadAt(b []byte, off int64) (n int, err error) {
	if reader, ok := d.Delegate.(io.ReaderAt); ok {
		return reader.ReadAt(b, off)
	}
	return 0, NewENOSYS("ReadAt not supported", d)
}

func (d *BlobAdapter) Read(p []byte) (n int, err error) {
	if reader, ok := d.Delegate.(io.Reader); ok {
		return reader.Read(p)
	}
	return 0, NewENOSYS("Read not supported", d)
}

func (d *BlobAdapter) WriteAt(b []byte, off int64) (n int, err error) {
	if writer, ok := d.Delegate.(io.WriterAt); ok {
		return writer.WriteAt(b, off)
	}
	return 0, NewENOSYS("WriteAt not supported", d)
}

func (d *BlobAdapter) Write(p []byte) (n int, err error) {
	if writer, ok := d.Delegate.(io.Writer); ok {
		return writer.Write(p)
	}
	return 0, NewENOSYS("Write not supported", d)
}

func (d *BlobAdapter) Seek(offset int64, whence int) (int64, error) {
	if seeker, ok := d.Delegate.(io.Seeker); ok {
		return seeker.Seek(offset, whence)
	}
	return 0, NewENOSYS("Seek not supported", d)
}

func (d *BlobAdapter) Close() error {
	if closer, ok := d.Delegate.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// NewENOSYS is a helper function to create an error which signals that an implementation is not available. The
// msg should indicate what is not implemented and who can be used to give a type hint for further inspection.
// If used correctly
func NewENOSYS(msg string, who interface{}) *DefaultError {
	strWho := "nil"
	if who != nil {
		strWho = reflect.TypeOf(who).String()
		if str, ok := who.(string); ok {
			strWho = str
		}
	}

	return &DefaultError{msg + ": " + strWho, ENOSYS, nil, nil}
}
