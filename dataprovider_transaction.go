package vfs

// An IsolationLevel determines the isolation between concurrent transactions.
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

// TxOptions are used to configure and spawn a new concurrent transaction.
type TxOptions struct {
	// Isolation is the transaction isolation level.
	Isolation IsolationLevel
	// If true, a transaction must deny all modification attempts.
	ReadOnly bool
}

// A TransactionableDataProvider supports also the usual style. If it implicitly creates a transaction per
// operation or in time slices or other criteria is implementation specific.
type TransactionableDataProvider interface {
	// Begins either a ReadOnly or ReadWrite transaction. ReadOnly may be ignored and used for optimizations only.
	// The returned Transaction must be closed by either committing or by rollback.
	Begin(opts TxOptions) (Tx, error)
	DataProvider
}

// A Tx is the DataProvider contract providing commit and rollback methods.
type Tx interface {
	Commit() error
	Rollback() error
	// A simple close of the DataProvider without a commit will perform a Rollback.
	DataProvider
}
