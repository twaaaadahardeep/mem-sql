package memsql

import (
	"errors"
	"fmt"
)

func tokenFromKeyword(k Keyword) Token {
	return Token{
		value: string(k),
		kind:  keywordKind,
	}
}

func tokenFromSymbol(s Symbol) Token {
	return Token{
		value: string(s),
		kind:  symbolKind,
	}
}

func expectToken(tokens []*Token, cursor uint, t Token) bool {
	if cursor >= uint((len(tokens))) {
		return false
	}

	return t.equals(tokens[cursor])
}

func helpMessage(tokens []*Token, cursor uint, msg string) {
	var c *Token
	if cursor < uint(len(tokens)) {
		c = tokens[cursor]
	} else {
		c = tokens[cursor-1]
	}

	fmt.Printf("[%d, %d]: %s, got %s\n", c.location.line, c.location.column, msg, c.value)
}

// parseToken helper will look for a token of a particular token kind
func parseToken(tokens []*Token, ic uint, kind TokenKind) (*Token, uint, bool) {
	cursor := ic

	if cursor >= uint(len(tokens)) {
		return nil, ic, false
	}

	cur := tokens[cursor]
	if cur.kind == kind {
		fmt.Println("parseToken: " + cur.value)
		return cur, cursor + 1, true
	}

	return nil, ic, false
}

// parseExpression helper will look for a numeric, string, or identifier token
func parseExpression(tokens []*Token, ic uint, _ Token) (*Expression, uint, bool) {
	cursor := ic

	kinds := []TokenKind{identifierKind, textKind, integerKind}
	for _, kind := range kinds {
		t, newCursor, ok := parseToken(tokens, cursor, kind)
		if ok {
			return &Expression{
				Literal: t,
				Kind:    LiteralKind,
			}, newCursor, true
		}
	}

	return nil, ic, false
}

// parseExpressions helper will look for tokens separated by a comma until a delimiter is found
func parseExpressions(tokens []*Token, ic uint, delimiters []Token) (*[]*Expression, uint, bool) {
	cursor := ic

	exps := []*Expression{}

outer:
	for {
		if cursor >= uint(len(tokens)) {
			return nil, ic, false
		}

		cur := tokens[cursor]
		for _, d := range delimiters {
			if d.equals(cur) {
				break outer
			}
		}

		// look for comma
		if len(exps) > 0 {
			if !expectToken(tokens, cursor, tokenFromSymbol(commaSymbol)) {
				helpMessage(tokens, cursor, "Expected comma")
				return nil, ic, false
			}

			cursor++
		}

		// look for expression
		exp, newCursor, ok := parseExpression(tokens, cursor, tokenFromSymbol(commaSymbol))
		if !ok {
			helpMessage(tokens, cursor, "Expected expression")
			return nil, ic, false
		}

		cursor = newCursor
		exps = append(exps, exp)
	}

	return &exps, cursor, true
}

func parseSelectStatement(tokens []*Token, ic uint, delimiter Token) (*SelectStatement, uint, bool) {
	cursor := ic

	if !expectToken(tokens, cursor, tokenFromKeyword(selectKeyword)) {
		return nil, ic, false
	}

	cursor++

	slct := SelectStatement{}

	exps, newCursor, ok := parseExpressions(tokens, cursor, []Token{tokenFromKeyword(fromKeyword), delimiter})
	if !ok {
		return nil, ic, false
	}

	slct.Item = *exps
	cursor = newCursor

	if expectToken(tokens, cursor, tokenFromKeyword(fromKeyword)) {
		cursor++

		fr, newCurs, ok := parseToken(tokens, cursor, identifierKind)
		if !ok {
			helpMessage(tokens, cursor, "Expected FROM token")
			return nil, ic, false
		}

		slct.From = *fr
		cursor = newCurs
	}

	return &slct, cursor, true
}

