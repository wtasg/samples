package engine

import (
	"os"
	"reflect"
	"testing"

	"rdbms/internal/parser"
)

func TestExecutorE2E(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "toydb_executor_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ex, err := NewExecutor(tempDir)
	if err != nil {
		t.Fatalf("NewExecutor failed: %v", err)
	}
	defer ex.Close()

	// 1. CREATE TABLE
	createStmt := &parser.CreateTableStmt{
		Table: "users",
		Columns: []parser.ColumnDef{
			{Name: "id", Type: "INT"},
			{Name: "name", Type: "TEXT"},
			{Name: "score", Type: "INT"},
		},
	}
	res, err := ex.Execute(createStmt)
	if err != nil {
		t.Fatalf("CREATE TABLE failed: %v", err)
	}
	if res.Message != `Table "users" created.` {
		t.Errorf("Unexpected message: %q", res.Message)
	}

	// 2. INSERT
	inserts := []*parser.InsertStmt{
		{Table: "users", Values: []string{"1", "Alice", "100"}},
		{Table: "users", Values: []string{"2", "Bob", "80"}},
		{Table: "users", Values: []string{"3", "Carol", "90"}},
	}
	for _, ins := range inserts {
		res, err = ex.Execute(ins)
		if err != nil {
			t.Fatalf("INSERT failed: %v", err)
		}
		if res.Message != "1 row inserted." {
			t.Errorf("Unexpected message: %q", res.Message)
		}
	}

	// 3. SELECT ALL
	selAll := &parser.SelectStmt{
		Table: "users",
	}
	res, err = ex.Execute(selAll)
	if err != nil {
		t.Fatalf("SELECT ALL failed: %v", err)
	}
	if len(res.Rows) != 3 {
		t.Errorf("Expected 3 rows, got %d", len(res.Rows))
	}
	expectedCols := []string{"id", "name", "score"}
	if !reflect.DeepEqual(res.Columns, expectedCols) {
		t.Errorf("Columns mismatch. Expected %v, got %v", expectedCols, res.Columns)
	}

	// 4. SELECT WHERE ID = 2 (Point Lookup via Bloom + B+ Tree)
	selPoint := &parser.SelectStmt{
		Table: "users",
		Where: &parser.WhereExpr{
			Column: "id",
			Op:     "=",
			Value:  "2",
		},
	}
	res, err = ex.Execute(selPoint)
	if err != nil {
		t.Fatalf("SELECT Point failed: %v", err)
	}
	if len(res.Rows) != 1 || res.Rows[0]["name"] != "Bob" {
		t.Errorf("Unexpected result for point lookup: %+v", res.Rows)
	}

	// 5. SELECT WHERE ID BETWEEN 2 AND 3 (Range Scan via B+ Tree)
	selRange := &parser.SelectStmt{
		Table: "users",
		Where: &parser.WhereExpr{
			Column: "id",
			Op:     "BETWEEN",
			Value:  "2",
			Value2: "3",
		},
	}
	res, err = ex.Execute(selRange)
	if err != nil {
		t.Fatalf("SELECT Range failed: %v", err)
	}
	if len(res.Rows) != 2 {
		t.Errorf("Expected 2 rows from range, got %d", len(res.Rows))
	}

	// 6. SELECT WHERE NAME LIKE 'Al%' (Prefix Scan via Trie)
	selPrefix := &parser.SelectStmt{
		Table: "users",
		Where: &parser.WhereExpr{
			Column: "name",
			Op:     "LIKE",
			Value:  "Al%",
		},
	}
	res, err = ex.Execute(selPrefix)
	if err != nil {
		t.Fatalf("SELECT Prefix failed: %v", err)
	}
	if len(res.Rows) != 1 || res.Rows[0]["name"] != "Alice" {
		t.Errorf("Expected only Alice, got: %+v", res.Rows)
	}

	// 7. SELECT ORDER BY SCORE DESC (RBTree sorting)
	selOrder := &parser.SelectStmt{
		Table:   "users",
		OrderBy: "score",
		Desc:    true,
	}
	res, err = ex.Execute(selOrder)
	if err != nil {
		t.Fatalf("SELECT ORDER BY failed: %v", err)
	}
	if len(res.Rows) != 3 {
		t.Errorf("Expected 3 rows, got %d", len(res.Rows))
	}
	// Verify ordered correctly: Alice (100) -> Carol (90) -> Bob (80)
	if res.Rows[0]["name"] != "Alice" || res.Rows[1]["name"] != "Carol" || res.Rows[2]["name"] != "Bob" {
		t.Errorf("Order by sorting failed: %+v", res.Rows)
	}

	// 8. UPDATE
	updateStmt := &parser.UpdateStmt{
		Table: "users",
		Assignments: map[string]string{
			"score": "95",
		},
		Where: &parser.WhereExpr{
			Column: "id",
			Op:     "=",
			Value:  "2",
		},
	}
	res, err = ex.Execute(updateStmt)
	if err != nil {
		t.Fatalf("UPDATE failed: %v", err)
	}
	if res.Message != "1 row(s) updated." {
		t.Errorf("Unexpected message: %q", res.Message)
	}

	// Re-select Bob and check score
	res, _ = ex.Execute(selPoint)
	if int64(res.Rows[0]["score"].(float64)) != 95 {
		t.Errorf("Score update not reflected: %+v", res.Rows[0])
	}

	// 9. DELETE
	delStmt := &parser.DeleteStmt{
		Table: "users",
		Where: &parser.WhereExpr{
			Column: "id",
			Op:     "=",
			Value:  "1",
		},
	}
	res, err = ex.Execute(delStmt)
	if err != nil {
		t.Fatalf("DELETE failed: %v", err)
	}
	if res.Message != "1 row(s) deleted." {
		t.Errorf("Unexpected message: %q", res.Message)
	}

	// Verify row is gone
	selDeleted := &parser.SelectStmt{
		Table: "users",
		Where: &parser.WhereExpr{
			Column: "id",
			Op:     "=",
			Value:  "1",
		},
	}
	res, _ = ex.Execute(selDeleted)
	if len(res.Rows) != 0 {
		t.Errorf("Deleted row still returned: %+v", res.Rows)
	}

	// 10. DROP TABLE
	dropStmt := &parser.DropTableStmt{
		Table: "users",
	}
	res, err = ex.Execute(dropStmt)
	if err != nil {
		t.Fatalf("DROP TABLE failed: %v", err)
	}
	if res.Message != `Table "users" dropped.` {
		t.Errorf("Unexpected message: %q", res.Message)
	}
}
