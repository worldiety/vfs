package vfs

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
)

type Fields = map[string]interface{}

type builderPathListener struct {
	path     Path
	listener ResourceListener
}

func (l builderPathListener) matches(path string) bool {
	return Path(path) == l.path
}

const mapEntryName = "n"
const mapEntrySize = "s"
const mapEntryIsDir = "d"
const mapEntrySys = "y"

const EventBeforeOpen = "BeforeOpen"
const EventBeforeBucketRead = "BeforeBucketRead"
const EventBeforeDelete = "BeforeDelete"
const EventBeforeReadAttrs = "BeforeReadAttrs"
const EventBeforeSymLink = "BeforeSymLink"
const EventBeforeHardLink = "BeforeHardLink"
const EventBeforeMkBucket = "BeforeMkBucket"

// The Builder is used to create a VFS from scratch in a simpler way. A list of included batteries:
//
//   * Supports a lot of events for read/write/delete/update etc. using string constants (Event*)
//   * Implementations may always provide a Size() but may return -1 or be incorrect
//   * Optimized reads in ReadAttrs if args is map[string]interface{}
//   * Each undefined method will return ENOSYS error
//   * Listeners can be used to intercept operations (before semantic)
type Builder struct {
	vfs               *AbstractFileSystem
	buckets           []*BucketBuilder
	blobs             []*BlobBuilder
	fallbackDelete    func(ctx context.Context, path string) error
	fallbackReadAttrs func(ctx context.Context, path string, options interface{}) (Entry, error)
	listeners         map[int]*builderPathListener
	lastHandle        int
}

func (b *Builder) nextHandle() int {
	b.lastHandle++
	return b.lastHandle
}

func (b *Builder) debugName() string {
	return b.vfs.String()
}

func (b *Builder) ensureInit() {
	if b.vfs == nil {
		b.vfs = &AbstractFileSystem{}
		b.listeners = make(map[int]*builderPathListener)
		b.vfs.FConnect = func(ctx context.Context, options interface{}) error {
			return newENOSYS("Connect not supported", b.debugName())
		}
		b.vfs.FClose = func() error {
			return nil // intentionally always no-op
		}

		b.vfs.FDisconnect = func(ctx context.Context) error {
			return newENOSYS("Disconnect not supported", b.debugName())
		}

		b.vfs.FRemoveListener = func(ctx context.Context, handle int) error {
			delete(b.listeners, handle)
			return nil
		}

		b.vfs.FAddListener = func(ctx context.Context, path string, listener ResourceListener) (hnd int, err error) {
			hnd = b.nextHandle()
			b.listeners[hnd] = &builderPathListener{
				path:     Path(path),
				listener: listener,
			}
			return hnd, nil
		}

		b.vfs.FFireEvent = func(ctx context.Context, path string, event interface{}) error {
			for _, listener := range b.listeners {
				if listener.matches(path) {
					err := listener.listener.OnEvent(path, event)
					if err != nil {
						return err
					}
				}
			}
			return nil
		}

		b.vfs.FBegin = func(ctx context.Context, options interface{}) (i context.Context, e error) {
			return nil, newENOSYS("Begin transaction not supported", b.debugName())
		}

		b.vfs.FCommit = func(ctx context.Context) error {
			return newENOSYS("Commit transaction not supported", b.debugName())
		}

		b.vfs.FRollback = func(ctx context.Context) error {
			return newENOSYS("Rollback transaction not supported", b.debugName())
		}

		b.vfs.FOpen = func(ctx context.Context, path string, flag int, options interface{}) (blob Blob, e error) {
			return nil, newENOSYS("Open not supported", b.debugName())
		}

		b.vfs.FDelete = func(ctx context.Context, path string) error {
			return newENOSYS("Delete not supported", b.debugName())
		}

		b.vfs.FReadAttrs = func(ctx context.Context, path string, options interface{}) (Entry, error) {
			return nil, newENOSYS("ReadAttrs not supported", b.debugName())
		}

		b.vfs.FReadForks = func(ctx context.Context, path string) (strings []string, e error) {
			return nil, newENOSYS("ReadForks not supported", b.debugName())
		}

		b.vfs.FWriteAttrs = func(ctx context.Context, path string, src interface{}) (Entry, error) {
			return nil, newENOSYS("WriteAttrs not supported", b.debugName())
		}

		b.vfs.FReadBucket = func(ctx context.Context, path string, options interface{}) (set ResultSet, e error) {
			return nil, newENOSYS("ReadBucket not supported", b.debugName())
		}

		b.vfs.FInvoke = func(ctx context.Context, endpoint string, args ...interface{}) (i interface{}, e error) {
			return nil, newENOSYS("Invoke not supported", b.debugName())
		}

		b.vfs.FMkBucket = func(ctx context.Context, path string, options interface{}) error {
			return newENOSYS("MkBucket not supported", b.debugName())
		}

		b.vfs.FRename = func(ctx context.Context, oldPath string, newPath string) error {
			return newENOSYS("Rename not supported", b.debugName())
		}

		b.vfs.FSymLink = func(ctx context.Context, oldPath string, newPath string) error {
			return newENOSYS("SymLink not supported", b.debugName())
		}

		b.vfs.FHardLink = func(ctx context.Context, oldPath string, newPath string) error {
			return newENOSYS("HardLink not supported", b.debugName())
		}

		b.vfs.FRefLink = func(ctx context.Context, oldPath string, newPath string) error {
			return newENOSYS("RefLink not supported", b.debugName())
		}
		b.vfs.FString = func() string {
			return "AbstractVirtualFilesystem"
		}
	}
}

