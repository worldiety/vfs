package vfs

import "io"

var _ DataProvider = (*RootProvider)(nil)

// A RootProvider is a kind of a mountable filesystem. Use it to prefix other DataProviders.
type RootProvider struct {
	mountPoints map[Path]DataProvider
}

// Returns the MountPoints
func (p *RootProvider) getMountPoints() map[Path]DataProvider {
	if p.mountPoints == nil {
		p.mountPoints = make(map[Path]DataProvider)
	}
	return p.mountPoints
}

func (p *RootProvider) Register(mountPoint Path, provider DataProvider) {
	p.getMountPoints()[mountPoint] = provider
}

// Returns nil or the registered DataProvider
func (p *RootProvider) DataProvider(mountPoint Path) DataProvider {
	return p.getMountPoints()[mountPoint]
}

// Applies the query on the mounted data provider. If no such provider is found, a MountPointNotFoundError is returned.
func (p *RootProvider) Query(query *Query) (ResultSet, error) {
	if query.IsFilterEmpty() {
		//we need to ask all providers
		resultSets := make([]ResultSet, 0)
		for prefix, provider := range p.mountPoints {
			strippedQuery := removePrefix(query, prefix)
			res, err := provider.Query(strippedQuery)
			if err != nil {
				return nil, err
			}
			resultSets = append(resultSets, res)
		}
		return &joinedResultSet{resultSets, 0}, nil
	} else {
		for prefix, provider := range p.mountPoints {
			if query.AnyMatchStartsWith(prefix) {
				//found the mount point
				strippedQuery := removePrefix(query, prefix)
				return provider.Query(strippedQuery)
			}
		}
	}
	return nil, &MountPointNotFoundError{}
}

func (p *RootProvider) Read(path Path) (io.ReadCloser, error) {
	for prefix, provider := range p.mountPoints {
		if path.StartsWith(prefix) {
			//found the mount point
			return provider.Read(path.TrimPrefix(prefix))
		}
	}
	return nil, &MountPointNotFoundError{}
}

func (p *RootProvider) Write(path Path) (io.WriteCloser, error) {
	for prefix, provider := range p.mountPoints {
		if path.StartsWith(prefix) {
			//found the mount point
			return provider.Write(path.TrimPrefix(prefix))
		}
	}
	return nil, &MountPointNotFoundError{}
}

func (p *RootProvider) Delete(path Path) error {
	for prefix, provider := range p.mountPoints {
		if path.StartsWith(prefix) {
			//found the mount point
			return provider.Delete(path.TrimPrefix(prefix))
		}
	}
	return &MountPointNotFoundError{}
}

func (p *RootProvider) SetAttributes(attribs ...*Attributes) error {
	//TODO this implementation does not propagate batch updates efficiently
	found := false
	for _, attrib := range attribs {
		for prefix, provider := range p.mountPoints {
			if attrib.Path.StartsWith(prefix) {
				//found the mount point
				found = true
				err := provider.SetAttributes(&Attributes{attrib.Path.TrimPrefix(prefix), attrib.Data})
				if err != nil {
					return err
				}
			}
		}
	}
	if !found {
		return &MountPointNotFoundError{}
	}
	return nil
}

//rewrites the query by substracting the
func removePrefix(query *Query, prefix Path) *Query {
	stripped := &Query{query.Fields, make([]Path, len(query.MatchParents)), make([]Path, len(query.MatchPaths))}
	for idx, path := range query.MatchParents {
		stripped.MatchParents[idx] = path.TrimPrefix(prefix)
	}
	for idx, path := range query.MatchPaths {
		stripped.MatchPaths[idx] = path.TrimPrefix(prefix)
	}
	return stripped
}

// A joined result set to aggregate multiple
type joinedResultSet struct {
	results   []ResultSet
	activeIdx int
}

func (r *joinedResultSet) Next() bool {
	if r.activeIdx >= len(r.results) {
		return false
	}
	currentHasNext := r.results[r.activeIdx].Next()
	if !currentHasNext {
		r.activeIdx++
	}
	return r.Next()
}

func (r *joinedResultSet) Size() int64 {
	sum := int64(0)
	for _, rs := range r.results {
		sum += rs.Size()
	}
	return sum
}

func (r *joinedResultSet) Scan(dest interface{}) error {
	return r.results[r.activeIdx].Scan(dest)
}

func (r *joinedResultSet) Close() error {
	var firstErr error = nil
	for _, rs := range r.results {
		err := rs.Close()
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
