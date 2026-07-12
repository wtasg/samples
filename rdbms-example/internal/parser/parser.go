// Package parser implements a hand-written lexer and recursive-descent parser
// for a small SQL subset.
//
// Supported statements:
//
//	CREATE TABLE t (col TYPE, ...)
//	INSERT INTO t VALUES (v1, v2, ...)
//	SELECT */col,... FROM t [WHERE expr] [ORDER BY col [DESC]]
//	UPDATE t SET col=val,... WHERE expr
//	DELETE FROM t WHERE expr
//	DROP TABLE t
//
// WHERE expr: col OP val | col BETWEEN val AND val | col LIKE 'pat'
// OP: =  !=  <  >  <=  >=
package parser

import (
	"fmt"
	"strings"
	"unicode"
)

// ── Token types ─────────────────────────────────────────────────────────────

type tokenKind int

const (
	tokEOF tokenKind = iota
	tokIdent
	tokIntLit
	tokFloatLit
	tokStrLit
	tokEQ   // =
	tokNEQ  // !=
	tokLT   // <
	tokGT   // >
	tokLTE  // <=
	tokGTE  // >=
	tokStar // *
	tokComma
	tokLParen
	tokRParen
	tokSemi
	// Keywords
	tokCreate
	tokTable
	tokInsert
	tokInto
	tokValues
	tokSelect
	tokFrom
	tokWhere
	tokUpdate
	tokSet
	tokDelete
	tokDrop
	tokAnd
	tokOr
	tokBetween
	tokLike
	tokOrder
	tokBy
	tokAsc
	tokDesc
	tokInt
	tokText
	tokFloat
	tokBool
	tokNot
	tokNull
)

var keywords = map[string]tokenKind{
	"CREATE":  tokCreate,
	"TABLE":   tokTable,
	"INSERT":  tokInsert,
	"INTO":    tokInto,
	"VALUES":  tokValues,
	"SELECT":  tokSelect,
	"FROM":    tokFrom,
	"WHERE":   tokWhere,
	"UPDATE":  tokUpdate,
	"SET":     tokSet,
	"DELETE":  tokDelete,
	"DROP":    tokDrop,
	"AND":     tokAnd,
	"OR":      tokOr,
	"BETWEEN": tokBetween,
	"LIKE":    tokLike,
	"ORDER":   tokOrder,
	"BY":      tokBy,
	"ASC":     tokAsc,
	"DESC":    tokDesc,
	"INT":     tokInt,
	"TEXT":    tokText,
	"FLOAT":   tokFloat,
	"BOOL":    tokBool,
	"NOT":     tokNot,
	"NULL":    tokNull,
}

type token struct {
	kind tokenKind
	val  string
}

// ── Lexer ────────────────────────────────────────────────────────────────────

type lexer struct {
	input []rune
	pos   int
}

func newLexer(s string) *lexer { return &lexer{input: []rune(s)} }

func (l *lexer) peek() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *lexer) advance() rune {
	ch := l.input[l.pos]
	l.pos++
	return ch
}

func (l *lexer) skipWS() {
	for l.pos < len(l.input) && unicode.IsSpace(l.input[l.pos]) {
		l.pos++
	}
}

