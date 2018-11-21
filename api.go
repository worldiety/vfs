//This package provides the API and basic tools for data providers, also known as virtual filesystem, in go.
package vfs

// A Key must be unique in it's context.
//
// Design decisions
//
// There are the following opinionated decisions:
//
//  * In the context of a filesystem, this is equal to the Name of a file entry.
//
//  * It is not a string, because strings have a lot of drawbacks, especially in Go. At first, a filesystem may provide
//    any kind of encoding, not only utf-8, which could also result in invalid utf-8 sequences. Secondly we cannot
//    optimize string allocations in Go (1.11), but we can recycle byte slices as we like.
//
//  * There are studies which claim that the average filename is between 11 and 15 characters long. Because we
//    want to optimize use cases like keeping 1 million file names in memory, using a fixed size 256 byte array would result
//    in a 17x overhead of memory usage: e.g. 17GiB instead of 1GiB of main memory.
type Key []byte

// A CompositeKey is unique
//
// Design decisions
//
// There are the following opinionated decisions:
//
type CompositeKey []Key

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
	ReadStat(CompositeKey,interface{})error
}

type StatUnix struct{

}

type StatWindows struct{

}

type StatX struct{

}