package vfs

import (
	"context"
	"os"
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

func (p *MountableFileSystem) Connect(ctx context.Context, path string, options interface{}) error {
	_, providerPath, dp, err := p.Resolve(path)
	if err != nil {
		return err
	}
	return dp.Connect(ctx, providerPath, options)
}

func (p *MountableFileSystem) Disconnect(ctx context.Context, path string) error {
	_, providerPath, dp, err := p.Resolve(path)
	if err != nil {
		return err
	}
	return dp.Disconnect(ctx, providerPath)
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
	panic("implement me")
}

func (p *MountableFileSystem) Delete(ctx context.Context, path string) error {
	panic("implement me")
}

func (p *MountableFileSystem) ReadAttrs(ctx context.Context, path string, args interface{}) (Entry, error) {
	panic("implement me")
}

func (p *MountableFileSystem) ReadForks(ctx context.Context, path string) ([]string, error) {
	panic("implement me")
}

func (p *MountableFileSystem) WriteAttrs(ctx context.Context, path string, src interface{}) error {
	panic("implement me")
}

func (p *MountableFileSystem) ReadBucket(ctx context.Context, path string, options interface{}) (ResultSet, error) {
	panic("implement me")
}

func (p *MountableFileSystem) Invoke(ctx context.Context, endpoint string, args ...interface{}) (interface{}, error) {
	panic("implement me")
}

func (p *MountableFileSystem) MkBucket(ctx context.Context, path string, options interface{}) error {
	panic("implement me")
}

func (p *MountableFileSystem) Rename(ctx context.Context, oldPath string, newPath string) error {
	panic("implement me")
}

func (p *MountableFileSystem) SymLink(ctx context.Context, oldPath string, newPath string) error {
	panic("implement me")
}

func (p *MountableFileSystem) HardLink(ctx context.Context, oldPath string, newPath string) error {
	panic("implement me")
}

func (p *MountableFileSystem) Copy(ctx context.Context, oldPath string, newPath string) error {
	panic("implement me")
}

func (p *MountableFileSystem) String() string {
	panic("implement me")
}

// Link details: see FileSystem:Link
func (p *MountableFileSystem) Link(oldPath string, newPath string, mode LinkMode, flags int32) error {
	mp0, _, dp0 := p.Resolve(Path(oldPath))
	mp1, _, _ := p.Resolve(Path(newPath))
	if mp0 != mp1 {
		return &UnsupportedOperationError{Message: "cannot Link across mount points: " + mp0.String() + " -> " + mp1.String()}
	}

	if dp0 != nil {
		unwrapedOld := Path(oldPath).TrimPrefix(mp0)
		unwrappedNew := Path(newPath).TrimPrefix(mp1)
		return dp0.Link(unwrapedOld.String(), unwrappedNew.String(), mode, flags)
	}
	return &MountPointNotFoundError{}
}

// Open details: see FileSystem#Open
func (p *MountableFileSystem) Open(ctx context.Context, flag int, perm os.FileMode, path string) (Resource, error) {
	_, providerPath, dp := p.Resolve(Path(path))
	if dp != nil {
		return dp.Open(
	}
	return nil, &MountPointNotFoundError{}
}

// Rename details: see FileSystem#Rename
func (p *MountableFileSystem) Rename(oldPath string, newPath string) error {
	mp0, _, dp0 := p.Resolve(Path(oldPath))
	mp1, _, _ := p.Resolve(Path(newPath))
	if mp0 != mp1 {
		return &UnsupportedOperationError{Message: "cannot rename across mount points: " + mp0.String() + " -> " + mp1.String()}
	}

	if dp0 != nil {
		unwrapedOld := Path(oldPath).TrimPrefix(mp0)
		unwrappedNew := Path(newPath).TrimPrefix(mp1)
		return dp0.Rename(unwrapedOld.String(), unwrappedNew.String())
	}
	return &MountPointNotFoundError{}
}

// MkDirs details: see FileSystem#MkDirs
func (p *MountableFileSystem) MkDirs(path string) error {
	_, providerPath, dp := p.Resolve(Path(path))
	if dp != nil {
		return dp.MkDirs(Path(providerPath).String())
	}
	return &MountPointNotFoundError{}
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

// ReadAttrs details: see FileSystem#ReadAttrs
func (p *MountableFileSystem) ReadAttrs(path string, dest interface{}) error {
	_, providerPath, dp := p.Resolve(Path(path))
	if dp != nil {
		return dp.ReadAttrs(providerPath.String(), dest)
	}
	return &MountPointNotFoundError{}
}

// WriteAttrs details: see FileSystem#WriteAttrs
func (p *MountableFileSystem) WriteAttrs(path string, src interface{}) error {
	_, providerPath, dp := p.Resolve(Path(path))
	if dp != nil {
		return dp.WriteAttrs(providerPath.String(), src)
	}
	return &MountPointNotFoundError{}
}

// ReadDir either dispatches as expected or the virtual directories. See also FileSystem#ReadDir
func (p *MountableFileSystem) ReadDir(path string, options interface{}) (DirEntList, error) {
	_, providerPath, dp := p.Resolve(Path(path))
	if dp != nil {
		return dp.ReadDir(providerPath.String(), options)
	}
	//just try to walk
	parent := p.getRoot()
	names := Path(path).Names()
	if len(names) == 0 {
		return asDirEntList(p.root), nil
	}
	var child *namedEntry
	for _, name := range names {
		child = parent.ChildByName(name)
		if vdir, ok := child.data.(*virtualDir); ok {
			parent = vdir
		} else {
			return nil, &ResourceNotFoundError{Path: Path(path)}
		}
	}
	if vdir, ok := child.data.(*virtualDir); ok {
		return asDirEntList(vdir), nil
	}
	panic("implementation failure")

}

// Delete dispatches as expected or removes a mount point. See FileSystem#Delete
func (p *MountableFileSystem) Delete(path string) error {
	_, providerPath, dp := p.Resolve(Path(path))
	if dp != nil {
		//do not delegate empty path delete
		if providerPath.NameCount() > 0 {
			return dp.Delete(providerPath.String())
		}
	}

	//just try to walk and clean any mount points
	parent := p.getRoot()
	names := Path(path).Names()
	for _, name := range names[0 : len(names)-1] {
		child := parent.ChildByName(name)
		if child == nil {
			return &ResourceNotFoundError{Path: Path(path)}
		}
		if vdir, ok := child.data.(*virtualDir); ok {
			parent = vdir
		}
	}
	tmp := parent.RemoveChild(names[len(names)-1])
	if tmp == nil {
		return &MountPointNotFoundError{}
	}
	return nil
}

func asDirEntList(parent *virtualDir) DirEntList {
	return NewDirEntList(int64(len(parent.children)), func(idx int64, dst ResourceInfo) error {
		child := parent.children[int(idx)]
		dst.SetSize(0)
		dst.SetMode(os.ModeDir)
		dst.SetName(child.name)
		return nil
	})
}

type mountpointListener struct {
	prefix   string
	delegate ResourceListener
}

func (l *mountpointListener) OnEvent(path string, event interface{}) error {
	if l.delegate != nil {
		p := Path(path)
		path = Path(l.prefix).Add(Path(path)).String()
		return l.delegate.OnEvent(path, event)
	}
	return nil
}

type hiddenPath string