func (b *Builder) Create() FileSystem {
	buckets := b.buckets
	blobs := b.blobs

	// open blobs behavior
	if len(blobs) > 0 {
		b.vfs.FOpen = func(ctx context.Context, path string, flag int, options interface{}) (blob Blob, e error) {
			err := b.vfs.FireEvent(ctx, path, EventBeforeOpen)
			if err != nil {
				return nil, err
			}
			for _, blob := range blobs {
				for _, matcher := range blob.matchPatterns {
					if matcher.isMatching(Path(path)) {
						if blob.open != nil {
							return blob.open(ctx, path, flag, options)
						}

						if flag == os.O_RDONLY && blob.reader != nil {
							return blob.reader(ctx, path, flag, options)
						}

						if flag != os.O_RDONLY && blob.writer != nil {
							return blob.writer(ctx, path, flag, options)
						}
					}
				}
			}

			// no matching blob found
			return nil, &DefaultError{Message: "unmatched blob: " + path, Code: ENOENT, DetailsPayload: Path(path)}
		}
	}

	// ReadBuckets behavior
	if len(buckets) > 0 {
		b.vfs.FReadBucket = func(ctx context.Context, path string, options interface{}) (set ResultSet, e error) {
			err := b.vfs.FireEvent(ctx, path, EventBeforeBucketRead)
			if err != nil {
				return nil, err
			}
			for _, bucket := range buckets {
				for _, matcher := range bucket.matchPatterns {
					if matcher.isMatching(Path(path)) {
						return bucket.onRead(ctx, path, options)
					}
				}
			}
			// no matching bucket found
			return nil, &DefaultError{Message: "unmatched bucket: " + path, Code: ENOENT, DetailsPayload: Path(path)}
		}
	}

	// Mixed behavior
	if len(buckets) > 0 || len(blobs) > 0 {
		// delete
		b.vfs.FDelete = func(ctx context.Context, path string) error {
			err := b.vfs.FireEvent(ctx, path, EventBeforeDelete)
			if err != nil {
				return err
			}
			for _, bucket := range buckets {
				for _, matcher := range bucket.matchPatterns {
					if bucket.delete != nil && matcher.isMatching(Path(path)) {
						return bucket.delete(ctx, path)
					}
				}
			}

			for _, blob := range blobs {
				for _, matcher := range blob.matchPatterns {
					if blob.delete != nil && matcher.isMatching(Path(path)) {
						return blob.delete(ctx, path)
					}
				}
			}
			if b.fallbackDelete != nil {
				return b.fallbackDelete(ctx, path)
			}

			// no matching bucket found, this is not an error by spec, because the resource is absent anyway
			return nil
		}

		// read attributes
		b.vfs.FReadAttrs = func(ctx context.Context, path string, options interface{}) (Entry, error) {
			err := b.vfs.FireEvent(ctx, path, EventBeforeReadAttrs)
			if err != nil {
				return nil, err
			}
			/*for _, bucket := range buckets {
				for _, matcher := range bucket.matchPatterns {
					if bucket.delete != nil && matcher.isMatching(Path(path)) {
						return bucket.(ctx, path)
					}
				}
			}

			for _, blob := range blobs {
				for _, matcher := range blob.matchPatterns {
					if blob.delete != nil && matcher.isMatching(Path(path)) {
						return blob.delete(ctx, path)
					}
				}
			}*/
			if b.fallbackReadAttrs != nil {
				return b.fallbackReadAttrs(ctx, path, options)
			}

			// no matching bucket found, this is not an error by spec, because the resource is absent anyway
			return nil, &DefaultError{Message: "ReadAttrs: " + path, Code: ENOENT, DetailsPayload: Path(path)}
		}

	}

	// clear this builder, to avoid inconsistent vfs instances, if developer reuses the builder
	vfs := b.vfs
	b.Reset()

	return vfs
}

