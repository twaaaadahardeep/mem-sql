package memsql

import "errors"

type ColumnType uint

const (
	TextType ColumnType = iota
	IntType
)

type Cell interface {
	AsText() string
	AsInt32() int32
}

type Results struct {
	Columns []struct {
		Type ColumnType
		Name string
	}
	Rows [][]Cell
}

var (
	ErrTableDoesNotExists  = errors.New("table does not exist")
	ErrColumnDoesNotExists = errors.New("column does not exist")
	ErrInvalidSelectItem   = errors.New("select item is not valid")
	ErrInvalidDatatype     = errors.New("invalid Datatype")
	ErrMissingValues       = errors.New("missing values")
)

type Backend interface {
	CreateTable(*CreateTableStatement) error
	Insert(*InsertStatement) error
	Select(*SelectStatement) (*Results, error)
}
