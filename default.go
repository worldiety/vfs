package vfs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"time"
)

var prov FileSystem = LocalFileSystem

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
func Read(path string) (io.ReadCloser, error) {
	return Default().Open(context.Background(), path, os.O_RDONLY, nil)
}

// Write opens the given resource for writing. Removes and recreates the file. May optionally also implement os.Seeker.
// If elements of the path do not exist, they are created implicitly. Delegates to Default()#Open.
func Write(path string) (io.WriteCloser, error) {
	return Default().Open(context.Background(), path, os.O_RDWR, nil)
}

// Delete a path entry and all contained children. It is not considered an error to delete a non-existing resource.
// Delegates to Default()#Delete.
func Delete(path string) error {
	return Default().Delete(context.Background(), path)
}

// ReadAttrs reads Attributes. Every implementation must support ResourceInfo. Delegates to Default()#ReadAttrs.
func ReadAttrs(path string, args interface{}) (Entry, error) {
	return Default().ReadAttrs(context.Background(), path, args)
}

// WriteAttrs writes Attributes. This is an optional implementation and may simply return UnsupportedOperationError.
// Delegates to Default()#WriteAttrs.
func WriteAttrs(path string, src interface{}) (Entry, error) {
	return Default().WriteAttrs(context.Background(), path, src)
}

// MkDirs tries to create the given path hierarchy. If path already denotes a directory nothing happens. If any path
// segment already refers a file, an error must be returned. Delegates to Default()#MkDirs.
func MkDirs(path string) error {
	return Default().MkBucket(context.Background(), path, nil)
}

// Rename moves a file from the old to the new path. If oldPath does not exist, ResourceNotFoundError is returned.
// If newPath exists, it will be replaced. Delegates to Default()#Rename.
func Rename(oldPath string, newPath string) error {
	return Default().Rename(context.Background(), oldPath, newPath)
}

// ReadBucket is a utility method to simply list a directory by querying all result set pages.
func ReadBucket(path string) ([]Entry, error) {
	list := make([]Entry, 10)[0:0]
	res, err := Default().ReadBucket(context.Background(), path, nil)
	for {
		// got error which may be EOF or something important
		if err != nil {
			if IsErr(err, EOF) {
				return list, nil
			}
			return list, err
		}

		// no error at all, collect results
		for i := 0; i < res.Len(); i++ {
			list = append(list, res.ReadAttrs(i, nil))
		}

		// query next page
		err = res.Next(context.Background())
	}

}

