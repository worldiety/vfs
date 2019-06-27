package vfs

import (
	"context"
	"sync"
)

var _ FileSystem = (*MountableFileSystem)(nil)

type virtualDir struct {
	children []*namedEntry
}

type namedEntry struct {
	name string
	// either a *virtualEntry or a FileSystem
	data interface{}
}

// Returns the namedEntry or nil
func (d *virtualDir) ChildByName(name string) *namedEntry {
	for _, child := range d.children {
		if child.name == name {
			return child
		}
	}
	return nil
}

// Removes and returns the child, if any
func (d *virtualDir) RemoveChild(name string) *namedEntry {
	index := -1
	for idx, child := range d.children {
		if child.name == name {
			index = idx
			break
		}
	}
	if index == -1 {
		return nil
	}
	c := d.children[index]
	d.children = append(d.children[:index], d.children[index+1:]...)
	return c
}

type wrappedHandle struct {
	handle int
	fs     FileSystem
}

// A MountableFileSystem contains only other DataProviders mounted under a path. Mounting cross paths is not
// supported.
//
// Example
//
// If you have /my/dir/provider0 and mount /my/dir/provider0/some/dir/provider1 the existing provider0 will be removed.
type MountableFileSystem struct {
	root       *virtualDir
	lastHandle int
	handles    map[int]wrappedHandle
	lock       sync.Mutex
}

func (p *MountableFileSystem) wrapHandle(fs FileSystem, handle int) int {
	if p.handles == nil {
		p.handles = make(map[int]wrappedHandle)
	}
	p.lastHandle++
	p.handles[p.lastHandle] = wrappedHandle{handle, fs}
	return p.lastHandle
}

func (p *MountableFileSystem) unwrapHandle(handle int) wrappedHandle {
	if p.handles == nil {
		return wrappedHandle{}
	}
	return p.handles[handle]
}

func (p *MountableFileSystem) Connect(ctx context.Context, path string, options interface{}) (interface{}, error) {
	_, providerPath, dp, err := p.Resolve(path)
	if err != nil {
		return nil, err
	}
	return dp.Connect(ctx, providerPath, options)
}

// Disconnect tries to dispatch the call to a mounted vfs. In any case, the vfs (leaf) is removed from the tree, even
// if disconnect has returned an error. If you need a different behavior, you can use #Resolve() to grab the
// actual vfs instance.
func (p *MountableFileSystem) Disconnect(ctx context.Context, path string) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	_, providerPath, dp, err := p.Resolve(path)
	if err != nil {
		return err
	}
	err = dp.Disconnect(ctx, providerPath)
	names := Path(path).Names()
	root := p.root
	for i, name := range names {
		if i == len(names)-1 {
			root.RemoveChild(name) //remove the leaf which is the mounted vfs
		} else {
			child := root.ChildByName(name)
			root = child.data.(*virtualDir) // we know that because of resolve and early exit
		}
	}
	if err != nil {
		return err
	}
	return nil
}

func (p *MountableFileSystem) FireEvent(ctx context.Context, path string, event interface{}) error {
	_, providerPath, dp, err := p.Resolve(path)
	if err != nil {
		return err
	}
	return dp.FireEvent(ctx, providerPath, event)
}

func (p *MountableFileSystem) AddListener(ctx context.Context, path string, listener ResourceListener) (handle int, err error) {
	prefix, providerPath, dp, err := p.Resolve(path)
	if err != nil {
		return -1, err
	}
	hnd, err := dp.AddListener(ctx, providerPath, &mountpointListener{prefix, listener})
	if err != nil {
		return hnd, err
	}
	return p.wrapHandle(dp, hnd), nil
}

func (p *MountableFileSystem) RemoveListener(ctx context.Context, handle int) error {
	unwrapped := p.unwrapHandle(handle)
	if unwrapped.fs != nil {
		return unwrapped.fs.RemoveListener(ctx, unwrapped.handle)
	}
	return nil
}

