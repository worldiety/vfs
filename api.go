//This package provides the API and basic tools for data providers, also known as virtual filesystem, in go.
package vfs



// A Path must be unique in it's context and has the role of a composite key. It's segments are always separated using
// a slash, even if they denote paths from windows.
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

//TBD
type Transaction interface {
	Begin() (DataProvider, error)
	Commit(DataProvider) error
	Rollback(DataProvider) error
}

// The DataProvider interface is the core contract to provide access to hierarchical structures using a compound
// key logic. This is an abstract of way of the design thinking behind a filesystem.
//
// Design decisions
//
// There are the following opinionated decisions:
//
//  * It is an Interface, because it cannot be expected to have a reasonable code reusage between implementations but we
//    need a common behavior.
//
//  * It contains both read and write contracts, because a distinction between read-only and write-only and filesystems
//    with both capabilities are edge cases. Mostly there will be implementations which provides each combination due to
//    their permission handling.
//
//  * Most implementations do not provide a transactional contract, however abstraction which do so, should only provide
//    their VFS contract through the Transaction interface.
//
type DataProvider interface {
	//A generic stat call to read information about a path, potentially without allocations.
	//Must be an instance of a Stat* struct, but each implementation may also provide custom things for any use case.
	ReadStat(Path,interface{})error
}

type StatUnix struct{

}

type StatWindows struct{

}

type StatX struct{

}