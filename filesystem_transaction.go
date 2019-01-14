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

// A TransactionableFileSystem supports also the usual style. If it implicitly creates a transaction per
// operation or in time slices or other criteria is implementation specific.
type TransactionableFileSystem interface {
	// Begins either a ReadOnly or ReadWrite transaction. ReadOnly may be ignored and used for optimizations only.
	// The returned Transaction must be closed by either committing or by rollback.
	Begin(opts TxOptions) (Tx, error)
	FileSystem
}

// A Tx is the FileSystem contract providing commit and rollback methods but also is a normal FileSystem.
// An implementation should rollback, if a transaction has not been explicitly closed by a
// Commit or Rollback.
type Tx interface {
	Commit() error
	Rollback() error
	// A simple close of the FileSystem without a commit will perform a Rollback.
	FileSystem
}