func (b *Builder) Symlink(f func(ctx context.Context, oldPath Path, newPath Path) error) *Builder {
	b.vfs.FSymLink = func(ctx context.Context, oldPath string, newPath string) error {
		err := b.vfs.FireEvent(ctx, oldPath+string(filepath.ListSeparator)+oldPath, EventBeforeSymLink)
		if err != nil {
			return err
		}
		return f(ctx, Path(oldPath), Path(newPath))
	}
	return b
}

func (b *Builder) Hardlink(f func(ctx context.Context, oldPath Path, newPath Path) error) *Builder {
	b.vfs.FHardLink = func(ctx context.Context, oldPath string, newPath string) error {
		err := b.vfs.FireEvent(ctx, oldPath+string(filepath.ListSeparator)+oldPath, EventBeforeHardLink)
		if err != nil {
			return err
		}
		return f(ctx, Path(oldPath), Path(newPath))
	}
	return b
}

// Delete has lowest priority, after all blob and bucket matches have been checked
func (b *Builder) Delete(f func(ctx context.Context, path Path) error) *Builder {
	b.fallbackDelete = func(_ctx context.Context, _path string) error {
		return f(_ctx, Path(_path))
	}
	return b
}

func (b *Builder) MkBucket(f func(ctx context.Context, path Path, options interface{}) error) *Builder {
	b.vfs.FMkBucket = func(ctx context.Context, path string, options interface{}) error {
		err := b.vfs.FireEvent(ctx, path, EventBeforeMkBucket)
		if err != nil {
			return err
		}
		return f(ctx, Path(path), options)
	}
	return b
}

func (b *Builder) ReadEntryAttrs(f func(ctx context.Context, path Path, dst *AbsEntry) error) *Builder {
	b.fallbackReadAttrs = func(_ctx context.Context, _path string, _dst interface{}) (Entry, error) {
		switch t := _dst.(type) {
		case *AbsEntry:
			return t, f(_ctx, Path(_path), t)
		case map[string]interface{}:
			tmp := &AbsEntry{}
			err := f(_ctx, Path(_path), tmp)
			if err != nil {
				return nil, err
			}
			t[mapEntryName] = tmp.Id
			t[mapEntryIsDir] = tmp.IsBucket
			t[mapEntrySize] = tmp.Size
			t[mapEntrySys] = tmp.Data
			return AbsMapEntry(t), nil
		default:
			return b.fallbackReadAttrs(_ctx, _path, make(map[string]interface{}))
		}
	}
	return b
}

// Reset throws the internal state away
func (b *Builder) Reset() {
	b.buckets = nil
	b.vfs = nil
	b.blobs = nil
	b.fallbackReadAttrs = nil
	b.fallbackDelete = nil
	b.listeners = nil
}

