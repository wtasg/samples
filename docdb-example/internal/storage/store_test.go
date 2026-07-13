package storage

import (
	"testing"
)

func TestStore_WriteAndRead(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir, "test")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	doc := Doc{"_id": "doc1", "name": "Alice", "age": 30.0}
	if err := s.Write(doc); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, err := s.Read("doc1")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got["name"] != "Alice" {
		t.Errorf("name = %v, want Alice", got["name"])
	}
}

func TestStore_SoftDelete(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir, "test")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	s.Write(Doc{"_id": "d1", "v": "1"})
	s.Write(Doc{"_id": "d2", "v": "2"})

	if err := s.SoftDelete("d1"); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	if _, err := s.Read("d1"); err == nil {
		t.Error("expected error reading deleted doc")
	}

	got, err := s.Read("d2")
	if err != nil {
		t.Fatalf("Read d2: %v", err)
	}
	if got["v"] != "2" {
		t.Errorf("d2.v = %v, want '2'", got["v"])
	}
}

func TestStore_Update(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir, "test")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	s.Write(Doc{"_id": "d1", "name": "Alice", "age": 30.0})

	if err := s.Update("d1", Doc{"name": "Alice", "age": 31.0}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := s.Read("d1")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got["age"] != 31.0 {
		t.Errorf("age = %v, want 31", got["age"])
	}
}

func TestStore_ScanAll(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir, "test")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	s.Write(Doc{"_id": "d1", "v": "1"})
	s.Write(Doc{"_id": "d2", "v": "2"})
	s.Write(Doc{"_id": "d3", "v": "3"})
	s.SoftDelete("d2")

	docs, err := s.ScanAll()
	if err != nil {
		t.Fatalf("ScanAll: %v", err)
	}
	if len(docs) != 2 {
		t.Errorf("ScanAll returned %d docs, want 2", len(docs))
	}
}

func TestStore_DocIDs(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir, "test")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	s.Write(Doc{"_id": "a"})
	s.Write(Doc{"_id": "b"})

	ids := s.DocIDs()
	if len(ids) != 2 {
		t.Errorf("DocIDs len = %d, want 2", len(ids))
	}
}

func TestStore_Persistence(t *testing.T) {
	dir := t.TempDir()

	s1, _ := Open(dir, "persist")
	s1.Write(Doc{"_id": "p1", "name": "Test"})
	s1.Close()

	s2, err := Open(dir, "persist")
	if err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	defer s2.Close()

	doc, err := s2.Read("p1")
	if err != nil {
		t.Fatalf("Read after reopen: %v", err)
	}
	if doc["name"] != "Test" {
		t.Errorf("name = %v, want Test", doc["name"])
	}
}
