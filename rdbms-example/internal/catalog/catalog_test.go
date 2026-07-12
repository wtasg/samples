package catalog

import (
	"os"
	"reflect"
	"testing"
)

func TestCatalogBasic(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "toydb_catalog_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 1. Load empty catalog
	cat, err := Load(tempDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// 2. CreateTable
	schema := &TableSchema{
		Name: "users",
		Columns: []Column{
			{Name: "id", Type: ColInt},
			{Name: "name", Type: ColText},
		},
	}
	err = cat.CreateTable(schema)
	if err != nil {
		t.Fatalf("CreateTable failed: %v", err)
	}

	// 3. GetTable
	gotSchema, err := cat.Get("users")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !reflect.DeepEqual(gotSchema, schema) {
		t.Errorf("Schema mismatch. Expected %+v, got %+v", schema, gotSchema)
	}

	// 4. Prefix search
	schema2 := &TableSchema{
		Name: "orders",
		Columns: []Column{
			{Name: "id", Type: ColInt},
			{Name: "amount", Type: ColFloat},
		},
	}
	err = cat.CreateTable(schema2)
	if err != nil {
		t.Fatalf("CreateTable 2 failed: %v", err)
	}

	tables := cat.TablesWithPrefix("use")
	if len(tables) != 1 || tables[0] != "users" {
		t.Errorf("Unexpected TablesWithPrefix('use'): %v", tables)
	}

	// 5. Reload and check persistence
	cat2, err := Load(tempDir)
	if err != nil {
		t.Fatalf("Load 2 failed: %v", err)
	}
	gotSchema2, err := cat2.Get("orders")
	if err != nil {
		t.Fatalf("Get 'orders' on reopened catalog failed: %v", err)
	}
	if gotSchema2.Name != "orders" {
		t.Errorf("Expected 'orders', got %q", gotSchema2.Name)
	}

	// 6. DropTable
	err = cat2.DropTable("users")
	if err != nil {
		t.Fatalf("DropTable failed: %v", err)
	}
	_, err = cat2.Get("users")
	if err == nil {
		t.Error("Expected error getting dropped table, got nil")
	}
}