func parseInsertStatement(tokens []*Token, ic uint, delimiter Token) (*InsertStatement, uint, bool) {
	cursor := ic

	// Look for INSERT
	if !expectToken(tokens, cursor, tokenFromKeyword(insertKeyword)) {
		return nil, ic, false
	}
	cursor++

	// Look for INTO
	if !expectToken(tokens, cursor, tokenFromKeyword(intoKeyword)) {
		helpMessage(tokens, cursor, "Expected keyword INTO")
		return nil, ic, false
	}
	cursor++

	// Look for tableName
	table, newCursor, ok := parseToken(tokens, cursor, identifierKind)
	if !ok {
		helpMessage(tokens, cursor, "Expected table name")
		return nil, ic, false
	}
	cursor = newCursor

	// Look for VALUES
	if !expectToken(tokens, cursor, tokenFromKeyword(valuesKeyword)) {
		helpMessage(tokens, cursor, "Expected keyword VALUES")
		return nil, ic, false
	}
	cursor++

	// Look for left parenthesis
	if !expectToken(tokens, cursor, tokenFromSymbol(leftParenSymbol)) {
		helpMessage(tokens, cursor, "Expected '('")
		return nil, ic, false
	}
	cursor++

	// Look for expressions
	values, newCursor, ok := parseExpressions(tokens, cursor, []Token{tokenFromSymbol(rightParenSymbol)})
	if !ok {
		return nil, ic, false
	}
	cursor = newCursor

	// Look for right parenthesis
	if !expectToken(tokens, cursor, tokenFromSymbol(rightParenSymbol)) {
		helpMessage(tokens, cursor, "Expected ')'")
		return nil, ic, false
	}
	cursor++

	return &InsertStatement{
		Table:  *table,
		Values: values,
	}, cursor, true
}

// parseColumnDefinitions helper will look column names followed by column types
// separated by a comma and ending with some delimiter
func parseColumnDefinitions(tokens []*Token, ic uint, delimiter Token) (*[]*ColumnDefinition, uint, bool) {
	cursor := ic

	var cds []*ColumnDefinition
	for {
		if cursor >= uint(len(tokens)) {
			return nil, ic, false
		}

		// Look for delimiter
		cur := tokens[cursor]
		fmt.Println("inside column func... current = " + cur.value)
		if delimiter.equals(cur) {
			fmt.Println("inside break... current = " + cur.value)
			break
		}

		// Look for comma
		if len(cds) > 0 {
			var ok bool
			_, cursor, ok = parseTokenAnother(tokens, cursor, tokenFromSymbol(commaSymbol))
			if !ok {
				helpMessage(tokens, cursor, "Expected comma")
				return nil, ic, false
			}
		}

		// Look for column name
		id, newCursor, ok := parseToken(tokens, cursor, identifierKind)
		if !ok {
			helpMessage(tokens, cursor, "Expected column name")
			return nil, ic, false
		}
		cursor = newCursor

		// Look for column type
		t, newCursor, ok := parseToken(tokens, cursor, keywordKind)
		if !ok {
			helpMessage(tokens, cursor, "Expected column type")
			return nil, ic, false
		}
		cursor = newCursor

		cds = append(cds, &ColumnDefinition{
			Name:     *id,
			Datatype: *t,
		})
	}

	return &cds, cursor, true
}

