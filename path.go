package vfs

import "strings"

// A Path must be unique in it's context and has the role of a composite key. It's segments are always separated using
// a slash, even if they denote paths from windows.
//
// Example
//
// Valid example paths
//
//  * /my/path/may/denote/a/file/or/folder
//  * c:/my/windows/folder
//  * mydomain.com/myresource
//  * mydomain.com:8080/myresource?size=720p#anchor
//  * c:/my/ntfs/file:alternate-data-stream
//
// Invalid example paths
//  * missing/slash
//  * /extra/slash/
//  * \using\backslashes
//  * /c///using/slashes without content
//  * ../../using/relative/paths
//  * https://mydomain.com:8080/myresource
//
// Design decisions
//
// There are the following opinionated decisions:
//  * In the context of a filesystem, this is equal to the full qualified name of a file entry.
//
//  * It is a string, because defacto all modern APIs are UTF-8 and web based. However there are also a lot of Unix or
//    Linux types which have different local encodings or just have an undefined byte sequence. Providers with such
//    requirements must support the path API through some kind of conversion and normalization, but they should also
//    provide an exact API using byte slices then.
//    One could also argue, that a string is a bad choice for Go, because of these corner case, potential invalid utf-8
//    sequences and suboptimal string allocations. But using just byte-slices by default would make a lot of things even
//    worse:
//       * You cannot simply compare byte slices in Go. You need to compare and acknowledge about a new standard.
//       * It can be expected that the developer using this library will certainly need a string representation which
//         will practically always cause additional allocations.
//       * Because a path is naturally always a string, you certainly want to use all the provided and standard string
//         handling infrastructures instead of reinventing your own.
//
//  * There are studies which claim that the average filename is between 11 and 15 characters long. Because we
//    want to optimize use cases like keeping 1 million file names in memory, using a fixed size 256 byte array would result
//    in a 17x overhead of memory usage: e.g. 17GiB instead of 1GiB of main memory. To save even more memory and lower
//    GC pressure, we do not use a slice of strings but just a pure string providing helper methods.
type Path string

// StartsWith tests whether the path begins with prefix.
func (p Path) StartsWith(prefix Path) bool {
	return strings.HasPrefix(string(p), string(prefix))
}

// EndsWith tests whether the path ends with prefix.
func (p Path) EndsWith(suffix Path) bool {
	return strings.HasSuffix(string(p), string(suffix))
}

// Names splits the path by / and returns all segments as a simple string array.
func (p Path) Names() []string {
	tmp := strings.Split(string(p), "/")
	cleaned := make([]string, len(tmp))
	idx := 0
	for _, str := range tmp {
		str = strings.TrimSpace(str)
		if len(str) > 0 {
			cleaned[idx] = str
			idx++
		}
	}
	return cleaned[0:idx]
}

// NameCount returns how many names are included in this path.
func (p Path) NameCount() int {
	return len(p.Names())
}

// NameAt returns the name at the given index.
func (p Path) NameAt(idx int) string {
	return p.Names()[idx]
}

// Name returns the last element in this path or the empty string if this path is empty.
func (p Path) Name() string {
	tmp := p.Names()
	if len(tmp) > 0 {
		return tmp[len(tmp)]
	}
	return ""
}

// Parent returns the parent path of this path.
func (p Path) Parent() Path {
	tmp := p.Names()
	if len(tmp) > 0 {
		return Path(strings.Join(tmp[:len(tmp)-1], "/"))
	}
	return ""
}

// String normalizes the slashes in Path
func (p Path) String() string {
	return "/" + strings.Join(p.Names(), "/")
}

// Child returns a new Path with name appended as a child
func (p Path) Child(name string) Path {
	if strings.HasPrefix(name, "/") {
		return Path(p.String() + name)
	}
	return Path(p.String() + "/" + name)
}

// TrimPrefix returns a path without the prefix
func (p Path) TrimPrefix(prefix Path) Path {
	tmp := "/" + strings.TrimPrefix(p.String(), prefix.String())
	return Path(tmp)
}

// ConcatPaths merges all paths together
func ConcatPaths(paths ...Path) Path {
	tmp := make([]string, 0)
	for _, path := range paths {
		for _, name := range path.Names() {
			tmp = append(tmp, name)
		}
	}
	return Path("/" + strings.Join(tmp, "/"))
}