// Details sets the name of the VFS
func (b *Builder) Details(name string, majorVersion int, minorVersion int, microVersion int) *Builder {
	b.ensureInit()
	b.vfs.FString = func() string {
		return name + " " + strconv.Itoa(majorVersion) + "." + strconv.Itoa(minorVersion) + "." + strconv.Itoa(microVersion)
	}
	return b
}

func (b *Builder) MatchBucket(pattern string) *BucketBuilder {
	builder := &BucketBuilder{parent: b}
	return builder.MatchAlso(pattern)
}

func (b *Builder) MatchBlob(pattern string) *BlobBuilder {
	builder := &BlobBuilder{parent: b}
	return builder.MatchAlso(pattern)
}

//==

type BlobBuilder struct {
	parent        *Builder
	matchPatterns []*pathMatcher
	reader        func(ctx context.Context, path string, flag int, perm interface{}) (Blob, error)
	writer        func(ctx context.Context, path string, flag int, perm interface{}) (Blob, error)
	open          func(ctx context.Context, path string, flag int, perm interface{}) (Blob, error)
	delete        func(ctx context.Context, path string) error
}

func (b *BlobBuilder) OnOpen(open func(context.Context, Path, int, interface{}) (Blob, error)) *BlobBuilder {
	b.open = func(ctx context.Context, path string, flag int, perm interface{}) (blob Blob, e error) {
		return open(ctx, Path(path), flag, perm)
	}
	b.reader = nil
	b.writer = nil
	return b
}

func (b *BlobBuilder) OnRead(open func(context.Context, Path) (io.Reader, error)) *BlobBuilder {
	b.reader = func(ctx context.Context, path string, flag int, perm interface{}) (blob Blob, e error) {
		reader, err := open(ctx, Path(path))
		if err != nil {
			return nil, err
		}
		return blobWrapper{reader}, nil
	}
	b.open = nil
	return b
}

func (b *BlobBuilder) OnWrite(open func(context.Context, Path) (io.Writer, error)) *BlobBuilder {
	b.writer = func(ctx context.Context, path string, flag int, perm interface{}) (blob Blob, e error) {
		writer, err := open(ctx, Path(path))
		if err != nil {
			return nil, err
		}
		return blobWrapper{writer}, nil
	}
	b.open = nil
	return b
}

func (b *BlobBuilder) OnDelete(delete func(context.Context, Path) error) *BlobBuilder {
	b.delete = func(ctx context.Context, path string) error {
		return delete(ctx, Path(path))
	}
	return b
}

// Match defines a pattern which is matched against a path and applies the defined data transformation rules
func (b *BlobBuilder) MatchAlso(pattern string) *BlobBuilder {
	b.matchPatterns = append(b.matchPatterns, &pathMatcher{})
	return b
}

func (b *BlobBuilder) Add() *Builder {
	b.parent.blobs = append(b.parent.blobs, b)
	return b.parent
}

//==
// The BucketBuilder helps to specify the data transformation for a buckets content or listing
type BucketBuilder struct {
	parent        *Builder
	matchPatterns []*pathMatcher
	onRead        func(ctx context.Context, path string, options interface{}) (ResultSet, error)
	delete        func(ctx context.Context, path string) error
}

func (b *BucketBuilder) OnDelete(delete func(context.Context, Path) error) *BucketBuilder {
	b.delete = func(ctx context.Context, path string) error {
		return delete(ctx, Path(path))
	}
	return b
}

// Match defines a pattern which is matched against a path and applies the defined data transformation rules
func (b *BucketBuilder) MatchAlso(pattern string) *BucketBuilder {
	b.matchPatterns = append(b.matchPatterns, &pathMatcher{})
	return b
}

// OnList configures the generic call to ReadBucket, which is either nil, *AbsEntry or map[string]interface{}.
// In any other case ReadBucket will return map[string]interface{} with the 3 fields n,s and b which
// contains name, size and the isBucket flag.
func (b *BucketBuilder) OnList(transformation func(Path) ([]AbsEntry, error)) *BucketBuilder {
	b.onRead = func(context context.Context, path string, options interface{}) (ResultSet, error) {
		entries, err := transformation(Path(path))
		if err != nil {
			return nil, err
		}
		return &entryResultSet{entries}, nil
	}
	return b
}

