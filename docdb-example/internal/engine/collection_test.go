package engine

import (
	"testing"

	"docdb/internal/catalog"
)

func TestCollection_InsertAndGet(t *testing.T) {
	dir := t.TempDir()
	c := newTestCollection(t, dir, "test")
	defer c.close()

	id, err := c.Insert(Doc{"name": "Alice", "age": 30.0})
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty ID")
	}

	doc, err := c.Get(id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if doc["name"] != "Alice" {
		t.Errorf("name = %v, want Alice", doc["name"])
	}
}

func TestCollection_InsertWithID(t *testing.T) {
	dir := t.TempDir()
	c := newTestCollection(t, dir, "test")
	defer c.close()

	id, err := c.Insert(Doc{"_id": "custom-id", "v": "1"})
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if id != "custom-id" {
		t.Errorf("id = %q, want %q", id, "custom-id")
	}
}

func TestCollection_DuplicateID(t *testing.T) {
	dir := t.TempDir()
	c := newTestCollection(t, dir, "test")
	defer c.close()

	c.Insert(Doc{"_id": "d1", "v": "1"})
	_, err := c.Insert(Doc{"_id": "d1", "v": "2"})
	if err == nil {
		t.Error("expected error for duplicate _id")
	}
}

func TestCollection_FindAll(t *testing.T) {
	dir := t.TempDir()
	c := newTestCollection(t, dir, "test")
	defer c.close()

	c.Insert(Doc{"_id": "d1", "v": "1"})
	c.Insert(Doc{"_id": "d2", "v": "2"})

	docs, err := c.Find(nil)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(docs) != 2 {
		t.Errorf("Find(nil) returned %d docs, want 2", len(docs))
	}
}

func TestCollection_FindByID(t *testing.T) {
	dir := t.TempDir()
	c := newTestCollection(t, dir, "test")
	defer c.close()

	c.Insert(Doc{"_id": "target", "name": "Bob"})
	c.Insert(Doc{"_id": "other", "name": "Eve"})

	docs, err := c.Find(map[string]any{"_id": "target"})
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(docs) != 1 || docs[0]["name"] != "Bob" {
		t.Errorf("expected Bob, got %v", docs)
	}
}

func TestCollection_FindByField(t *testing.T) {
	dir := t.TempDir()
	c := newTestCollection(t, dir, "test")
	defer c.close()

	c.Insert(Doc{"_id": "d1", "category": "electronics"})
	c.Insert(Doc{"_id": "d2", "category": "clothing"})
	c.Insert(Doc{"_id": "d3", "category": "electronics"})

	docs, err := c.Find(map[string]any{"category": "electronics"})
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(docs) != 2 {
		t.Errorf("expected 2 docs, got %d", len(docs))
	}
}

func TestCollection_FindWithGT(t *testing.T) {
	dir := t.TempDir()
	c := newTestCollection(t, dir, "test")
	defer c.close()

	c.Insert(Doc{"_id": "d1", "price": 50.0})
	c.Insert(Doc{"_id": "d2", "price": 150.0})
	c.Insert(Doc{"_id": "d3", "price": 250.0})

	docs, err := c.Find(map[string]any{"price": map[string]any{"$gt": 100.0}})
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(docs) != 2 {
		t.Errorf("expected 2 docs with price > 100, got %d", len(docs))
	}
}

func TestCollection_FindWithPrefix(t *testing.T) {
	dir := t.TempDir()
	c := newTestCollection(t, dir, "test")
	defer c.close()

	c.Insert(Doc{"_id": "d1", "name": "Alice"})
	c.Insert(Doc{"_id": "d2", "name": "Albert"})
	c.Insert(Doc{"_id": "d3", "name": "Bob"})

	docs, err := c.Find(map[string]any{"name": map[string]any{"$prefix": "Al"}})
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(docs) != 2 {
		t.Errorf("expected 2 docs with prefix Al, got %d", len(docs))
	}
}

func TestCollection_Update(t *testing.T) {
	dir := t.TempDir()
	c := newTestCollection(t, dir, "test")
	defer c.close()

	c.Insert(Doc{"_id": "d1", "name": "Alice", "age": 30.0})

	n, err := c.Update(
		map[string]any{"_id": "d1"},
		map[string]any{"$set": map[string]any{"age": 31.0}},
	)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if n != 1 {
		t.Errorf("updated %d, want 1", n)
	}

	doc, _ := c.Get("d1")
	if doc["age"] != 31.0 {
		t.Errorf("age = %v, want 31", doc["age"])
	}
}

func TestCollection_Delete(t *testing.T) {
	dir := t.TempDir()
	c := newTestCollection(t, dir, "test")
	defer c.close()

	c.Insert(Doc{"_id": "d1", "v": "1"})
	c.Insert(Doc{"_id": "d2", "v": "2"})

	n, err := c.Delete(map[string]any{"_id": "d1"})
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if n != 1 {
		t.Errorf("deleted %d, want 1", n)
	}

	docs, _ := c.Find(nil)
	if len(docs) != 1 {
		t.Errorf("expected 1 doc after delete, got %d", len(docs))
	}
}

func TestSortDocs(t *testing.T) {
	docs := []Doc{
		{"name": "Charlie", "age": 35.0},
		{"name": "Alice", "age": 25.0},
		{"name": "Bob", "age": 30.0},
	}

	sorted := SortDocs(docs, "age", 1)
	if sorted[0]["name"] != "Alice" || sorted[2]["name"] != "Charlie" {
		t.Errorf("ascending sort failed: %v", sorted)
	}

	sortedDesc := SortDocs(docs, "age", -1)
	if sortedDesc[0]["name"] != "Charlie" || sortedDesc[2]["name"] != "Alice" {
		t.Errorf("descending sort failed: %v", sortedDesc)
	}
}

// newTestCollection creates a test collection in a temporary directory.
func newTestCollection(t *testing.T, dir, name string) *Collection {
	t.Helper()
	meta := &catalog.CollectionMeta{Name: name}
	c, err := openCollection(meta, dir)
	if err != nil {
		t.Fatalf("openCollection: %v", err)
	}
	return c
}