// ReadBucketRecur fully reads the given directory recursively and returns Entries with full qualified paths.
func ReadBucketRecur(path string) ([]*PathEntry, error) {
	res := make([]*PathEntry, 0)
	err := Walk(path, func(path string, info Entry, err error) error {
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

// A WalkClosure is invoked for each entry in Walk, as long as no error is returned and Entries are available.
type WalkClosure func(path string, info Entry, err error) error

// Walk recursively goes down the entire path hierarchy starting at the given path
func Walk(path string, each WalkClosure) error {

	res, err := Default().ReadBucket(context.Background(), path, nil)
	for {

		// got error which may be EOF or something important
		if err != nil {
			if IsErr(err, EOF) {
				return nil
			}
			failedEntry := &DefaultEntry{Id: Path(path).Name()}
			// let the dev override any error case. If an err is turned to nil, the Walk-callee will continue
			err = each(Path(path).Child(failedEntry.Name()).String(), failedEntry, err)
			return err
		}

		// no error at all, collect results
		for i := 0; i < res.Len(); i++ {
			entry := res.ReadAttrs(i, nil)
			err = each(Path(path).Child(entry.Name()).String(), entry, nil)
			if err != nil {
				return err
			}
		}

		// query next page
		err = res.Next(context.Background())
	}

}

// A PathEntry simply provides a Path and the related information
type PathEntry struct {
	Path  string
	Entry Entry
}

// Equals checks for equality with another PathEntry
func (e *PathEntry) Equals(other interface{}) bool {
	if e == nil || other == nil {
		return false
	}
	if o, ok := other.(*PathEntry); ok {
		return o.Path == e.Path && Equals(o.Entry, e.Entry)
	}
	return false
}

// ReadAll loads the entire resource into memory. Only use it, if you know that it fits into memory
func ReadAll(path string) ([]byte, error) {
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
func WriteAll(path string, data []byte) (int, error) {
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

// Stat emulates a standard library file info contract. See also #ReadAttrs() which allows a bit more control on
// how the call is made.
func Stat(path string) (os.FileInfo, error) {
	entry, err := Default().ReadAttrs(context.Background(), path, nil)
	if err != nil {
		return nil, err
	}
	return entryDelegator{entry}, nil
}

// CopyOptions is used to define the process of copying.
type CopyOptions struct {
	cancelled int32

	// OnScan is called while scanning the source
	OnScan func(obj string, objects int64, bytes int64)

	// OnCopied is called after each transferred object.
	OnCopied func(obj string, objectsTransferred int64, bytesTransferred int64)

	// OnProgress is called for each file which is progress of being copied
	OnProgress func(src string, dst string, bytes int64, size int64)

	// OnError is called if an error occurs. If an error is returned, the process is stopped and the returned error is returned.
	OnError func(object string, err error) error
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

func (o *CopyOptions) onProgress(src string, dst string, bytes int64, size int64) {
	if o == nil || o.OnProgress == nil {
		return
	}
	o.OnProgress(src, dst, bytes, size)
}

func (o *CopyOptions) onScan(obj string, objects int64, bytes int64) {
	if o == nil || o.OnScan == nil {
		return
	}
	o.OnScan(obj, objects, bytes)
}

func (o *CopyOptions) onCopied(obj string, objectsTransferred int64, bytesTransferred int64) {
	if o == nil || o.OnCopied == nil {
		return
	}
	o.OnCopied(obj, objectsTransferred, bytesTransferred)
}

func (o *CopyOptions) onError(object string, err error) error {
	if o == nil || o.OnError == nil {
		return err
	}
	return o.OnError(object, err)
}

// size inspects the given entry and returns something which looks like a size. Returns a negative number, if unknown.
func size(entry Entry) int64 {
	if sizer, ok := entry.(interface{ Size() int64 }); ok {
		return sizer.Size()
	}
	if lengther, ok := entry.(interface{ Length() int64 }); ok {
		return lengther.Length()
	}
	return -1
}

// Copy performs a copy from src to dst. Dst is always removed and replaced with the contents of src.
// The copy options can be nil and can be used to get detailed information on the progress. The implementation
// tries to use RefLink if possible.
func Copy(src string, dst string, options *CopyOptions) error {

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

	if info.IsDir() {
		var objectsFound int64
		var bytesFound int64
		var objectsProcessed int64
		var bytesProcessed int64
		// collect info
		list := make([]*PathEntry, 0)
		err = Walk(src, func(path string, info Entry, err error) error {
			if err != nil {
				return options.onError(path, err)
			}
			list = append(list, &PathEntry{path, info})
			objectsFound++
			if !info.IsDir() {
				len := size(info)
				if len > 0 {
					bytesFound += len
				}

			}
			options.onScan(path, objectsFound, bytesFound)
			return nil
		})

		if err != nil {
			return err
		}

		//walk through, directory are first
		for _, entry := range list {
			dstPath := ConcatPaths(Path(dst), Path(entry.Path).TrimPrefix(Path(src)))
			if entry.Entry.IsDir() {
				err := MkDirs(dstPath.String())
				if err != nil {
					err = options.onError(dstPath.String(), err)
					if err != nil {
						return err
					}
				}
				objectsProcessed++
				options.onCopied(entry.Path, objectsProcessed, bytesProcessed)
			} else if !entry.Entry.IsDir() {
				reader, err := Read(entry.Path)
				if err != nil {
					return err
				}
				writer, err := Write(dstPath.String())
				if err != nil {
					silentClose(reader)
					return err
				}
				written, err := copyBuffer(entry.Path, dstPath.String(), size(entry.Entry), reader, writer, nil, options)
				silentClose(reader)
				silentClose(writer)
				if err != nil {
					err = options.onError(dstPath.String(), err)
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

func copyBuffer(srcPath string, dstPath string, totalSize int64, src io.Reader, dst io.Writer, buf []byte, options *CopyOptions) (written int64, err error) {
	if buf == nil {
		size := 32 * 1024
		buf = make([]byte, size)
	}
	for {
		if options.IsCancelled() {
			err = &DefaultError{Code: EINTR}
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

// Equals returns true if the values defined by Entry are equal.
// However it does not inspect or check other fields or values, especially not Sys()
func Equals(a Entry, b Entry) bool {
	if a == nil || b == nil {
		return false
	}
	return a.Name() == b.Name() && a.IsDir() == b.IsDir()
}

type entryDelegator struct {
	entry Entry
}

func (e entryDelegator) Name() string {
	return e.entry.Name()
}

func (e entryDelegator) Size() int64 {
	if sizer, ok := e.entry.(interface{ Size() int64 }); ok {
		return sizer.Size()
	}
	return -1
}

func (e entryDelegator) Mode() os.FileMode {
	if e.entry.IsDir() {
		return os.ModeDir
	}
	return 0 //regular
}

func (e entryDelegator) ModTime() time.Time {
	if timer, ok := e.entry.(interface{ ModTime() time.Time }); ok {
		return timer.ModTime()
	}
	return time.Unix(0, 0)
}

func (e entryDelegator) IsDir() bool {
	return e.entry.IsDir()
}

func (e entryDelegator) Sys() interface{} {
	return e.entry.Sys()
}
