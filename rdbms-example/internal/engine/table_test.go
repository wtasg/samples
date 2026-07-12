package engine

import (
	"os"
	"testing"

	"rdbms/internal/catalog"
)

func TestTableOperations(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "toydb_engine_table_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	schema := &catalog.TableSchema{
		Name: "users",
		Columns: []catalog.Column{
			{Name: "id", Type: catalog.ColInt},
			{Name: "name", Type: catalog.ColText},
			{Name: "score", Type: catalog.ColInt},
		},
	}

	// 1. Open table
	tbl, err := openTable(schema, tempDir)
	if err != nil {
		t.Fatalf("openTable failed: %v", err)
	}
	defer tbl.close()

	// 2. Insert
	err = tbl.Insert([]string{"1", "Alice", "100"})
	if err != nil {
		t.Errorf("Insert Alice failed: %v", err)
	}
	err = tbl.Insert([]string{"2", "Bob", "80"})
	if err != nil {
		t.Errorf("Insert Bob failed: %v", err)
	}
	err = tbl.Insert([]string{"3", "Alonzo", "90"})
	if err != nil {
		t.Errorf("Insert Alonzo failed: %v", err)
	}

	// Duplicate insert should fail
	err = tbl.Insert([]string{"1", "Duplicate", "10"})
	if err == nil {
		t.Error("Expected error inserting duplicate PK, got nil")
	}

	// 3. GetByPK (Bloom Filter + B+ Tree + Pager path)
	row, err := tbl.GetByPK(1)
	if err != nil {
		t.Fatalf("GetByPK failed: %v", err)
	}
	// Note: float64 is returned since it is unmarshaled from JSON
	if row["name"] != "Alice" || int64(row["score"].(float64)) != 100 {
		t.Errorf("Unexpected row content: %+v", row)
	}

	// Non-existent PK should fail
	_, err = tbl.GetByPK(999)
	if err == nil {
		t.Error("Expected error getting non-existent PK, got nil")
	}

	// 4. RangeScan (B+ Tree range scan)
	rows, err := tbl.RangeScan(1, 2)
	if err != nil {
		t.Fatalf("RangeScan failed: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("Expected 2 rows from range scan, got %d", len(rows))
	}

	// 5. PrefixScan (Trie Index path)
	rows, err = tbl.PrefixScan("name", "Al")
	if err != nil {
		t.Fatalf("PrefixScan failed: %v", err)
	}
	// Should match Alice and Alonzo
	if len(rows) != 2 {
		t.Errorf("Expected 2 rows from PrefixScan('Al'), got %d", len(rows))
	}

	// 6. SubstringScan (Rabin-Karp path)
	rows, err = tbl.SubstringScan("name", "onz")
	if err != nil {
		t.Fatalf("SubstringScan failed: %v", err)
	}
	// Should match Alonzo
	if len(rows) != 1 || rows[0]["name"] != "Alonzo" {
		t.Errorf("Unexpected result from SubstringScan: %+v", rows)
	}

	// 7. Update
	count, err := tbl.Update(func(r Row) bool {
		return r["name"] == "Bob"
	}, map[string]string{"score": "85"}, schema)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 row updated, got %d", count)
	}
	row, _ = tbl.GetByPK(2)
	if int64(row["score"].(float64)) != 85 {
		t.Errorf("Score not updated: %+v", row)
	}

	// 8. Delete
	count, err = tbl.Delete(func(r Row) bool {
		return r["name"] == "Alonzo"
	})
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 row deleted, got %d", count)
	}
	_, err = tbl.GetByPK(3)
	if err == nil {
		t.Error("Expected error getting deleted row, got nil")
	}
}

func TestOrderBy(t *testing.T) {
	rows := []Row{
		{"id": 1.0, "name": "Alice", "score": 100.0},
		{"id": 2.0, "name": "Bob", "score": 80.0},
		{"id": 3.0, "name": "Carol", "score": 90.0},
	}

	sorted := OrderByInt(rows, "score", false) // asc
	if len(sorted) != 3 || int64(sorted[0]["score"].(float64)) != 80 || int64(sorted[2]["score"].(float64)) != 100 {
		t.Errorf("Ascending sort failed: %+v", sorted)
	}

	sortedDesc := OrderByInt(rows, "score", true) // desc
	if len(sortedDesc) != 3 || int64(sortedDesc[0]["score"].(float64)) != 100 || int64(sortedDesc[2]["score"].(float64)) != 80 {
		t.Errorf("Descending sort failed: %+v", sortedDesc)
	}
}
