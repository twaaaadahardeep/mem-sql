package memsql

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
)

type MemoryCell []byte

func (mc MemoryCell) AsInt32() int32 {
	var i int32
	err := binary.Read(bytes.NewBuffer(mc), binary.BigEndian, &i)
	if err != nil {
		panic(err)
	}
	return i
}

func (mc MemoryCell) AsText() string {
	return string(mc)
}

type Table struct {
	columns     []string
	columnTypes []ColumnType
	rows        [][]MemoryCell
}

type MemoryBackend struct {
	tables map[string]*Table
}

func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{
		tables: map[string]*Table{},
	}
}

func (mb *MemoryBackend) CreateTable(cts *CreateTableStatement) error {
	t := Table{}
	mb.tables[cts.Name.value] = &t
	if cts.Columns == nil {
		return ErrMissingValues
	}

	for _, cols := range *cts.Columns {
		t.columns = append(t.columns, cols.Name.value)

		var dt ColumnType
		switch cols.Datatype.value {
		case "int":
			dt = IntType
		case "text":
			dt = TextType
		default:
			return ErrInvalidDatatype
		}

		t.columnTypes = append(t.columnTypes, dt)
	}

	return nil
}

func (mb *MemoryBackend) tokenToCell(token *Token) MemoryCell {
	if token.kind == integerKind {
		buf := new(bytes.Buffer)
		i, err := strconv.Atoi(token.value)
		if err != nil {
			panic(err)
		}

		err = binary.Write(buf, binary.BigEndian, int32(i))
		if err != nil {
			panic(err)
		}

		return MemoryCell(buf.Bytes())
	}

	if token.kind == textKind {
		return MemoryCell(token.value)
	}

	return nil
}

func (mb *MemoryBackend) Insert(is *InsertStatement) error {
	table, ok := mb.tables[is.Table.value]
	if !ok {
		return ErrTableDoesNotExists
	}

	if is.Values == nil {
		return nil
	}

	row := []MemoryCell{}
	if len(*is.Values) != len(table.columns) {
		return ErrMissingValues
	}

	for _, value := range *is.Values {
		if value.Kind != LiteralKind {
			fmt.Println("Skipping non-literal")
			continue
		}

		row = append(row, mb.tokenToCell(value.Literal))
	}

	table.rows = append(table.rows, row)
	return nil
}

func (mb *MemoryBackend) Select(ss *SelectStatement) (*Results, error) {
	// get table from memory
	table, ok := mb.tables[ss.From.value]
	if !ok {
		return nil, ErrTableDoesNotExists
	}

	results := [][]Cell{}
	cols := []struct {
		Type ColumnType
		Name string
	}{}

	// iterate over table rows
	for i, row := range table.rows {
		result := []Cell{}
		isFirstRow := i == 0

		// iterate over the items (expression) we want to find in table
		for _, exp := range ss.Item {
			if exp.Kind != LiteralKind {
				fmt.Println("Skipping non-literal expression")
				continue
			}

			lit := exp.Literal
			if lit.kind == identifierKind {
				found := false

				// iterate over the table columns to find a matching expression
				for i, tableCol := range table.columns {
					// if we found it
					if tableCol == lit.value {
						if isFirstRow {
							// append column name and type of current column to columns list
							cols = append(cols, struct {
								Type ColumnType
								Name string
							}{
								Type: table.columnTypes[i],
								Name: lit.value,
							})
						}

						// append the row to the rows list, this one represents a single row
						result = append(result, row[i])
						found = true
						break
					}
				}

				// if no column matches our expression
				if !found {
					return nil, ErrColumnDoesNotExists
				}

				continue
			}

			return nil, ErrColumnDoesNotExists
		}

		// append the single row we have to the overall rows we need to output
		results = append(results, result)
	}

	return &Results{
		Columns: cols,
		Rows:    results,
	}, nil
}