func (l *lexer) nextToken() token {
	l.skipWS()
	if l.pos >= len(l.input) {
		return token{tokEOF, ""}
	}

	ch := l.peek()

	// String literal
	if ch == '\'' {
		l.advance()
		var sb strings.Builder
		for l.pos < len(l.input) && l.peek() != '\'' {
			c := l.advance()
			if c == '\\' && l.pos < len(l.input) {
				c = l.advance()
			}
			sb.WriteRune(c)
		}
		if l.pos < len(l.input) {
			l.advance() // closing '
		}
		return token{tokStrLit, sb.String()}
	}

	// Number literal
	if unicode.IsDigit(ch) || (ch == '-' && l.pos+1 < len(l.input) && unicode.IsDigit(l.input[l.pos+1])) {
		var sb strings.Builder
		if ch == '-' {
			sb.WriteRune(l.advance())
		}
		isFloat := false
		for l.pos < len(l.input) && (unicode.IsDigit(l.peek()) || l.peek() == '.') {
			if l.peek() == '.' {
				isFloat = true
			}
			sb.WriteRune(l.advance())
		}
		if isFloat {
			return token{tokFloatLit, sb.String()}
		}
		return token{tokIntLit, sb.String()}
	}

	// Identifier or keyword
	if unicode.IsLetter(ch) || ch == '_' {
		var sb strings.Builder
		for l.pos < len(l.input) && (unicode.IsLetter(l.peek()) || unicode.IsDigit(l.peek()) || l.peek() == '_') {
			sb.WriteRune(l.advance())
		}
		upper := strings.ToUpper(sb.String())
		if kind, ok := keywords[upper]; ok {
			return token{kind, upper}
		}
		return token{tokIdent, sb.String()}
	}

	// Operators and punctuation
	l.advance()
	switch ch {
	case '=':
		return token{tokEQ, "="}
	case '!':
		if l.peek() == '=' {
			l.advance()
			return token{tokNEQ, "!="}
		}
	case '<':
		if l.peek() == '=' {
			l.advance()
			return token{tokLTE, "<="}
		}
		return token{tokLT, "<"}
	case '>':
		if l.peek() == '=' {
			l.advance()
			return token{tokGTE, ">="}
		}
		return token{tokGT, ">"}
	case '*':
		return token{tokStar, "*"}
	case ',':
		return token{tokComma, ","}
	case '(':
		return token{tokLParen, "("}
	case ')':
		return token{tokRParen, ")"}
	case ';':
		return token{tokSemi, ";"}
	}
	return token{tokEOF, string(ch)}
}

// tokenize returns all tokens for the input.
func tokenize(s string) []token {
	l := newLexer(s)
	var toks []token
	for {
		t := l.nextToken()
		toks = append(toks, t)
		if t.kind == tokEOF {
			break
		}
	}
	return toks
}

// ── AST ─────────────────────────────────────────────────────────────────────

// Statement is the top-level AST node.
type Statement interface{ stmtNode() }

// ColumnDef is a column definition in CREATE TABLE.
type ColumnDef struct {
	Name string
	Type string // "INT", "TEXT", "FLOAT", "BOOL"
}

// WhereExpr is a single WHERE predicate.
type WhereExpr struct {
	Column string
	Op     string // "=", "!=", "<", ">", "<=", ">=", "LIKE", "BETWEEN"
	Value  string
	Value2 string // only for BETWEEN
}

type CreateTableStmt struct {
	Table   string
	Columns []ColumnDef
}

type InsertStmt struct {
	Table  string
	Values []string // raw literal strings in column order
}

type SelectStmt struct {
	Table   string
	Columns []string // nil → SELECT *
	Where   *WhereExpr
	OrderBy string
	Desc    bool
}

type UpdateStmt struct {
	Table       string
	Assignments map[string]string // col → raw value string
	Where       *WhereExpr
}

type DeleteStmt struct {
	Table string
	Where *WhereExpr
}

type DropTableStmt struct {
	Table string
}

func (s *CreateTableStmt) stmtNode() {}
func (s *InsertStmt) stmtNode()      {}
func (s *SelectStmt) stmtNode()      {}
func (s *UpdateStmt) stmtNode()      {}
func (s *DeleteStmt) stmtNode()      {}
func (s *DropTableStmt) stmtNode()   {}

// ── Parser ───────────────────────────────────────────────────────────────────

type parser struct {
	tokens []token
	pos    int
}

func newParser(tokens []token) *parser { return &parser{tokens: tokens} }

func (p *parser) peek() token {
	if p.pos >= len(p.tokens) {
		return token{tokEOF, ""}
	}
	return p.tokens[p.pos]
}

