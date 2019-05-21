package vfs

import (
	"context"
)

var _ FileSystem = (*ChRoot)(nil)

// A ChRoot is a filesystem which is basically a poor man's chroot which just adds a prefix to all endpoints and
// delegates all calls. A security note: the path is normalized before prefixing, so that path based attacks
// using .. are not possible.
type ChRoot struct {
	// The Prefix which is added before delegating
	Prefix Path
	// The Delegate to call with the prefixed path
	Delegate FileSystem
}

// Resolve normalizes the given Path and inserts the prefix.
// We normalize our path, before adding the prefix to avoid breaking out of our root
func (f *ChRoot) Resolve(path string) string {
	return f.Prefix.Add(Path(path).Normalize()).String()
}

func (f *ChRoot) Connect(ctx context.Context, path string, options interface{}) error {
	return f.Delegate.Connect(ctx, f.Resolve(path), options)
}

func (f *ChRoot) Disconnect(ctx context.Context, path string) error {
	return f.Delegate.Disconnect(ctx, f.Resolve(path))
}

func (f *ChRoot) FireEvent(ctx context.Context, path string, event interface{}) error {
	return f.Delegate.FireEvent(ctx, f.Resolve(path), event)
}

func (f *ChRoot) AddListener(ctx context.Context, path string, listener ResourceListener) (handle int, err error) {
	return f.Delegate.AddListener(ctx, f.Resolve(path), &chrootListener{f, listener})
}

func (f *ChRoot) RemoveListener(ctx context.Context, handle int) error {
	return f.Delegate.RemoveListener(ctx, handle)
}

func (f *ChRoot) Begin(ctx context.Context, path string, options interface{}) (context.Context, error) {
	return f.Delegate.Begin(ctx, f.Resolve(path), options)
}

func (f *ChRoot) Commit(ctx context.Context) error {
	return f.Delegate.Commit(ctx)
}

func (f *ChRoot) Rollback(ctx context.Context) error {
	return f.Delegate.Rollback(ctx)
}

func (f *ChRoot) Open(ctx context.Context, path string, flag int, options interface{}) (Blob, error) {
	return f.Delegate.Open(ctx, f.Resolve(path), flag, options)
}

func (f *ChRoot) Delete(ctx context.Context, path string) error {
	return f.Delegate.Delete(ctx, f.Resolve(path))
}

func (f *ChRoot) ReadAttrs(ctx context.Context, path string, args interface{}) (Entry, error) {
	return f.Delegate.ReadAttrs(ctx, f.Resolve(path), args)
}

func (f *ChRoot) ReadForks(ctx context.Context, path string) ([]string, error) {
	return f.Delegate.ReadForks(ctx, f.Resolve(path))
}

func (f *ChRoot) WriteAttrs(ctx context.Context, path string, src interface{}) (Entry, error) {
	return f.Delegate.WriteAttrs(ctx, f.Resolve(path), src)
}

func (f *ChRoot) ReadBucket(ctx context.Context, path string, options interface{}) (ResultSet, error) {
	return f.Delegate.ReadBucket(ctx, f.Resolve(path), options)
}

func (f *ChRoot) Invoke(ctx context.Context, endpoint string, args ...interface{}) (interface{}, error) {
	return f.Delegate.Invoke(ctx, endpoint, args)
}

func (f *ChRoot) MkBucket(ctx context.Context, path string, options interface{}) error {
	return f.Delegate.MkBucket(ctx, f.Resolve(path), options)
}

func (f *ChRoot) Rename(ctx context.Context, oldPath string, newPath string) error {
	return f.Delegate.Rename(ctx, f.Resolve(oldPath), f.Resolve(newPath))
}

func (f *ChRoot) SymLink(ctx context.Context, oldPath string, newPath string) error {
	return f.Delegate.SymLink(ctx, f.Resolve(oldPath), f.Resolve(newPath))
}

func (f *ChRoot) HardLink(ctx context.Context, oldPath string, newPath string) error {
	return f.Delegate.HardLink(ctx, f.Resolve(oldPath), f.Resolve(newPath))
}

func (f *ChRoot) RefLink(ctx context.Context, oldPath string, newPath string) error {
	return f.Delegate.RefLink(ctx, f.Resolve(oldPath), f.Resolve(newPath))
}

func (f *ChRoot) String() string {
	return "chroot(" + f.Delegate.String() + ")"
}

func (f *ChRoot) Close() error {
	return f.Delegate.Close()
}

type chrootListener struct {
	parent   *ChRoot
	delegate ResourceListener
}

func (l *chrootListener) OnEvent(path string, event interface{}) error {
	p := Path(path)
	path = p.TrimPrefix(l.parent.Prefix).String()
	if l.delegate != nil {
		return l.delegate.OnEvent(path, event)
	}
	return nil
}
