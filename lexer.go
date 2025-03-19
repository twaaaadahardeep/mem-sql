package memsql

import (
	"fmt"
	"strings"
)

type Location struct {
	line   uint
	column uint
}

type Keyword string

const (
	createKeyword Keyword = "create"
	selectKeyword Keyword = "select"
	fromKeyword   Keyword = "from"
	tableKeyword  Keyword = "table"
	insertKeyword Keyword = "insert"
	intoKeyword   Keyword = "into"
	valuesKeyword Keyword = "values"
	intKeyword    Keyword = "int"
	textKeyword   Keyword = "text"
)

// create table <tablename> ;
// insert into <tablename> (<columns>) values (<values>);
// select * from <tablename>;

type Symbol string

const (
	semiColonSymbol  Symbol = ";"
	asteriskSymbol   Symbol = "*"
	commaSymbol      Symbol = ","
	leftParenSymbol  Symbol = "("
	rightParenSymbol Symbol = ")"
)

type TokenKind uint

const (
	keywordKind TokenKind = iota
	symbolKind
	identifierKind
	textKind
	integerKind
)

type Token struct {
	value    string
	kind     TokenKind
	location Location
}

type cursor struct {
	location Location
	pointer  uint
}

func (t *Token) equals(ot *Token) bool {
	return t.value == ot.value && t.kind == ot.kind
}

type lexer func(string, cursor) (*Token, cursor, bool)

func lexNumber(source string, ic cursor) (*Token, cursor, bool) {
	cursor := ic

	periodFound := false
	exponentFound := false

	for ; cursor.pointer < uint(len(source)); cursor.pointer++ {
		c := source[cursor.pointer]
		cursor.location.column++

		isDigit := c >= '0' && c <= '9'
		isPeriod := c == '.'
		isExponent := c == 'e'

		if cursor.pointer == ic.pointer {
			if !isDigit && !isPeriod {
				return nil, ic, false
			}

			periodFound = isPeriod
			continue
		}

		if isPeriod {
			if periodFound {
				return nil, ic, false
			}

			periodFound = true
			continue
		}

		if isExponent {
			if exponentFound {
				return nil, ic, false
			}

			periodFound = true
			exponentFound = true

			// exponent should be followed by some number (can't be last character)
			if cursor.pointer == uint(len(source)-1) {
				return nil, ic, false
			}

			cNext := source[cursor.pointer+1]
			if cNext == '-' || cNext == '+' {
				cursor.pointer++
				cursor.location.column++
			}

			continue
		}

		if !isDigit {
			break
		}
	}

	if cursor.pointer == ic.pointer {
		return nil, ic, false
	}

	return &Token{
		value:    source[ic.pointer:cursor.pointer],
		kind:     integerKind,
		location: ic.location,
	}, cursor, true
}

func lexCharacterDelimited(source string, ic cursor, delimiter byte) (*Token, cursor, bool) {
	cursor := ic

	if (len(source[cursor.pointer:])) == 0 {
		return nil, ic, false
	}

	if source[cursor.pointer] != delimiter {
		return nil, ic, false
	}

	cursor.pointer++
	cursor.location.column++

	var value []byte
	for ; cursor.pointer < uint(len(source)); cursor.pointer++ {
		c := source[cursor.pointer]

		if c == delimiter {
			// SQL Escapes are via double characters, not backslash
			if cursor.pointer+1 >= uint(len(source)) || source[cursor.pointer+1] != delimiter {
				cursor.pointer++
				cursor.location.column++
				return &Token{
					value:    string(value),
					kind:     textKind,
					location: ic.location,
				}, cursor, true
			}

			value = append(value, delimiter)
			cursor.pointer++
			cursor.location.column++

		}

		value = append(value, c)
		cursor.location.column++
	}

	return nil, ic, false
}

func lexString(source string, ic cursor) (*Token, cursor, bool) {
	return lexCharacterDelimited(source, ic, '\'')
}

