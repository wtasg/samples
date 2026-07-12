// Package toydb is the ToyDB Go client library.
//
// It provides a clean, idiomatic Go API over the ToyDB Connect-RPC service.
// The library is wire-compatible with standard gRPC clients; it uses the
// Connect protocol by default (HTTP/1.1 or HTTP/2, JSON or binary).
//
// Quick start:
//
//	c, err := toydb.NewClient("http://localhost:9090")
//	if err != nil { log.Fatal(err) }
//	defer c.Close()
//
//	// Schema operations
//	err = c.CreateTable("users", toydb.Schema{
//	    {Name: "id",   Type: toydb.INT},
//	    {Name: "name", Type: toydb.TEXT},
//	    {Name: "age",  Type: toydb.INT},
//	})
//
//	// Insert
//	err = c.Table("users").Insert(toydb.Row{"id": 1, "name": "Alice", "age": 30})
//
//	// Query with fluent builder
//	rows, err := c.Table("users").Where("name LIKE 'Al%'").Select()
//	rows, err := c.Table("users").Where("id BETWEEN 1 AND 10").OrderBy("age").Select()
//
//	// Raw SQL
//	result, err := c.Execute("SELECT * FROM users WHERE id = 1")
package toydb

import "fmt"

// ── Column types ──────────────────────────────────────────────────────────────

// ColumnType is the SQL data type of a column.
type ColumnType string

const (
	INT   ColumnType = "INT"
	TEXT  ColumnType = "TEXT"
	FLOAT ColumnType = "FLOAT"
	BOOL  ColumnType = "BOOL"
)

// ── Schema ────────────────────────────────────────────────────────────────────

// Column describes a column definition.
type Column struct {
	Name       string
	Type       ColumnType
	PrimaryKey bool // set by DescribeTable; the first column is always the PK
}

// Schema is a slice of column definitions, used when creating a table.
type Schema []Column

// String returns a SQL column-list for use in CREATE TABLE.
func (s Schema) String() string {
	parts := make([]string, len(s))
	for i, c := range s {
		parts[i] = c.Name + " " + string(c.Type)
	}
	return joinComma(parts)
}

// ── Row ───────────────────────────────────────────────────────────────────────

// Row is a database row: a map of column name → value.
// Values are typed according to the column schema:
//
//	INT   → int64
//	FLOAT → float64
//	TEXT  → string
//	BOOL  → bool
type Row map[string]any

// Int returns the column value as int64, or 0 if absent or wrong type.
func (r Row) Int(col string) int64 {
	switch v := r[col].(type) {
	case int64:
		return v
	case float64:
		return int64(v)
	}
	return 0
}

// Float returns the column value as float64.
func (r Row) Float(col string) float64 {
	switch v := r[col].(type) {
	case float64:
		return v
	case int64:
		return float64(v)
	}
	return 0
}

// Text returns the column value as string.
func (r Row) Text(col string) string {
	if v, ok := r[col].(string); ok {
		return v
	}
	return fmt.Sprintf("%v", r[col])
}

// Bool returns the column value as bool.
func (r Row) Bool(col string) bool {
	if v, ok := r[col].(bool); ok {
		return v
	}
	return false
}

// IsNull returns true if the column value is SQL NULL.
func (r Row) IsNull(col string) bool {
	return r[col] == nil
}

// ── Result ────────────────────────────────────────────────────────────────────

// Result is the output of a SELECT query.
type Result struct {
	Columns []string
	Rows    []Row
	Message string // populated for non-SELECT statements
}

// ── TableSchemaInfo ───────────────────────────────────────────────────────────

// TableSchemaInfo is returned by DescribeTable.
type TableSchemaInfo struct {
	Name    string
	Columns []Column
}

// ── helpers ───────────────────────────────────────────────────────────────────

func joinComma(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for _, p := range parts[1:] {
		out += ", " + p
	}
	return out
}
