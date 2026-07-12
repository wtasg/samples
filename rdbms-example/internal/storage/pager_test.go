package storage

import (
	"os"
	"testing"
)

func TestPagerBasic(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "toydb_pager_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 1. Open
	p, err := Open(tempDir, "users")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer p.Close()

	// 2. Write
	row := Row{"name": "Alice", "age": 30.0}
	rid, err := p.Write(row)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// 3. Read
	r, err := p.Read(rid)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if r["name"] != "Alice" || r["age"] != 30.0 {
		t.Errorf("Unexpected row content: %+v", r)
	}

	// 4. Update (in-place pad / same length or shorter)
	err = p.Update(rid, Row{"name": "Bob", "age": 30.0})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	r, err = p.Read(rid)
	if err != nil {
		t.Fatalf("Read after update failed: %v", err)
	}
	if r["name"] != "Bob" {
		t.Errorf("Expected 'Bob', got %v", r["name"])
	}

	// 5. Update (longer, forces append)
	err = p.Update(rid, Row{"name": "Christopher Columbus", "age": 40.0})
	if err != nil {
		t.Fatalf("Long Update failed: %v", err)
	}
	r, err = p.Read(rid)
	if err != nil {
		t.Fatalf("Read after long update failed: %v", err)
	}
	if r["name"] != "Christopher Columbus" || r["age"] != 40.0 {
		t.Errorf("Unexpected content after append: %+v", r)
	}

	// 6. ScanAll
	rows, err := p.ScanAll()
	if err != nil {
		t.Fatalf("ScanAll failed: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("Expected 1 row, got %d", len(rows))
	}

	// 7. SoftDelete
	err = p.SoftDelete(rid)
	if err != nil {
		t.Fatalf("SoftDelete failed: %v", err)
	}
	if !p.IsDeleted(rid) {
		t.Errorf("Row should be deleted")
	}

	rows, err = p.ScanAll()
	if err != nil {
		t.Fatalf("ScanAll after delete failed: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("Expected 0 rows after delete, got %d", len(rows))
	}
}

func TestPagerPersistence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "toydb_pager_persist_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tableName := "items"
	// Write row to first pager instance
	p1, err := Open(tempDir, tableName)
	if err != nil {
		t.Fatalf("Open 1 failed: %v", err)
	}
	_, err = p1.Write(Row{"item": "Sword", "power": 100.0})
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	p1.Close()

	// Reopen with a new pager instance on the same file
	p2, err := Open(tempDir, tableName)
	if err != nil {
		t.Fatalf("Open 2 failed: %v", err)
	}
	defer p2.Close()

	rows, err := p2.ScanAll()
	if err != nil {
		t.Fatalf("ScanAll on reopened pager failed: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("Expected 1 row, got %d", len(rows))
	}
	if rows[0]["item"] != "Sword" || rows[0]["power"] != 100.0 {
		t.Errorf("Unexpected content: %+v", rows[0])
	}
}