func lexSymbol(source string, ic cursor) (*Token, cursor, bool) {
	c := source[ic.pointer]
	cursor := ic

	cursor.pointer++
	cursor.location.column++

	switch c {
	case '\n':
		cursor.location.line++
		cursor.location.column = 0
		fallthrough
	case '\t':
		fallthrough
	case ' ':
		return nil, cursor, true
	}

	symbols := []Symbol{
		commaSymbol,
		semiColonSymbol,
		asteriskSymbol,
		leftParenSymbol,
		rightParenSymbol,
	}

	var options []string

	for _, s := range symbols {
		options = append(options, string(s))
	}

	match := longestMatch(source, ic, options)

	if match == "" {
		return nil, ic, false
	}

	cursor.pointer = ic.pointer + uint(len(match))
	cursor.location.column = ic.location.column + uint(len(match))

	return &Token{
		value:    match,
		location: ic.location,
		kind:     symbolKind,
	}, cursor, true
}

func lexKeyword(source string, ic cursor) (*Token, cursor, bool) {
	cursor := ic
	keywords := []Keyword{
		createKeyword,
		selectKeyword,
		fromKeyword,
		tableKeyword,
		insertKeyword,
		intoKeyword,
		valuesKeyword,
		intKeyword,
		textKeyword,
	}

	var options []string
	for _, k := range keywords {
		options = append(options, string(k))
	}

	match := longestMatch(source, ic, options)
	if match == "" {
		return nil, ic, false
	}

	cursor.pointer = ic.pointer + uint(len(match))
	cursor.location.column = ic.location.column + uint(len(match))

	return &Token{
		value:    match,
		location: ic.location,
		kind:     keywordKind,
	}, cursor, true
}

func longestMatch(source string, ic cursor, options []string) string {
	var value []byte
	var skipList []int
	var match string

	cursor := ic

	for cursor.pointer < uint(len(source)) {
		value = append(value, strings.ToLower(string(source[cursor.pointer]))...)
		cursor.pointer++

	match:
		for i, option := range options {
			for _, skip := range skipList {
				if i == skip {
					continue match
				}
			}

			// Deal with cases like INT and INTO
			if option == string(value) {
				skipList = append(skipList, i)
				if len(option) > len(match) {
					match = option
				}

				continue
			}

			sharesPrefix := string(value) == option[:cursor.pointer-ic.pointer]
			tooLong := len(value) > len(option)

			if tooLong || !sharesPrefix {
				skipList = append(skipList, i)
			}
		}

		if len(skipList) == len(options) {
			break
		}
	}

	return match
}

func lexIdentifier(source string, ic cursor) (*Token, cursor, bool) {
	// handle separately if it is a double quotes
	if token, newCursor, ok := lexCharacterDelimited(source, ic, '"'); ok {
		// Overwrite from stringkind to identifierkind
		token.kind = identifierKind
		return token, newCursor, true
	}

	cursor := ic

	c := source[cursor.pointer]

	isAlpha := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
	if !isAlpha {
		return nil, ic, false
	}

	cursor.pointer++
	cursor.location.column++

	value := []byte{c}

	for ; cursor.pointer < uint(len(source)); cursor.pointer++ {
		c = source[cursor.pointer]

		isAlpha := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
		isNumber := c >= '0' && c <= '9'
		if isAlpha || isNumber || c == '_' || c == '$' {
			value = append(value, c)
			cursor.location.column++
			continue
		}

		break
	}

	return &Token{
		value:    strings.ToLower(string(value)),
		kind:     identifierKind,
		location: ic.location,
	}, cursor, true
}

func lex(source string) ([]*Token, error) {
	var tokens []*Token
	cursor := cursor{}

lex:
	for cursor.pointer < uint(len(source)) {
		lexers := []lexer{lexKeyword, lexSymbol, lexString, lexNumber, lexIdentifier}

		for _, l := range lexers {
			if t, newCursor, ok := l(source, cursor); ok {
				cursor = newCursor

				if t != nil {
					tokens = append(tokens, t)
				}

				continue lex
			}
		}

		hint := ""
		if len(tokens) > 0 {
			hint = " after " + tokens[len(tokens)-1].value
		}

		return nil, fmt.Errorf("unable to lex tokens%s, at %d:%d", hint, cursor.location.line, cursor.location.column)
	}

	return tokens, nil
}
