//This package provides the API and basic tools for virtual filesystem in go.
package vfs

// A name represents a part of a path.
// Design decisions
// * It is not a string, because strings have a lot of drawbacks, especially in Go. At first, a filesystem may provide
//   any kind of encoding, not only utf-8, which could also result in invalid utf-8 sequences. Secondly we cannot
//   optimize string allocations in Go (1.11), but we can recycle byte slices as we like
// * There are studies which claim that the average filename is between 11 and 15 characters long. Because we
//   want to optimize use cases like keeping 1 million file names in memory, using a fixed size 256 byte array would result
//   in a 17x overhead of memory usage: e.g. 17GiB instead of 1GiB of main memory.
type Name []byte

//TBD
type Path []Name

//TBD
type Transaction interface {
	Begin() (VFS, error)
	Commit(VFS) error
	Rollback(VFS) error
}

// The VFS interface is the core contract to provide a filesystem
//Design decisions
// * It is an Interface, because it cannot be expected to have a reasonable code reusage between implementations
// * It contains both read and write contracts, because a distinction between read-only and write-only and filesystem with both
//   capabilities are edge cases. Mostly there will be implementations which provides each combination due to their permission handling
// * Most implementations do not provide a transactional contract, however abstraction which do so, should only provide their VFS contract
//   through the Transaction interface.
type VFS interface {
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