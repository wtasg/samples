package toydb

import (
	"context"
	"fmt"
	"strings"
)

// TableQuery is a fluent builder for queries against a single table.
// Chain methods and call Select/Insert/Update/Delete to execute.
//
// Example:
//
//	rows, err := c.Table("users").
//	    Where("age > 18").
//	    OrderBy("name").
//	    Columns("id", "name").
//	    Select(ctx)
type TableQuery struct {
	client    *Client
	tableName string
	where     string
	orderBy   string
	desc      bool
	cols      []string
}

// Where sets the WHERE clause.  Raw SQL expression, e.g. "id = 1" or
// "name LIKE 'Al%'" or "id BETWEEN 5 AND 20".
func (q *TableQuery) Where(expr string) *TableQuery {
	q2 := *q
	q2.where = expr
	return &q2
}

// OrderBy sets the ORDER BY column.
func (q *TableQuery) OrderBy(col string) *TableQuery {
	q2 := *q
	q2.orderBy = col
	return &q2
}

// Desc adds DESC to the ORDER BY clause.
func (q *TableQuery) Desc() *TableQuery {
	q2 := *q
	q2.desc = true
	return &q2
}

// Columns restricts the SELECT to the named columns (default: all columns).
func (q *TableQuery) Columns(cols ...string) *TableQuery {
	q2 := *q
	q2.cols = cols
	return &q2
}

// Select executes the SELECT query and returns all matching rows.
// Uses the streaming Query RPC internally for efficiency.
func (q *TableQuery) Select(ctx context.Context) (*Result, error) {
	return q.client.Query(ctx, q.buildSelect())
}

// SelectStream executes the SELECT query and calls fn for each row.
func (q *TableQuery) SelectStream(ctx context.Context, fn func(columns []string, row Row) error) error {
	return q.client.QueryStream(ctx, q.buildSelect(), fn)
}

// Get fetches a single row by primary key (executes WHERE pk = id).
// Returns (nil, nil) if the row does not exist.
func (q *TableQuery) Get(ctx context.Context, pk int64) (Row, error) {
	result, err := q.client.Execute(ctx, fmt.Sprintf(
		"SELECT * FROM %s WHERE %s = %d",
		q.tableName, q.pkPlaceholder(), pk,
	))
	if err != nil {
		return nil, err
	}
	if len(result.Rows) == 0 {
		return nil, nil
	}
	return result.Rows[0], nil
}

// Insert inserts a new row. The Row must include all columns defined in the schema.
// Column names must match the table schema exactly.
func (q *TableQuery) Insert(ctx context.Context, row Row) error {
	// Build an ordered column/value list by fetching schema if needed.
	// For simplicity we build SQL from the row's keys in declaration order.
	sql, err := q.buildInsert(ctx, row)
	if err != nil {
		return err
	}
	_, err = q.client.Execute(ctx, sql)
	return err
}

// Update applies updates to all rows matching the WHERE clause.
// Returns the number of rows updated.
func (q *TableQuery) Update(ctx context.Context, updates Row) (int, error) {
	if q.where == "" {
		return 0, fmt.Errorf("toydb: Update requires a WHERE clause to avoid full-table updates")
	}
	sql := q.buildUpdate(updates)
	result, err := q.client.Execute(ctx, sql)
	if err != nil {
		return 0, err
	}
	var n int
	fmt.Sscanf(result.Message, "%d", &n)
	return n, nil
}

// Delete removes all rows matching the WHERE clause.
// Returns the number of rows deleted.
func (q *TableQuery) Delete(ctx context.Context) (int, error) {
	if q.where == "" {
		return 0, fmt.Errorf("toydb: Delete requires a WHERE clause to avoid full-table deletes")
	}
	result, err := q.client.Execute(ctx, q.buildDelete())
	if err != nil {
		return 0, err
	}
	var n int
	fmt.Sscanf(result.Message, "%d", &n)
	return n, nil
}

// ── SQL builders ──────────────────────────────────────────────────────────────

func (q *TableQuery) buildSelect() string {
	cols := "*"
	if len(q.cols) > 0 {
		cols = strings.Join(q.cols, ", ")
	}
	sql := fmt.Sprintf("SELECT %s FROM %s", cols, q.tableName)
	if q.where != "" {
		sql += " WHERE " + q.where
	}
	if q.orderBy != "" {
		sql += " ORDER BY " + q.orderBy
		if q.desc {
			sql += " DESC"
		}
	}
	return sql
}

func (q *TableQuery) buildInsert(ctx context.Context, row Row) (string, error) {
	// Try to fetch column order from server schema so values are ordered correctly.
	schema, err := q.client.DescribeTable(ctx, q.tableName)
	if err != nil {
		// Fallback: use map iteration order (non-deterministic but functional for small schemas).
		cols := make([]string, 0, len(row))
		vals := make([]string, 0, len(row))
		for k, v := range row {
			cols = append(cols, k)
			vals = append(vals, formatVal(v))
		}
		return fmt.Sprintf("INSERT INTO %s VALUES (%s)", q.tableName, strings.Join(vals, ", ")), nil
	}

	// Use server-defined column order.
	vals := make([]string, 0, len(schema.Columns))
	for _, col := range schema.Columns {
		v, ok := row[col.Name]
		if !ok {
			return "", fmt.Errorf("toydb: missing column %q in row", col.Name)
		}
		vals = append(vals, formatVal(v))
	}
	return fmt.Sprintf("INSERT INTO %s VALUES (%s)", q.tableName, strings.Join(vals, ", ")), nil
}

func (q *TableQuery) buildUpdate(updates Row) string {
	parts := make([]string, 0, len(updates))
	for k, v := range updates {
		parts = append(parts, fmt.Sprintf("%s = %s", k, formatVal(v)))
	}
	sql := fmt.Sprintf("UPDATE %s SET %s", q.tableName, strings.Join(parts, ", "))
	if q.where != "" {
		sql += " WHERE " + q.where
	}
	return sql
}

func (q *TableQuery) buildDelete() string {
	sql := fmt.Sprintf("DELETE FROM %s", q.tableName)
	if q.where != "" {
		sql += " WHERE " + q.where
	}
	return sql
}

// pkPlaceholder returns a best-guess PK column name ("id" by default).
// For a production library this would be fetched from schema.
func (q *TableQuery) pkPlaceholder() string { return "id" }

// formatVal formats a Go value as a SQL literal.
func formatVal(v any) string {
	if v == nil {
		return "NULL"
	}
	switch v := v.(type) {
	case string:
		escaped := strings.ReplaceAll(v, "'", "\\'")
		return "'" + escaped + "'"
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", v)
	}
}