func (b *BucketBuilder) Add() *Builder {
	b.parent.buckets = append(b.parent.buckets, b)
	return b.parent
}

//==

type pathMatcher struct {
	path string
}

func (p *pathMatcher) isMatching(path Path) bool {
	return false
}

type AbsMapEntry map[string]interface{}

func (a AbsMapEntry) Name() string {
	return a[mapEntryName].(string)
}

func (a AbsMapEntry) IsDir() bool {
	return a[mapEntryIsDir].(bool)
}

func (a AbsMapEntry) Sys() interface{} {
	return a[mapEntrySys]
}

func (a AbsMapEntry) Size() int64 {
	return a[mapEntrySize].(int64)
}

// AbsEntry is a minimal type, especially used by VFS instances created by Builder
type AbsEntry struct {
	Id       string      // Id must be at least unique per bucket
	IsBucket bool        // IsBucket denotes the directory or folder flag
	Length   int64       // Length in bytes, if unknown set to -1
	Data     interface{} // Data is the original payload, if any, otherwise nil
}

func (a AbsEntry) Name() string {
	return a.Id
}

func (a AbsEntry) IsDir() bool {
	return a.IsBucket
}

func (a AbsEntry) Sys() interface{} {
	return a.Data
}

func (a AbsEntry) Size() int64 {
	return a.Length
}

type entryResultSet struct {
	entries []AbsEntry
}

func (r *entryResultSet) ReadAttrs(idx int, args interface{}) Entry {
	entry := r.entries[idx]
	switch t := args.(type) {
	case map[string]interface{}:
		t[mapEntryName] = entry.Id
		t[mapEntrySize] = entry.Size
		t[mapEntryIsDir] = entry.IsBucket
		t[mapEntrySys] = entry.Data
		return AbsMapEntry(t)
	case *AbsEntry:
		t.Id = entry.Id
		t.Length = entry.Length
		t.IsBucket = entry.IsBucket
		t.Data = entry.Data
		return t
	default:
		return r.ReadAttrs(idx, make(map[string]interface{}))
	}
}

func (r *entryResultSet) Len() int {
	return len(r.entries)
}

func (r *entryResultSet) Total() int64 {
	return int64(r.Len())
}

func (r *entryResultSet) Pages() int64 {
	return 1
}

func (r *entryResultSet) Next(ctx context.Context) error {
	return eof
}

// Data always returns []AbsEntry
func (r *entryResultSet) Sys() interface{} {
	return r.entries
}

//==

type blobWrapper struct {
	delegate interface{}
}

func (d blobWrapper) ReadAt(b []byte, off int64) (n int, err error) {
	if reader, ok := d.delegate.(io.ReaderAt); ok {
		return reader.ReadAt(b, off)
	}
	return 0, newENOSYS("ReadAt not supported", d)
}

func (d blobWrapper) Read(p []byte) (n int, err error) {
	if reader, ok := d.delegate.(io.Reader); ok {
		return reader.Read(p)
	}
	return 0, newENOSYS("Read not supported", d)
}

func (d blobWrapper) WriteAt(b []byte, off int64) (n int, err error) {
	if writer, ok := d.delegate.(io.WriterAt); ok {
		return writer.WriteAt(b, off)
	}
	return 0, newENOSYS("WriteAt not supported", d)
}

func (d blobWrapper) Write(p []byte) (n int, err error) {
	if writer, ok := d.delegate.(io.Writer); ok {
		return writer.Write(p)
	}
	return 0, newENOSYS("Write not supported", d)
}

func (d blobWrapper) Seek(offset int64, whence int) (int64, error) {
	if seeker, ok := d.delegate.(io.Seeker); ok {
		return seeker.Seek(offset, whence)
	}
	return 0, newENOSYS("Seek not supported", d)
}

func (d blobWrapper) Close() error {
	if closer, ok := d.delegate.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func newENOSYS(msg string, who interface{}) *DefaultError {
	strWho := reflect.TypeOf(who).String()
	if str, ok := who.(string); ok {
		strWho = str
	}
	return &DefaultError{msg + ": " + strWho, ENOSYS, nil, nil}
}