func parseCreateTableStatement(tokens []*Token, ic uint, _ Token) (*CreateTableStatement, uint, bool) {
	cursor := ic
	ok := false

	// Look for CREATE
	_, cursor, ok = parseTokenAnother(tokens, cursor, tokenFromKeyword(createKeyword))
	if !ok {
		return nil, ic, false
	}

	// Look for TABLE
	_, cursor, ok = parseTokenAnother(tokens, cursor, tokenFromKeyword(tableKeyword))
	if !ok {
		return nil, ic, false
	}

	// Look for tableName
	table, newCursor, ok := parseToken(tokens, cursor, identifierKind)
	if !ok {
		helpMessage(tokens, cursor, "Expected table name")
		return nil, ic, false
	}
	cursor = newCursor

	// Look for left parenthesis
	_, cursor, ok = parseTokenAnother(tokens, cursor, tokenFromSymbol(leftParenSymbol))
	if !ok {
		helpMessage(tokens, cursor, "Expected '('")
		return nil, ic, false
	}

	// Look for column definitions
	cols, newCursor, ok := parseColumnDefinitions(tokens, cursor, tokenFromSymbol(rightParenSymbol))
	if !ok {
		return nil, ic, false
	}
	cursor = newCursor

	// Look for right parenthesis
	fmt.Println("Before right paren...")
	_, cursor, ok = parseTokenAnother(tokens, cursor, tokenFromSymbol(rightParenSymbol))
	if !ok {
		fmt.Println("Right Paren failed")
		helpMessage(tokens, cursor, "Expected ')'")
		return nil, ic, false
	}

	fmt.Println("Completed Create...")
	return &CreateTableStatement{
		Name:    *table,
		Columns: cols,
	}, cursor, true
}

func parseStatement(tokens []*Token, ic uint, _ Token) (*Statement, uint, bool) {
	cursor := ic

	semicolonToken := tokenFromSymbol(semiColonSymbol)

	// Look for SELECT statement
	slct, newCursor, ok := parseSelectStatement(tokens, cursor, semicolonToken)
	if ok {
		return &Statement{
			SelectStatement: slct,
			Kind:            SelectKind,
		}, newCursor, true
	}

	// Look for INSERT statement
	inst, newCursor, ok := parseInsertStatement(tokens, cursor, semicolonToken)
	if ok {
		return &Statement{
			InsertStatement: inst,
			Kind:            InsertKind,
		}, newCursor, true
	}

	// Look for CREATE statement
	ctstmt, newCursor, ok := parseCreateTableStatement(tokens, cursor, semicolonToken)
	if ok {
		fmt.Println("Completed Create... ok done")
		return &Statement{
			CreateTableStatement: ctstmt,
			Kind:                 CreateTableKind,
		}, newCursor, true
	}

	return nil, ic, false
}

func parseTokenAnother(tokens []*Token, initialCursor uint, t Token) (*Token, uint, bool) {
	cursor := initialCursor

	if cursor >= uint(len(tokens)) {
		return nil, initialCursor, false
	}

	if p := tokens[cursor]; t.equals(p) {
		fmt.Println("parseTokenAnother: " + p.value)
		return p, cursor + 1, true
	}

	return nil, initialCursor, false
}

func Parse(source string) (*Ast, error) {
	tokens, err := lex(source)
	if err != nil {
		return nil, err
	}

	semicolonToken := tokenFromSymbol(semiColonSymbol)
	if len(tokens) > 0 && !tokens[len(tokens)-1].equals(&semicolonToken) {
		tokens = append(tokens, &semicolonToken)
	}

	a := Ast{}
	cursor := uint(0)
	// for _, t := range tokens {
	// 	fmt.Printf("val: %s, kind: %d", t.value, t.kind)
	// }

	for cursor < uint(len(tokens)) {
		stmt, newCursor, ok := parseStatement(tokens, cursor, tokenFromSymbol(semiColonSymbol))

		if !ok {
			helpMessage(tokens, cursor, "Expected statement")
			return nil, errors.New("Failed to parse, expected statement")
		}

		cursor = newCursor

		a.Statements = append(a.Statements, stmt)

		atLeastOneSemicolon := false
		for {
			_, cursor, ok = parseTokenAnother(tokens, cursor, tokenFromSymbol(semiColonSymbol))
			if ok {
				atLeastOneSemicolon = true
			} else {
				break
			}
		}

		if !atLeastOneSemicolon {
			helpMessage(tokens, cursor, "Expected semicolon between statements")
			return nil, errors.New("Missing semicolon between statements")
		}
	}

	return &a, nil
}