func (p *parser) advance() token {
	t := p.tokens[p.pos]
	p.pos++
	return t
}

func (p *parser) expect(kind tokenKind) (token, error) {
	t := p.peek()
	if t.kind != kind {
		return token{}, fmt.Errorf("expected token kind %d, got %q", kind, t.val)
	}
	p.advance()
	return t, nil
}

func (p *parser) expectIdent() (string, error) {
	t := p.peek()
	if t.kind != tokIdent {
		return "", fmt.Errorf("expected identifier, got %q", t.val)
	}
	p.advance()
	return t.val, nil
}

func (p *parser) parseColType() (string, error) {
	t := p.peek()
	switch t.kind {
	case tokInt:
		p.advance()
		return "INT", nil
	case tokText:
		p.advance()
		return "TEXT", nil
	case tokFloat:
		p.advance()
		return "FLOAT", nil
	case tokBool:
		p.advance()
		return "BOOL", nil
	}
	return "", fmt.Errorf("expected column type (INT/TEXT/FLOAT/BOOL), got %q", t.val)
}

func (p *parser) parseLiteral() (string, error) {
	t := p.peek()
	switch t.kind {
	case tokIntLit, tokFloatLit, tokStrLit:
		p.advance()
		return t.val, nil
	case tokNull:
		p.advance()
		return "NULL", nil
	case tokIdent:
		valUpper := strings.ToUpper(t.val)
		if valUpper == "TRUE" || valUpper == "FALSE" {
			p.advance()
			return t.val, nil
		}
	}
	return "", fmt.Errorf("expected literal value, got %q", t.val)
}

// ─── Statement parsers ────────────────────────────────────────────────────────

func (p *parser) parseCreate() (*CreateTableStmt, error) {
	if _, err := p.expect(tokTable); err != nil {
		return nil, err
	}
	name, err := p.expectIdent()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(tokLParen); err != nil {
		return nil, err
	}

	var cols []ColumnDef
	for {
		colName, err := p.expectIdent()
		if err != nil {
			return nil, err
		}
		colType, err := p.parseColType()
		if err != nil {
			return nil, err
		}
		cols = append(cols, ColumnDef{colName, colType})

		t := p.peek()
		if t.kind == tokRParen {
			p.advance()
			break
		}
		if _, err := p.expect(tokComma); err != nil {
			return nil, err
		}
	}
	return &CreateTableStmt{Table: name, Columns: cols}, nil
}

func (p *parser) parseInsert() (*InsertStmt, error) {
	if _, err := p.expect(tokInto); err != nil {
		return nil, err
	}
	name, err := p.expectIdent()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(tokValues); err != nil {
		return nil, err
	}
	if _, err := p.expect(tokLParen); err != nil {
		return nil, err
	}

	var vals []string
	for {
		v, err := p.parseLiteral()
		if err != nil {
			return nil, err
		}
		vals = append(vals, v)

		t := p.peek()
		if t.kind == tokRParen {
			p.advance()
			break
		}
		if _, err := p.expect(tokComma); err != nil {
			return nil, err
		}
	}
	return &InsertStmt{Table: name, Values: vals}, nil
}

func (p *parser) parseSelect() (*SelectStmt, error) {
	stmt := &SelectStmt{}

	// Column list or *
	if p.peek().kind == tokStar {
		p.advance()
	} else {
		for {
			col, err := p.expectIdent()
			if err != nil {
				return nil, err
			}
			stmt.Columns = append(stmt.Columns, col)
			if p.peek().kind != tokComma {
				break
			}
			p.advance()
		}
	}

	if _, err := p.expect(tokFrom); err != nil {
		return nil, err
	}
	name, err := p.expectIdent()
	if err != nil {
		return nil, err
	}
	stmt.Table = name

	// Optional WHERE
	if p.peek().kind == tokWhere {
		p.advance()
		w, err := p.parseWhere()
		if err != nil {
			return nil, err
		}
		stmt.Where = w
	}

	// Optional ORDER BY
	if p.peek().kind == tokOrder {
		p.advance()
		if _, err := p.expect(tokBy); err != nil {
			return nil, err
		}
		col, err := p.expectIdent()
		if err != nil {
			return nil, err
		}
		stmt.OrderBy = col
		if p.peek().kind == tokDesc {
			p.advance()
			stmt.Desc = true
		} else if p.peek().kind == tokAsc {
			p.advance()
		}
	}
	return stmt, nil
}