func (p *MountableFileSystem) Begin(ctx context.Context, path string, options interface{}) (context.Context, error) {
	_, providerPath, dp, err := p.Resolve(path)
	if err != nil {
		return nil, err
	}
	txCtx, err := dp.Begin(ctx, providerPath, options)
	if err != nil {
		return txCtx, err
	}
	txCtx = context.WithValue(txCtx, hiddenPath("path"), path)
	return txCtx, nil
}

func (p *MountableFileSystem) Commit(ctx context.Context) error {
	if path, ok := ctx.Value(hiddenPath("path")).(string); ok {
		_, _, dp, err := p.Resolve(path)
		if err != nil {
			return err
		}
		return dp.Commit(ctx)
	}
	return &DefaultError{Code: ENOMP, Message: "wrong context"}
}

func (p *MountableFileSystem) Rollback(ctx context.Context) error {
	if path, ok := ctx.Value(hiddenPath("path")).(string); ok {
		_, _, dp, err := p.Resolve(path)
		if err != nil {
			return err
		}
		return dp.Rollback(ctx)
	}
	return &DefaultError{Code: ENOMP, Message: "wrong context"}
}

func (p *MountableFileSystem) Open(ctx context.Context, path string, flag int, options interface{}) (Blob, error) {
	_, providerPath, dp, err := p.Resolve(path)
	if err != nil {
		return nil, err
	}
	return dp.Open(ctx, providerPath, flag, options)
}

func (p *MountableFileSystem) Delete(ctx context.Context, path string) error {
	_, providerPath, dp, err := p.Resolve(path)
	if err != nil {
		return err
	}
	return dp.Delete(ctx, providerPath)
}

func (p *MountableFileSystem) ReadAttrs(ctx context.Context, path string, args interface{}) (Entry, error) {
	_, providerPath, dp, err := p.Resolve(path)
	if err != nil {
		return nil, err
	}
	return dp.ReadAttrs(ctx, providerPath, args)
}

func (p *MountableFileSystem) ReadForks(ctx context.Context, path string) ([]string, error) {
	_, providerPath, dp, err := p.Resolve(path)
	if err != nil {
		return nil, err
	}
	return dp.ReadForks(ctx, providerPath)
}

func (p *MountableFileSystem) WriteAttrs(ctx context.Context, path string, src interface{}) (Entry, error) {
	_, providerPath, dp, err := p.Resolve(path)
	if err != nil {
		return nil, err
	}
	return dp.WriteAttrs(ctx, providerPath, src)

}

func (p *MountableFileSystem) ReadBucket(ctx context.Context, path string, options interface{}) (ResultSet, error) {
	_, providerPath, dp, err := p.Resolve(path)
	if err != nil {
		return nil, err
	}
	return dp.ReadBucket(ctx, providerPath, options)
}

// Invoke also relies on the prefixed endpoint
func (p *MountableFileSystem) Invoke(ctx context.Context, endpoint string, args ...interface{}) (interface{}, error) {
	_, providerPath, dp, err := p.Resolve(endpoint)
	if err != nil {
		return nil, err
	}
	return dp.Invoke(ctx, providerPath, args)
}

func (p *MountableFileSystem) MkBucket(ctx context.Context, path string, options interface{}) error {
	_, providerPath, dp, err := p.Resolve(path)
	if err != nil {
		return err
	}
	return dp.MkBucket(ctx, providerPath, options)
}

func (p *MountableFileSystem) resolveOldNewPath(oldPath string, newPath string) (dp FileSystem, oldP string, newP string, err error) {
	mp0, _, dp0, err0 := p.Resolve(oldPath)
	mp1, _, _, err1 := p.Resolve(newPath)

	if err0 != nil {
		return nil, "", "", err0
	}

	if err1 != nil {
		return nil, "", "", err1
	}

	if mp0 != mp1 {
		return nil, "", "", &DefaultError{Message: "cannot operate across mount points: " + mp0 + " -> " + mp1, Code: EINVAL}
	}

	unwrapedOld := Path(oldPath).TrimPrefix(Path(mp0))
	unwrappedNew := Path(newPath).TrimPrefix(Path(mp1))

	return dp0, unwrapedOld.String(), unwrappedNew.String(), nil
}

