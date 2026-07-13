package engine

import (
	"testing"

	"docdb/internal/parser"
)

func TestExecutor_CreateAndDrop(t *testing.T) {
	dir := t.TempDir()
	ex, err := NewExecutor(dir)
	if err != nil {
		t.Fatalf("NewExecutor: %v", err)
	}
	defer ex.Close()

	// Create.
	stmt, _ := parser.Parse(`db.createCollection("test")`)
	result, err := ex.Execute(stmt)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if result.Message == "" {
		t.Error("expected non-empty message")
	}

	names := ex.CollectionNames()
	if len(names) != 1 || names[0] != "test" {
		t.Errorf("CollectionNames = %v, want [test]", names)
	}

	// Drop.
	stmt, _ = parser.Parse(`db.dropCollection("test")`)
	_, err = ex.Execute(stmt)
	if err != nil {
		t.Fatalf("Drop: %v", err)
	}

	if len(ex.CollectionNames()) != 0 {
		t.Error("expected empty collections after drop")
	}
}

func TestExecutor_InsertAndFind(t *testing.T) {
	dir := t.TempDir()
	ex, err := NewExecutor(dir)
	if err != nil {
		t.Fatalf("NewExecutor: %v", err)
	}
	defer ex.Close()

	// Create collection.
	stmt, _ := parser.Parse(`db.createCollection("users")`)
	ex.Execute(stmt)

	// Insert.
	stmt, _ = parser.Parse(`db.users.insert({"_id": "u1", "name": "Alice", "age": 30})`)
	_, err = ex.Execute(stmt)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}

	// Find all.
	stmt, _ = parser.Parse(`db.users.find({})`)
	result, err := ex.Execute(stmt)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(result.Docs) != 1 {
		t.Errorf("expected 1 doc, got %d", len(result.Docs))
	}
}

func TestExecutor_Update(t *testing.T) {
	dir := t.TempDir()
	ex, err := NewExecutor(dir)
	if err != nil {
		t.Fatalf("NewExecutor: %v", err)
	}
	defer ex.Close()

	stmt, _ := parser.Parse(`db.createCollection("items")`)
	ex.Execute(stmt)

	stmt, _ = parser.Parse(`db.items.insert({"_id": "i1", "name": "Widget", "price": 50})`)
	ex.Execute(stmt)

	stmt, _ = parser.Parse(`db.items.update({"_id": "i1"}, {"$set": {"price": 75}})`)
	result, err := ex.Execute(stmt)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if result.Message != "1 document(s) updated." {
		t.Errorf("Message = %q", result.Message)
	}
}

func TestExecutor_Delete(t *testing.T) {
	dir := t.TempDir()
	ex, err := NewExecutor(dir)
	if err != nil {
		t.Fatalf("NewExecutor: %v", err)
	}
	defer ex.Close()

	stmt, _ := parser.Parse(`db.createCollection("temp")`)
	ex.Execute(stmt)

	stmt, _ = parser.Parse(`db.temp.insert({"_id": "t1", "v": "1"})`)
	ex.Execute(stmt)

	stmt, _ = parser.Parse(`db.temp.delete({"_id": "t1"})`)
	result, err := ex.Execute(stmt)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if result.Message != "1 document(s) deleted." {
		t.Errorf("Message = %q", result.Message)
	}
}

func TestExecutor_FindWithSort(t *testing.T) {
	dir := t.TempDir()
	ex, err := NewExecutor(dir)
	if err != nil {
		t.Fatalf("NewExecutor: %v", err)
	}
	defer ex.Close()

	stmt, _ := parser.Parse(`db.createCollection("products")`)
	ex.Execute(stmt)

	stmt, _ = parser.Parse(`db.products.insert({"_id": "p1", "name": "C", "price": 300})`)
	ex.Execute(stmt)
	stmt, _ = parser.Parse(`db.products.insert({"_id": "p2", "name": "A", "price": 100})`)
	ex.Execute(stmt)
	stmt, _ = parser.Parse(`db.products.insert({"_id": "p3", "name": "B", "price": 200})`)
	ex.Execute(stmt)

	stmt, _ = parser.Parse(`db.products.find({}).sort({"price": 1})`)
	result, err := ex.Execute(stmt)
	if err != nil {
		t.Fatalf("Find+Sort: %v", err)
	}
	if len(result.Docs) != 3 {
		t.Fatalf("expected 3 docs, got %d", len(result.Docs))
	}
	if result.Docs[0]["name"] != "A" {
		t.Errorf("first doc should be A (lowest price), got %v", result.Docs[0]["name"])
	}
}
