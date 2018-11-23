package vfs

type TransactionMode uint32

const (
	ReadOnly TransactionMode = 1 << (32 - 1 - iota)
	ReadWrite
)

// A TransactionableDataProvider supports also the usual style. If it implicitly creates a transaction per
// operation or in time slices or other criterias is implementation specific.
type TransactionableDataProvider interface {
	// Begins either a ReadOnly or ReadWrite transaction. ReadOnly may be ignored and used for optimizations only.
	// The returned Transaction must be closed by either Commiting or by Rollback.
	Begin(mode TransactionMode) (Transaction, error)
	DataProvider
}

type Transaction interface {
	Commit() error
	Rollback() error
	// A simple close of the DataProvider without a commit will perform a Rollback.
	DataProvider
}
