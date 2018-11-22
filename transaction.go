package vfs

type TransactionMode uint32

const (
	ReadOnly        TransactionMode = 1 << (32 - 1 - iota)
)
//TBD
type Transactionable interface {
	Begin(mode TransactionMode) (Transaction, error)
}

type Transaction interface{
	DataProvider
	Commit()error
	Rollback()error
}

