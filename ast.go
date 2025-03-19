package memsql

type AstKind uint

const (
	SelectKind AstKind = iota
	CreateTableKind
	InsertKind
)

type ExpressionKind uint

const (
	LiteralKind ExpressionKind = iota
)

type Expression struct {
	Literal *Token
	Kind    ExpressionKind
}

type InsertStatement struct {
	Table  Token
	Values *[]*Expression
}

type ColumnDefinition struct {
	Name     Token
	Datatype Token
}

type CreateTableStatement struct {
	Name    Token
	Columns *[]*ColumnDefinition
}

type SelectStatement struct {
	Item []*Expression
	From Token
}

type Statement struct {
	SelectStatement      *SelectStatement
	CreateTableStatement *CreateTableStatement
	InsertStatement      *InsertStatement
	Kind                 AstKind
}

type Ast struct {
	Statements []*Statement
}