func (p *MountableFileSystem) Rename(ctx context.Context, oldPath string, newPath string) error {
	dp, oldP, newP, err := p.resolveOldNewPath(oldPath, newPath)
	if err != nil {
		return err
	}
	return dp.Rename(ctx, oldP, newP)
}

func (p *MountableFileSystem) SymLink(ctx context.Context, oldPath string, newPath string) error {
	dp, oldP, newP, err := p.resolveOldNewPath(oldPath, newPath)
	if err != nil {
		return err
	}
	return dp.SymLink(ctx, oldP, newP)
}

func (p *MountableFileSystem) HardLink(ctx context.Context, oldPath string, newPath string) error {
	dp, oldP, newP, err := p.resolveOldNewPath(oldPath, newPath)
	if err != nil {
		return err
	}
	return dp.HardLink(ctx, oldP, newP)
}

// RefLink is like RefLink
func (p *MountableFileSystem) RefLink(ctx context.Context, oldPath string, newPath string) error {
	dp, oldP, newP, err := p.resolveOldNewPath(oldPath, newPath)
	if err != nil {
		return err
	}
	return dp.RefLink(ctx, oldP, newP)
}

func (p *MountableFileSystem) String() string {
	return "MountableFileSystem"
}

// Close does nothing.
func (p *MountableFileSystem) Close() error {
	return nil
}

func (p *MountableFileSystem) getRoot() *virtualDir {
	if p.root == nil {
		p.root = &virtualDir{}
	}
	return p.root
}

// Mount includes the given provider into the leaf of the path. Important: you cannot mount one provider into another.
func (p *MountableFileSystem) Mount(mountPoint Path, provider FileSystem) {
	parent := p.getRoot()
	names := mountPoint.Names()
	// ensure the path
	for _, name := range names[0 : len(names)-1] {
		child := parent.ChildByName(name)
		if child == nil {
			child = &namedEntry{name: name, data: &virtualDir{}}
			parent.children = append(parent.children, child)
		}
		if vdir, ok := child.data.(*virtualDir); ok {
			parent = vdir
		} else {
			//mounting on a leaf or similar
			vdir = &virtualDir{}
			child.data = vdir
			parent = vdir
		}
	}

	//now attach the child
	name := names[len(names)-1]
	parent.RemoveChild(name)
	parent.children = append(parent.children, &namedEntry{name, provider})
}

// Mounted returns the mounted filesystem or nil if the path cannot be resolved to a mountpoint.
func (p *MountableFileSystem) Mounted(path string) FileSystem {
	_, _, vfs, _ := p.Resolve(path)
	return vfs
}

// Resolve searches the virtual structure and returns a provider and the according data or nil and empty paths
func (p *MountableFileSystem) Resolve(path string) (mountPoint string, providerPath string, provider FileSystem, err error) {
	names := Path(path).Names()
	parent := p.getRoot()
	var child *namedEntry
	for _, name := range names {
		child = parent.ChildByName(name)
		if child == nil {
			return "", "", nil, &DefaultError{Code: ENOMP, DetailsPayload: path, Message: "mount point not found"}
		}

		mountPoint = Path(mountPoint).Child(name).String()
		if dp, ok := child.data.(FileSystem); ok {
			//found the mount point
			return mountPoint, Path(path).TrimPrefix(Path(mountPoint)).String(), dp, nil
		}
		if vdir, ok := child.data.(*virtualDir); ok {
			parent = vdir
		} else {
			panic("implementation assertion")
		}
	}
	return "", "", nil, &DefaultError{Code: ENOMP, DetailsPayload: path, Message: "mount point not found"}
}

type mountpointListener struct {
	prefix   string
	delegate ResourceListener
}

func (l *mountpointListener) OnEvent(path string, event interface{}) error {
	if l.delegate != nil {
		p := Path(path)
		path = Path(l.prefix).Add(p).String()
		return l.delegate.OnEvent(path, event)
	}
	return nil
}

type hiddenPath string
