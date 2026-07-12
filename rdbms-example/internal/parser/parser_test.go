package parser

import (
	"reflect"
	"testing"
)

func TestParseCreate(t *testing.T) {
	sql := "CREATE TABLE users (id INT, name TEXT, salary FLOAT, active BOOL);"
	stmt, err := Parse(sql)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	create, ok := stmt.(*CreateTableStmt)
	if !ok {
		t.Fatalf("Expected CreateTableStmt, got %T", stmt)
	}

	if create.Table != "users" {
		t.Errorf("Expected table 'users', got %q", create.Table)
	}

	expectedCols := []ColumnDef{
		{Name: "id", Type: "INT"},
		{Name: "name", Type: "TEXT"},
		{Name: "salary", Type: "FLOAT"},
		{Name: "active", Type: "BOOL"},
	}

	if !reflect.DeepEqual(create.Columns, expectedCols) {
		t.Errorf("Columns mismatch. Expected %+v, got %+v", expectedCols, create.Columns)
	}
}

func TestParseInsert(t *testing.T) {
	sql := "INSERT INTO users VALUES (1, 'Alice', 95000.5, true);"
	stmt, err := Parse(sql)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	insert, ok := stmt.(*InsertStmt)
	if !ok {
		t.Fatalf("Expected InsertStmt, got %T", stmt)
	}

	if insert.Table != "users" {
		t.Errorf("Expected table 'users', got %q", insert.Table)
	}

	expectedVals := []string{"1", "Alice", "95000.5", "true"}
	if !reflect.DeepEqual(insert.Values, expectedVals) {
		t.Errorf("Values mismatch. Expected %v, got %v", expectedVals, insert.Values)
	}
}

func TestParseSelect(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected *SelectStmt
	}{
		{
			name: "select all",
			sql:  "SELECT * FROM users",
			expected: &SelectStmt{
				Table: "users",
			},
		},
		{
			name: "select cols",
			sql:  "SELECT id, name FROM users",
			expected: &SelectStmt{
				Table:   "users",
				Columns: []string{"id", "name"},
			},
		},
		{
			name: "select with where and order by",
			sql:  "SELECT * FROM users WHERE salary > 50000 ORDER BY id DESC",
			expected: &SelectStmt{
				Table: "users",
				Where: &WhereExpr{
					Column: "salary",
					Op:     ">",
					Value:  "50000",
				},
				OrderBy: "id",
				Desc:    true,
			},
		},
		{
			name: "select between",
			sql:  "SELECT name FROM users WHERE id BETWEEN 10 AND 20",
			expected: &SelectStmt{
				Table:   "users",
				Columns: []string{"name"},
				Where: &WhereExpr{
					Column: "id",
					Op:     "BETWEEN",
					Value:  "10",
					Value2: "20",
				},
			},
		},
		{
			name: "select like",
			sql:  "SELECT * FROM users WHERE name LIKE 'A%'",
			expected: &SelectStmt{
				Table: "users",
				Where: &WhereExpr{
					Column: "name",
					Op:     "LIKE",
					Value:  "A%",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stmt, err := Parse(tc.sql)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}
			sel, ok := stmt.(*SelectStmt)
			if !ok {
				t.Fatalf("Expected SelectStmt, got %T", stmt)
			}
			if !reflect.DeepEqual(sel, tc.expected) {
				t.Errorf("SelectStmt mismatch. Expected %+v, got %+v", tc.expected, sel)
			}
		})
	}
}

func TestParseUpdate(t *testing.T) {
	sql := "UPDATE users SET salary = 100000, active = false WHERE id = 1;"
	stmt, err := Parse(sql)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	update, ok := stmt.(*UpdateStmt)
	if !ok {
		t.Fatalf("Expected UpdateStmt, got %T", stmt)
	}

	if update.Table != "users" {
		t.Errorf("Expected table 'users', got %q", update.Table)
	}

	expectedAssigns := map[string]string{
		"salary": "100000",
		"active": "false",
	}

	if !reflect.DeepEqual(update.Assignments, expectedAssigns) {
		t.Errorf("Assignments mismatch. Expected %+v, got %+v", expectedAssigns, update.Assignments)
	}

	expectedWhere := &WhereExpr{
		Column: "id",
		Op:     "=",
		Value:  "1",
	}

	if !reflect.DeepEqual(update.Where, expectedWhere) {
		t.Errorf("Where mismatch. Expected %+v, got %+v", expectedWhere, update.Where)
	}
}

func TestParseDelete(t *testing.T) {
	sql := "DELETE FROM users WHERE active = false"
	stmt, err := Parse(sql)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	del, ok := stmt.(*DeleteStmt)
	if !ok {
		t.Fatalf("Expected DeleteStmt, got %T", stmt)
	}

	if del.Table != "users" {
		t.Errorf("Expected table 'users', got %q", del.Table)
	}

	expectedWhere := &WhereExpr{
		Column: "active",
		Op:     "=",
		Value:  "false",
	}

	if !reflect.DeepEqual(del.Where, expectedWhere) {
		t.Errorf("Where mismatch. Expected %+v, got %+v", expectedWhere, del.Where)
	}
}

func TestParseDrop(t *testing.T) {
	sql := "DROP TABLE users;"
	stmt, err := Parse(sql)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	drop, ok := stmt.(*DropTableStmt)
	if !ok {
		t.Fatalf("Expected DropTableStmt, got %T", stmt)
	}

	if drop.Table != "users" {
		t.Errorf("Expected table 'users', got %q", drop.Table)
	}
}
