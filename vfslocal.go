package vfs

import (
	"context"
	"io/ioutil"
	"os"
)

var LocalFileSystem FileSystem

func init() {
	LocalFileSystem = createLocalVFS()
}

func createLocalVFS() FileSystem {
	builder := &Builder{}

	vfs := builder.Details("local", 1, 0, 0).
		// bucket listing
		MatchBucket("/").
		OnList(func(path Path) ([]AbsEntry, error) {
			files, err := ioutil.ReadDir(path.String())
			if err != nil {
				return nil, err
			}
			res := make([]AbsEntry, len(files))
			for i, f := range files {
				res[i].Id = f.Name()
				res[i].Length = f.Size()
				res[i].IsBucket = f.IsDir()
				res[i].Data = f
			}
			return res, nil
		}).
		Add().
		// generic (fallback) delete
		Delete(func(i context.Context, path Path) error {
			return os.RemoveAll(path.String())
		}).
		// generic (fallback) read attributes
		ReadEntryAttrs(func(ctx context.Context, path Path, dst *AbsEntry) error {
			stat, err := os.Stat(path.String())
			if err != nil {
				return err
			}
			dst.Data = stat
			dst.Length = stat.Size()
			dst.IsBucket = stat.IsDir()
			dst.Id = stat.Name()
			return nil
		}).
		// generic mkdir
		MkBucket(func(ctx context.Context, path Path, options interface{}) error {
			perm := os.ModePerm
			if p, ok := options.(os.FileMode); ok {
				perm = p
			}
			return os.MkdirAll(path.String(), perm)
		}).
		// blob matching
		MatchBlob("/").
		OnOpen(func(_ context.Context, path Path, flag int, perm interface{}) (blob Blob, e error) {
			mode := os.ModePerm
			if m, ok := perm.(os.FileMode); ok {
				mode = m
			}
			if flag == os.O_RDONLY {
				return os.OpenFile(path.String(), flag, 0)
			}
			file, err := os.OpenFile(path.String(), flag, mode)
			if _, ok := err.(*os.PathError); ok {
				//try to recreate parent folder
				err2 := os.MkdirAll(path.Parent().String(), mode)
				if err2 != nil {
					//suppress err2 intentionally and return the original failure
					return nil, err
				}
				// mkdir is fine, retry again
				file, err = os.OpenFile(path.String(), flag, mode)
				if err != nil {
					return nil, err
				}
			}
			return file, nil
		}).Add().
		// linkings
		Symlink(func(ctx context.Context, oldPath Path, newPath Path) error {
			return os.Symlink(oldPath.String(), newPath.String())
		}).
		Hardlink(func(ctx context.Context, oldPath Path, newPath Path) error {
			return os.Link(oldPath.String(), newPath.String())
		}).
		// finally create the vfs
		Create()

	return vfs
}
