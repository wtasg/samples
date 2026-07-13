package parser

import (
	"testing"
)

func TestParse_CreateCollection(t *testing.T) {
	stmt, err := Parse(`db.createCollection("users")`)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	cs, ok := stmt.(*CreateCollectionStmt)
	if !ok {
		t.Fatalf("expected *CreateCollectionStmt, got %T", stmt)
	}
	if cs.Name != "users" {
		t.Errorf("Name = %q, want %q", cs.Name, "users")
	}
}

func TestParse_DropCollection(t *testing.T) {
	stmt, err := Parse(`db.dropCollection("users")`)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	ds, ok := stmt.(*DropCollectionStmt)
	if !ok {
		t.Fatalf("expected *DropCollectionStmt, got %T", stmt)
	}
	if ds.Name != "users" {
		t.Errorf("Name = %q, want %q", ds.Name, "users")
	}
}

func TestParse_Insert(t *testing.T) {
	stmt, err := Parse(`db.products.insert({"name": "Apple", "price": 100})`)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	is, ok := stmt.(*InsertStmt)
	if !ok {
		t.Fatalf("expected *InsertStmt, got %T", stmt)
	}
	if is.Collection != "products" {
		t.Errorf("Collection = %q, want %q", is.Collection, "products")
	}
	if is.Document["name"] != "Apple" {
		t.Errorf("Document[name] = %v, want Apple", is.Document["name"])
	}
}

func TestParse_FindNoFilter(t *testing.T) {
	stmt, err := Parse(`db.users.find({})`)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	fs, ok := stmt.(*FindStmt)
	if !ok {
		t.Fatalf("expected *FindStmt, got %T", stmt)
	}
	if fs.Collection != "users" {
		t.Errorf("Collection = %q, want %q", fs.Collection, "users")
	}
	if fs.Filter != nil {
		t.Errorf("Filter = %v, want nil (match all)", fs.Filter)
	}
}

func TestParse_FindWithFilter(t *testing.T) {
	stmt, err := Parse(`db.products.find({"price": {"$gt": 50}})`)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	fs, ok := stmt.(*FindStmt)
	if !ok {
		t.Fatalf("expected *FindStmt, got %T", stmt)
	}
	if fs.Collection != "products" {
		t.Errorf("Collection = %q", fs.Collection)
	}
	if fs.Filter == nil {
		t.Fatal("expected non-nil filter")
	}
}

func TestParse_FindWithSort(t *testing.T) {
	stmt, err := Parse(`db.products.find({}).sort({"price": -1})`)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	fs, ok := stmt.(*FindStmt)
	if !ok {
		t.Fatalf("expected *FindStmt, got %T", stmt)
	}
	if fs.SortField != "price" {
		t.Errorf("SortField = %q, want %q", fs.SortField, "price")
	}
	if fs.SortOrder != -1 {
		t.Errorf("SortOrder = %d, want -1", fs.SortOrder)
	}
}

func TestParse_Update(t *testing.T) {
	stmt, err := Parse(`db.users.update({"_id": "abc"}, {"$set": {"age": 31}})`)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	us, ok := stmt.(*UpdateStmt)
	if !ok {
		t.Fatalf("expected *UpdateStmt, got %T", stmt)
	}
	if us.Collection != "users" {
		t.Errorf("Collection = %q", us.Collection)
	}
	if us.Filter["_id"] != "abc" {
		t.Errorf("Filter[_id] = %v, want 'abc'", us.Filter["_id"])
	}
}

func TestParse_Delete(t *testing.T) {
	stmt, err := Parse(`db.users.delete({"_id": "abc"})`)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	ds, ok := stmt.(*DeleteStmt)
	if !ok {
		t.Fatalf("expected *DeleteStmt, got %T", stmt)
	}
	if ds.Collection != "users" {
		t.Errorf("Collection = %q", ds.Collection)
	}
}

func TestParse_EmptyInput(t *testing.T) {
	stmt, err := Parse("")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if stmt != nil {
		t.Errorf("expected nil statement for empty input")
	}
}

func TestParse_InvalidPrefix(t *testing.T) {
	_, err := Parse("SELECT * FROM users")
	if err == nil {
		t.Error("expected error for non-db prefix")
	}
}

func TestParse_Semicolon(t *testing.T) {
	stmt, err := Parse(`db.createCollection("test");`)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if _, ok := stmt.(*CreateCollectionStmt); !ok {
		t.Fatalf("expected *CreateCollectionStmt, got %T", stmt)
	}
}
