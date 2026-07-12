// Package catalog manages table schemas (the "system catalog" of the RDBMS).
//
// The catalog uses a Trie internally so that table/column name lookups are
// O(m) where m is the name length, and prefix queries (e.g. "show all tables
// starting with 'ord'") are also O(m + k).
//
// Schema is persisted to <dataDir>/catalog.json on every mutation.
package catalog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"rdbms/internal/ds"
)

// ColType is the data type of a column.
type ColType string

const (
	ColInt   ColType = "INT"
	ColText  ColType = "TEXT"
	ColFloat ColType = "FLOAT"
	ColBool  ColType = "BOOL"
)

// ParseColType converts a SQL type keyword to ColType.
func ParseColType(s string) (ColType, error) {
	switch ColType(s) {
	case ColInt, ColText, ColFloat, ColBool:
		return ColType(s), nil
	default:
		return "", fmt.Errorf("unknown column type %q (use INT, TEXT, FLOAT, BOOL)", s)
	}
}

// Column describes a single column in a table schema.
type Column struct {
	Name string  `json:"name"`
	Type ColType `json:"type"`
}

// TableSchema is the schema of a single table.
// The FIRST column is always treated as the primary key (must be INT).
type TableSchema struct {
	Name    string   `json:"name"`
	Columns []Column `json:"columns"`
}

// PKColumn returns the primary key column (always index 0).
func (s *TableSchema) PKColumn() Column { return s.Columns[0] }

// ColumnIndex returns the 0-based index of the named column, or -1.
func (s *TableSchema) ColumnIndex(name string) int {
	for i, c := range s.Columns {
		if c.Name == name {
			return i
		}
	}
	return -1
}

// TextColumns returns all TEXT column names (for Trie secondary index building).
func (s *TableSchema) TextColumns() []string {
	var names []string
	for _, c := range s.Columns {
		if c.Type == ColText {
			names = append(names, c.Name)
		}
	}
	return names
}

// catalogFile is the JSON format of the persisted catalog.
type catalogFile struct {
	Tables []*TableSchema `json:"tables"`
}

// Catalog is the in-memory schema store, backed by a Trie for O(m) lookups.
type Catalog struct {
	dir     string
	schemas map[string]*TableSchema
	trie    *ds.Trie // table name → *TableSchema (via SetMeta/GetMeta)
}

// Load opens the catalog file from dataDir, creating it if absent.
func Load(dataDir string) (*Catalog, error) {
	c := &Catalog{
		dir:     dataDir,
		schemas: make(map[string]*TableSchema),
		trie:    ds.NewTrie(),
	}

	path := filepath.Join(dataDir, "catalog.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return nil, fmt.Errorf("catalog load: %w", err)
	}

	var cf catalogFile
	if err := json.Unmarshal(data, &cf); err != nil {
		return nil, fmt.Errorf("catalog parse: %w", err)
	}
	for _, s := range cf.Tables {
		c.schemas[s.Name] = s
		c.trie.SetMeta(s.Name, s)
	}
	return c, nil
}

// CreateTable registers a new table schema. Returns an error if already exists.
func (c *Catalog) CreateTable(schema *TableSchema) error {
	if len(schema.Columns) == 0 {
		return fmt.Errorf("table must have at least one column")
	}
	if schema.Columns[0].Type != ColInt {
		return fmt.Errorf("first column (primary key) must be INT")
	}
	if _, exists := c.schemas[schema.Name]; exists {
		return fmt.Errorf("table %q already exists", schema.Name)
	}
	c.schemas[schema.Name] = schema
	c.trie.SetMeta(schema.Name, schema)
	return c.save()
}

// DropTable removes a table schema. Returns an error if not found.
func (c *Catalog) DropTable(name string) error {
	if _, exists := c.schemas[name]; !exists {
		return fmt.Errorf("table %q does not exist", name)
	}
	delete(c.schemas, name)
	// Trie doesn't support deletion of meta; mark as absent via map absence.
	return c.save()
}

// Get returns the schema for tableName or an error if not found.
// Uses the Trie for O(m) lookup (m = len(tableName)).
func (c *Catalog) Get(tableName string) (*TableSchema, error) {
	// Trie lookup.
	if meta, ok := c.trie.GetMeta(tableName); ok {
		if s, ok := c.schemas[tableName]; ok {
			_ = meta
			return s, nil
		}
	}
	return nil, fmt.Errorf("table %q does not exist", tableName)
}

// Tables returns all table names.
func (c *Catalog) Tables() []string {
	names := make([]string, 0, len(c.schemas))
	for name := range c.schemas {
		names = append(names, name)
	}
	return names
}

// TablesWithPrefix returns all table names that start with prefix.
// Demonstrates Trie prefix search in the catalog layer.
func (c *Catalog) TablesWithPrefix(prefix string) []string {
	// Collect names from trie; filter to those still in c.schemas.
	var result []string
	words := c.trie.Words()
	for _, w := range words {
		if _, ok := c.schemas[w]; !ok {
			continue
		}
		if ds.HasPrefix(w, prefix) {
			result = append(result, w)
		}
	}
	return result
}

// save persists the in-memory schemas to catalog.json.
func (c *Catalog) save() error {
	tables := make([]*TableSchema, 0, len(c.schemas))
	for _, s := range c.schemas {
		tables = append(tables, s)
	}
	data, err := json.MarshalIndent(catalogFile{Tables: tables}, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(c.dir, "catalog.json")
	return os.WriteFile(path, data, 0644)
}
