package vfs

import (
	"io"
	"os"
)

var _ DataProvider = (*MountableDataProvider)(nil)

type virtualDir struct {
	children []*namedEntry
}

type namedEntry struct {
	name string
	// either a *virtualEntry or a DataProvider
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

// A MountableDataProvider contains only other DataProviders mounted under a path. Mounting cross paths is not
// supported.
//
// Example
//
// If you have /my/dir/provider0 and mount /my/dir/provider0/some/dir/provider1 the existing provider0 will be removed.
type MountableDataProvider struct {
	root *virtualDir
}

// Rename details: see DataProvider#Rename
func (p *MountableDataProvider) Rename(oldPath Path, newPath Path) error {
	mp0, providerPath0, dp0 := p.Resolve(oldPath)
	mp1, _, _ := p.Resolve(newPath)
	if mp0 != mp1 {
		return &UnsupportedOperationError{Message: "cannot rename across mount points: " + mp0.String() + " -> " + mp1.String()}
	}

	if dp0 != nil {
		return dp0.MkDirs(providerPath0)
	}
	return &MountPointNotFoundError{}
}

// MkDirs details: see DataProvider#MkDirs
func (p *MountableDataProvider) MkDirs(path Path) error {
	_, providerPath, dp := p.Resolve(path)
	if dp != nil {
		return dp.MkDirs(providerPath)
	}
	return &MountPointNotFoundError{}
}

// Close does nothing.
func (p *MountableDataProvider) Close() error {
	return nil
}

func (p *MountableDataProvider) getRoot() *virtualDir {
	if p.root == nil {
		p.root = &virtualDir{}
	}
	return p.root
}

// Mount includes the given provider into the leaf of the path. Important: you cannot mount one provider into another.
func (p *MountableDataProvider) Mount(mountPoint Path, provider DataProvider) {
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
func (p *MountableDataProvider) Resolve(path Path) (mountPoint Path, providerPath Path, provider DataProvider) {
	names := path.Names()
	parent := p.getRoot()
	var child *namedEntry
	for _, name := range names {
		child = parent.ChildByName(name)
		if child == nil {
			return "", "", nil
		}

		mountPoint = mountPoint.Child(name)
		if dp, ok := child.data.(DataProvider); ok {
			//found the mount point
			return mountPoint, path.TrimPrefix(mountPoint), dp
		}
	}
	return "", "", nil
}

// ReadAttrs details: see DataProvider#ReadAttrs
func (p *MountableDataProvider) ReadAttrs(path Path, dest interface{}) error {
	_, providerPath, dp := p.Resolve(path)
	if dp != nil {
		return dp.ReadAttrs(providerPath, dest)
	}
	return &MountPointNotFoundError{}
}

// WriteAttrs details: see DataProvider#WriteAttrs
func (p *MountableDataProvider) WriteAttrs(path Path, src interface{}) error {
	_, providerPath, dp := p.Resolve(path)
	if dp != nil {
		return dp.WriteAttrs(providerPath, src)
	}
	return &MountPointNotFoundError{}
}

// ReadDir either dispatches as expected or the virtual directories. See also DataProvider#ReadDir
func (p *MountableDataProvider) ReadDir(path Path) (DirEntList, error) {
	_, providerPath, dp := p.Resolve(path)
	if dp != nil {
		return dp.ReadDir(providerPath)
	}
	//just try to walk
	parent := p.getRoot()
	names := path.Names()
	var child *namedEntry
	for _, name := range names {
		child = parent.ChildByName(name)
		if vdir, ok := child.data.(*virtualDir); ok {
			parent = vdir
		} else {
			return nil, &ResourceNotFoundError{Path: path}
		}
	}
	if vdir, ok := child.data.(*virtualDir); ok {
		return &virtualDirEntList{vdir}, nil
	}
	panic("implementation failure")

}

func (p *MountableDataProvider) Read(path Path) (io.ReadCloser, error) {
	_, providerPath, dp := p.Resolve(path)
	if dp != nil {
		return dp.Read(providerPath)
	}
	return nil, &MountPointNotFoundError{}
}

func (p *MountableDataProvider) Write(path Path) (io.WriteCloser, error) {
	_, providerPath, dp := p.Resolve(path)
	if dp != nil {
		return dp.Write(providerPath)
	}
	return nil, &MountPointNotFoundError{}
}

// Delete dispatches as expected or removes a mount point. See DataProvider#Delete
func (p *MountableDataProvider) Delete(path Path) error {
	_, providerPath, dp := p.Resolve(path)
	if dp != nil {
		return dp.Delete(providerPath)
	}
	//just try to walk and clean any mount points
	parent := p.getRoot()
	names := path.Names()
	for _, name := range names[0 : len(names)-1] {
		child := parent.ChildByName(name)
		if child == nil {
			return &ResourceNotFoundError{Path: path}
		}
	}
	parent.RemoveChild(names[len(names)-1])
	return &MountPointNotFoundError{}
}

type virtualDirEntList struct {
	parent *virtualDir
}

func (v *virtualDirEntList) ForEach(each func(scanner Scanner) error) error {
	tmp := &entScanner{}
	for _, ent := range v.parent.children {
		tmp.entry = ent
		err := each(tmp)
		if err != nil {
			return err
		}
	}
	return nil
}

func (v *virtualDirEntList) Size() int64 {
	return int64(len(v.parent.children))
}

func (v *virtualDirEntList) Close() error {
	return nil
}

type entScanner struct {
	entry *namedEntry
}

func (s *entScanner) Scan(dest interface{}) error {
	if info, ok := dest.(*ResourceInfo); ok {
		info.Size = 0
		info.Mode = os.ModeDir
		info.Name = s.entry.name
		return nil
	}
	return &UnsupportedAttributesError{Data: dest}

}