func (p *parser) parseUpdate() (*UpdateStmt, error) {
	name, err := p.expectIdent()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(tokSet); err != nil {
		return nil, err
	}

	assignments := make(map[string]string)
	for {
		col, err := p.expectIdent()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(tokEQ); err != nil {
			return nil, err
		}
		val, err := p.parseLiteral()
		if err != nil {
			return nil, err
		}
		assignments[col] = val

		if p.peek().kind != tokComma {
			break
		}
		p.advance()
	}

	stmt := &UpdateStmt{Table: name, Assignments: assignments}
	if p.peek().kind == tokWhere {
		p.advance()
		w, err := p.parseWhere()
		if err != nil {
			return nil, err
		}
		stmt.Where = w
	}
	return stmt, nil
}

func (p *parser) parseDelete() (*DeleteStmt, error) {
	if _, err := p.expect(tokFrom); err != nil {
		return nil, err
	}
	name, err := p.expectIdent()
	if err != nil {
		return nil, err
	}
	stmt := &DeleteStmt{Table: name}
	if p.peek().kind == tokWhere {
		p.advance()
		w, err := p.parseWhere()
		if err != nil {
			return nil, err
		}
		stmt.Where = w
	}
	return stmt, nil
}

func (p *parser) parseDrop() (*DropTableStmt, error) {
	if _, err := p.expect(tokTable); err != nil {
		return nil, err
	}
	name, err := p.expectIdent()
	if err != nil {
		return nil, err
	}
	return &DropTableStmt{Table: name}, nil
}

func (p *parser) parseWhere() (*WhereExpr, error) {
	col, err := p.expectIdent()
	if err != nil {
		return nil, err
	}

	t := p.peek()
	switch t.kind {
	case tokBetween:
		p.advance()
		lo, err := p.parseLiteral()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(tokAnd); err != nil {
			return nil, err
		}
		hi, err := p.parseLiteral()
		if err != nil {
			return nil, err
		}
		return &WhereExpr{Column: col, Op: "BETWEEN", Value: lo, Value2: hi}, nil

	case tokLike:
		p.advance()
		pat, err := p.parseLiteral()
		if err != nil {
			return nil, err
		}
		return &WhereExpr{Column: col, Op: "LIKE", Value: pat}, nil

	case tokEQ, tokNEQ, tokLT, tokGT, tokLTE, tokGTE:
		op := t.val
		p.advance()
		val, err := p.parseLiteral()
		if err != nil {
			return nil, err
		}
		return &WhereExpr{Column: col, Op: op, Value: val}, nil
	}

	return nil, fmt.Errorf("expected operator after column %q, got %q", col, t.val)
}

// Parse parses a single SQL statement (without the trailing semicolon).
func Parse(sql string) (Statement, error) {
	// Strip trailing semicolon and whitespace.
	sql = strings.TrimRight(strings.TrimSpace(sql), ";")
	tokens := tokenize(sql)
	p := newParser(tokens)

	t := p.advance()
	switch t.kind {
	case tokCreate:
		return p.parseCreate()
	case tokInsert:
		return p.parseInsert()
	case tokSelect:
		return p.parseSelect()
	case tokUpdate:
		return p.parseUpdate()
	case tokDelete:
		return p.parseDelete()
	case tokDrop:
		return p.parseDrop()
	case tokEOF:
		return nil, nil
	}
	return nil, fmt.Errorf("unknown statement starting with %q", t.val)
}
