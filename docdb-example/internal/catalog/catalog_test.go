package catalog

import (
	"os"
	"testing"
)

func TestCatalog_CreateAndGet(t *testing.T) {
	dir := t.TempDir()
	c, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if err := c.CreateCollection("users"); err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}

	m, err := c.Get("users")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if m.Name != "users" {
		t.Errorf("Name = %q, want %q", m.Name, "users")
	}
}

func TestCatalog_DuplicateCreate(t *testing.T) {
	dir := t.TempDir()
	c, _ := Load(dir)

	c.CreateCollection("test")
	err := c.CreateCollection("test")
	if err == nil {
		t.Error("expected error for duplicate create")
	}
}

func TestCatalog_DropCollection(t *testing.T) {
	dir := t.TempDir()
	c, _ := Load(dir)

	c.CreateCollection("temp")
	if err := c.DropCollection("temp"); err != nil {
		t.Fatalf("DropCollection: %v", err)
	}
	if c.Has("temp") {
		t.Error("collection should not exist after drop")
	}
}

func TestCatalog_DropNonexistent(t *testing.T) {
	dir := t.TempDir()
	c, _ := Load(dir)

	if err := c.DropCollection("nope"); err == nil {
		t.Error("expected error for dropping non-existent collection")
	}
}

func TestCatalog_Collections(t *testing.T) {
	dir := t.TempDir()
	c, _ := Load(dir)

	c.CreateCollection("a")
	c.CreateCollection("b")

	cols := c.Collections()
	if len(cols) != 2 {
		t.Errorf("Collections() len = %d, want 2", len(cols))
	}
}

func TestCatalog_Persistence(t *testing.T) {
	dir := t.TempDir()
	c1, _ := Load(dir)
	c1.CreateCollection("persisted")

	// Reload from disk.
	c2, err := Load(dir)
	if err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if !c2.Has("persisted") {
		t.Error("collection should persist across reload")
	}
}

func TestCatalog_LoadEmpty(t *testing.T) {
	dir := t.TempDir()
	os.Remove(dir + "/catalog.json") // ensure it doesn't exist

	c, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(c.Collections()) != 0 {
		t.Error("expected empty catalog")
	}
}
