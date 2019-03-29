package vfs

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"sync/atomic"
)

var prov FileSystem = &LocalFileSystem{}

// Default returns the root data provider. By default this is a vfs.LocalFileSystem. Consider to reconfigure it to
// a vfs.MountableFileSystem which allows arbitrary prefixes (also called mount points). Use it to configure and setup
// a virtualized filesystem structure for your app.
//
// Best practice
//
//  * Mount your static app data into /assets
//  * Implement variant and localization data at mount time not runtime, e.g.
//    /assets contains the data for a specific client with a specific locale
//    instead of a manual lookup e.g. in /assets/customer/DE_de. Keep your code
//    clean.
//  * Mount your user specific data into something like /media/local and
//    /media/ftp and /media/gphotos etc.
//
func Default() FileSystem {
	return prov
}

// SetDefault updates the default data provider. See also #Default()
func SetDefault(provider FileSystem) {
	prov = provider
}

// Read opens the given resource for reading. May optionally also implement os.Seeker. If called on a directory
// UnsupportedOperationError is returned. Delegates to Default()#Open.
func Read(path Path) (Resource, error) {
	return Default().Open(path.String(), os.O_RDONLY, 0)
}

// Write opens the given resource for writing. Removes and recreates the file. May optionally also implement os.Seeker.
// If elements of the path do not exist, they are created implicitly. Delegates to Default()#Open.
func Write(path Path) (Resource, error) {
	return Default().Open(path.String(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
}

// Delete a path entry and all contained children. It is not considered an error to delete a non-existing resource.
// Delegates to Default()#Delete.
func Delete(path Path) error {
	return Default().Delete(path.String())
}

// ReadAttrs reads Attributes. Every implementation must support ResourceInfo. Delegates to Default()#ReadAttrs.
func ReadAttrs(path Path, dest interface{}) error {
	return Default().ReadAttrs(path.String(), dest)
}

// WriteAttrs writes Attributes. This is an optional implementation and may simply return UnsupportedOperationError.
// Delegates to Default()#WriteAttrs.
func WriteAttrs(path Path, src interface{}) error {
	return Default().WriteAttrs(path.String(), src)
}

// MkDirs tries to create the given path hierarchy. If path already denotes a directory nothing happens. If any path
// segment already refers a file, an error must be returned. Delegates to Default()#MkDirs.
func MkDirs(path Path) error {
	return Default().MkDirs(path.String())
}

// Rename moves a file from the old to the new path. If oldPath does not exist, ResourceNotFoundError is returned.
// If newPath exists, it will be replaced. Delegates to Default()#Rename.
func Rename(oldPath Path, newPath Path) error {
	return Default().Rename(oldPath.String(), newPath.String())
}

// ReadDir is utility method to simply list a directory listing as *ResourceInfo, which is supported by all
// DataProviders.
func ReadDir(path Path) ([]ResourceInfo, error) {
	res, err := Default().ReadDir(path.String(), nil)
	if err != nil {
		return nil, err
	}
	// a little bit of premature optimization
	expectedEntries := 0
	if res.Size() > 0 {
		if res.Size() > math.MaxInt32 {
			return nil, fmt.Errorf("to many entries: %v", res.Size())
		}
		expectedEntries = int(res.Size())
	}
	list := make([]ResourceInfo, expectedEntries)[0:0]
	for res.Next() {
		row := &DefaultResourceInfo{}
		err = res.Scan(row)
		if err != nil {
			return list, err
		}
		list = append(list, row)
	}
	return list, res.Err()
}

// ReadDirRecur fully reads the given directory recursively and returns entries with full qualified paths.
func ReadDirRecur(path Path) ([]*PathEntry, error) {
	res := make([]*PathEntry, 0)
	err := Walk(path, func(path Path, info ResourceInfo, err error) error {
		if err != nil {
			return err
		}
		res = append(res, &PathEntry{path, info})
		return nil
	})
	if err != nil {
		return res, err
	}
	return res, nil
}

// A WalkClosure is invoked for each entry in Walk, as long as no error is returned and entries are available.
type WalkClosure func(path Path, info ResourceInfo, err error) error

// Walk recursively goes down the entire path hierarchy starting at the given path
func Walk(path Path, each WalkClosure) error {
	res, err := Default().ReadDir(path.String(), nil)
	if err != nil {
		return err
	}

	for res.Next() {
		tmp := &DefaultResourceInfo{}
		err := res.Scan(tmp)
		if err != nil {
			// the dev may decide to ignore errors and continue walking, e.g. due to permission denied
			shouldBreak := each(path, nil, err)
			if shouldBreak != nil {
				return shouldBreak
			}
			return nil

		}

		//delegate call
		err = each(path.Child(tmp.Name()), tmp, nil)
		if err != nil {
			return err
		}

		if tmp.Mode().IsDir() {
			return Walk(path.Child(tmp.Name()), each)
		}
		return nil
	}
	return res.Err()
}

// A PathEntry simply provides a Path and the related ResourceInfo
type PathEntry struct {
	Path     Path
	Resource ResourceInfo
}

// Equals checks for equality with another PathEntry
func (e *PathEntry) Equals(other interface{}) bool {
	if e == nil || other == nil {
		return false
	}
	if o, ok := other.(*PathEntry); ok {
		return o.Path == e.Path && Equals(o.Resource, e.Resource)
	}
	return false
}

// ReadAll loads the entire resource into memory. Only use it, if you know that it fits into memory
func ReadAll(path Path) ([]byte, error) {
	reader, err := Read(path)
	if err != nil {
		return nil, err
	}
	defer silentClose(reader)

	buf := &bytes.Buffer{}
	_, err = io.Copy(buf, reader)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// WriteAll just puts the given data into the path
func WriteAll(path Path, data []byte) (int, error) {
	writer, err := Write(path)
	if err != nil {
		return 0, err
	}
	defer silentClose(writer)

	n, err := writer.Write(data)
	if err != nil {
		return n, err
	}
	if n != len(data) {
		return n, fmt.Errorf("provider %v.Write has violated the Write contract", Default())
	}
	return n, nil
}

// Stat simply allocates a ResourceInfo and reads it, which must be supported by all implementations.
func Stat(path Path) (ResourceInfo, error) {
	info := &DefaultResourceInfo{}
	err := Default().ReadAttrs(path.String(), info)
	if err != nil {
		return info, err
	}
	return info, nil
}

// CopyOptions is used to define the process of copying.
type CopyOptions struct {
	cancelled int32

	// OnScan is called while scanning the source
	OnScan func(obj Path, objects int64, bytes int64)

	// OnCopied is called after each transferred object.
	OnCopied func(obj Path, objectsTransferred int64, bytesTransferred int64)

	// OnProgress is called for each file which is progress of being copied
	OnProgress func(src Path, dst Path, bytes int64, size int64)

	// OnError is called if an error occurs. If an error is returned, the process is stopped and the returned error is returned.
	OnError func(object Path, err error) error
}

// Cancel is used to signal an interruption
func (o *CopyOptions) Cancel() {
	atomic.StoreInt32(&o.cancelled, 1)
}

// IsCancelled checks if the copy process has been cancelled
func (o *CopyOptions) IsCancelled() bool {
	if o == nil {
		return false
	}
	return atomic.LoadInt32(&o.cancelled) == 1
}

func (o *CopyOptions) onProgress(src Path, dst Path, bytes int64, size int64) {
	if o == nil || o.OnProgress == nil {
		return
	}
	o.OnProgress(src, dst, bytes, size)
}

func (o *CopyOptions) onScan(obj Path, objects int64, bytes int64) {
	if o == nil || o.OnScan == nil {
		return
	}
	o.OnScan(obj, objects, bytes)
}

func (o *CopyOptions) onCopied(obj Path, objectsTransferred int64, bytesTransferred int64) {
	if o == nil || o.OnCopied == nil {
		return
	}
	o.OnCopied(obj, objectsTransferred, bytesTransferred)
}

func (o *CopyOptions) onError(object Path, err error) error {
	if o == nil || o.OnError == nil {
		return err
	}
	return o.OnError(object, err)
}

// Copy performs a copy from src to dst. Dst is always removed and replaced with the contents of src.
// The copy options can be nil and can be used to get detailed information on the progress
func Copy(src Path, dst Path, options *CopyOptions) error {

	// first try to stat
	info, err := Stat(src)
	if err != nil {
		return err
	}

	// cleanup dst
	err = Delete(dst)
	if err != nil {
		return err
	}

	if info.Mode().IsDir() {
		var objectsFound int64
		var bytesFound int64
		var objectsProcessed int64
		var bytesProcessed int64
		// collect info
		list := make([]*PathEntry, 0)
		err = Walk(src, func(path Path, info ResourceInfo, err error) error {
			if err != nil {
				return options.onError(path, err)
			}
			list = append(list, &PathEntry{path, info})
			objectsFound++
			if info.Mode().IsRegular() {
				bytesFound += info.Size()
			}
			options.onScan(path, objectsFound, bytesFound)
			return nil
		})

		if err != nil {
			return err
		}

		//walk through, directory are first
		for _, entry := range list {
			dstPath := ConcatPaths(dst, entry.Path.TrimPrefix(src))
			if entry.Resource.Mode().IsDir() {
				err := MkDirs(dstPath)
				if err != nil {
					err = options.onError(dstPath, err)
					if err != nil {
						return err
					}
				}
				objectsProcessed++
				options.onCopied(entry.Path, objectsProcessed, bytesProcessed)
			} else if entry.Resource.Mode().IsRegular() {
				reader, err := Read(entry.Path)
				if err != nil {
					return err
				}
				writer, err := Write(dstPath)
				if err != nil {
					silentClose(reader)
					return err
				}
				written, err := copyBuffer(entry.Path, dstPath, entry.Resource.Size(), reader, writer, nil, options)
				silentClose(reader)
				silentClose(writer)
				if err != nil {
					err = options.onError(dstPath, err)
					if err != nil {
						return err
					}
					return err
				}
				objectsProcessed++
				bytesProcessed += written
				options.onCopied(entry.Path, objectsProcessed, bytesProcessed)

			} else {
				err = options.onError(entry.Path, fmt.Errorf("unsupported path object %v", entry.Path))
				if err != nil {
					return err
				}
			}
		}
		return nil
	}

	options.onScan(src, 1, info.Size())
	//just copy file
	reader, err := Read(src)
	if err != nil {
		return err
	}
	defer silentClose(reader)
	writer, err := Write(dst)
	if err != nil {
		return err
	}
	defer silentClose(writer)
	written, err := copyBuffer(src, dst, info.Size(), reader, writer, nil, options)
	if err != nil {
		return err
	}
	options.onCopied(src, 1, written)
	return nil

}

func copyBuffer(srcPath Path, dstPath Path, totalSize int64, src io.Reader, dst io.Writer, buf []byte, options *CopyOptions) (written int64, err error) {
	if buf == nil {
		size := 32 * 1024
		buf = make([]byte, size)
	}
	for {
		if options.IsCancelled() {
			err = &CancellationError{}
			break
		}

		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			options.onProgress(srcPath, dstPath, written, totalSize)
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}

// A genericDirEntList is a simple implementation for fixed size result sets providing only *ResourceInfo targets.
type genericDirEntList struct {
	currentIdx int64
	count      int64
	getAt      func(idx int64, dst ResourceInfo) error
}

func (d *genericDirEntList) Next() bool {
	if d.currentIdx < d.count {
		d.currentIdx++
		return true
	}
	return false
}

// Err never returns an error, because the count is known at construction time, and seeking errors cannot occur.
func (d *genericDirEntList) Err() error {
	return nil
}

func (d *genericDirEntList) Scan(dest interface{}) error {
	if out, ok := dest.(ResourceInfo); ok {
		if d.currentIdx >= d.count {
			return d.getAt(d.count-1, out)
		}
		return d.getAt(d.currentIdx-1, out)
	}
	return &UnsupportedAttributesError{dest, nil}
}

func (d *genericDirEntList) Size() int64 {
	return d.count
}

func (d *genericDirEntList) Close() error {
	return nil
}

// NewDirEntList is a utility function to simply wrap a function into a lazy DirEntList implementation
func NewDirEntList(size int64, getter func(idx int64, dst ResourceInfo) error) DirEntList {
	return &genericDirEntList{0, size, getter}
}

// NewResourceFromReader wraps a reader and returns a Resource implementation which only delegates the Read method
// and only supports limited (forward) Seek support by just discarding 1 byte after another.
// Delegates also the Close call, if reader also implements Closeable.
func NewResourceFromReader(reader io.Reader) Resource {
	return &resourceReader{reader}
}

type resourceReader struct {
	delegate io.Reader
}

func (r *resourceReader) ReadAt(b []byte, off int64) (n int, err error) {
	return 0, &UnsupportedOperationError{}
}

func (r *resourceReader) Read(p []byte) (n int, err error) {
	return r.delegate.Read(p)
}

func (r *resourceReader) WriteAt(b []byte, off int64) (n int, err error) {
	return 0, &UnsupportedOperationError{}
}

func (r *resourceReader) Write(p []byte) (n int, err error) {
	return 0, &UnsupportedOperationError{}
}

func (r *resourceReader) Seek(offset int64, whence int) (int64, error) {
	if whence == io.SeekCurrent && offset >= 0 {
		count := int64(0)
		tmp := make([]byte, 1)
		for i := int64(0); i < offset; i++ {
			n, err := r.delegate.Read(tmp)
			count += int64(n)
			if err != nil {
				return count, err
			}
		}
		return count, nil
	}
	return 0, &UnsupportedOperationError{}
}

func (r *resourceReader) Close() error {
	if closer, ok := r.delegate.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// NewResourceFromWriter wraps a writer and returns a Resource implementation which only delegates the Write method.
// Delegates also the Close call, if writer also implements Closeable.
func NewResourceFromWriter(writer io.Writer) Resource {
	return &resourceWriter{writer}
}

type resourceWriter struct {
	delegate io.Writer
}

func (r *resourceWriter) ReadAt(b []byte, off int64) (n int, err error) {
	return 0, &UnsupportedOperationError{}
}

func (r *resourceWriter) Read(p []byte) (n int, err error) {
	return 0, &UnsupportedOperationError{}
}

func (r *resourceWriter) WriteAt(b []byte, off int64) (n int, err error) {
	return 0, &UnsupportedOperationError{}
}

func (r *resourceWriter) Write(p []byte) (n int, err error) {
	return r.delegate.Write(p)
}

func (r *resourceWriter) Seek(offset int64, whence int) (int64, error) {
	return 0, &UnsupportedOperationError{}
}

func (r *resourceWriter) Close() error {
	if closer, ok := r.delegate.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

var _ ResourceInfo = (*DefaultResourceInfo)(nil)

// A DefaultResourceInfo is the default implementation of the ResourceInfo interface
type DefaultResourceInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime int64
}

// SetName see ResourceInfo#SetName
func (r *DefaultResourceInfo) SetName(name string) {
	r.name = name
}

// Name see ResourceInfo#Name
func (r *DefaultResourceInfo) Name() string {
	return r.name
}

// SetSize see ResourceInfo#SetSize
func (r *DefaultResourceInfo) SetSize(size int64) {
	r.size = size
}

// Size see ResourceInfo#Size
func (r *DefaultResourceInfo) Size() int64 {
	return r.size
}

// SetMode see ResourceInfo#SetMode
func (r *DefaultResourceInfo) SetMode(mode os.FileMode) {
	r.mode = mode
}

// Mode see ResourceInfo#Mode
func (r *DefaultResourceInfo) Mode() os.FileMode {
	return r.mode
}

// SetModTime see ResourceInfo#SetModTime
func (r *DefaultResourceInfo) SetModTime(time int64) {
	r.modTime = time
}

// ModTime see ResourceInfo#ModTime
func (r *DefaultResourceInfo) ModTime() int64 {
	return r.modTime
}

// Equals checks for equality with another PathEntry
func (r *DefaultResourceInfo) Equals(other interface{}) bool {
	if r == nil || other == nil {
		return false
	}
	if o, ok := other.(DefaultResourceInfo); ok {
		return o.name == r.name && o.size == r.size && o.modTime == r.modTime && o.mode == r.mode
	}
	return false
}

// Equals returns true if the values defined by ResourceInfo are equal.
// However it does not inspect or check other fields or values.
func Equals(a ResourceInfo, b ResourceInfo) bool {
	if a == nil || b == nil {
		return false
	}
	return a.Name() == b.Name() && a.Size() == b.Size() && a.ModTime() == b.ModTime() && a.Mode() == b.Mode()
}
