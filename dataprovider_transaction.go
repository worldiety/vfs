package vfs

// A transaction has different isolation levels
type IsolationLevel int

// See https://en.wikipedia.org/wiki/Isolation_(database_systems)#Isolation_levels to learn more about isolation levels.
const (
	LevelDefault IsolationLevel = iota
	LevelReadUncommitted
	LevelReadCommitted
	LevelWriteCommitted
	LevelRepeatableRead
	LevelSnapshot
	LevelSerializable
	LevelLinearizable
)

type TxOptions struct {
	// Isolation is the transaction isolation level.
	Isolation IsolationLevel
	// If true, a transaction must deny all modification attempts.
	ReadOnly bool
}

// A TransactionableDataProvider supports also the usual style. If it implicitly creates a transaction per
// operation or in time slices or other criterias is implementation specific.
type TransactionableDataProvider interface {
	// Begins either a ReadOnly or ReadWrite transaction. ReadOnly may be ignored and used for optimizations only.
	// The returned Transaction must be closed by either Commiting or by Rollback.
	Begin(opts TxOptions) (Tx, error)
	DataProvider
}

type Tx interface {
	Commit() error
	Rollback() error
	// A simple close of the DataProvider without a commit will perform a Rollback.
	DataProvider
}
