// Package catalog manages collection metadata (the "system catalog" of DocDB).
//
// The catalog stores collection names and configuration. It is persisted to
// <dataDir>/catalog.json on every mutation.
package catalog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// CollectionMeta describes a single collection.
type CollectionMeta struct {
	Name string `json:"name"`
}

// catalogFile is the JSON format of the persisted catalog.
type catalogFile struct {
	Collections []*CollectionMeta `json:"collections"`
}

// Catalog is the in-memory collection registry.
type Catalog struct {
	dir         string
	collections map[string]*CollectionMeta
}

// Load opens the catalog file from dataDir, creating it if absent.
func Load(dataDir string) (*Catalog, error) {
	c := &Catalog{
		dir:         dataDir,
		collections: make(map[string]*CollectionMeta),
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
	for _, col := range cf.Collections {
		c.collections[col.Name] = col
	}
	return c, nil
}

// CreateCollection registers a new collection. Returns an error if it already
// exists.
func (c *Catalog) CreateCollection(name string) error {
	if _, exists := c.collections[name]; exists {
		return fmt.Errorf("collection %q already exists", name)
	}
	c.collections[name] = &CollectionMeta{Name: name}
	return c.save()
}

// DropCollection removes a collection. Returns an error if not found.
func (c *Catalog) DropCollection(name string) error {
	if _, exists := c.collections[name]; !exists {
		return fmt.Errorf("collection %q does not exist", name)
	}
	delete(c.collections, name)
	return c.save()
}

// Has reports whether a collection exists.
func (c *Catalog) Has(name string) bool {
	_, ok := c.collections[name]
	return ok
}

// Get returns the metadata for a collection or an error if not found.
func (c *Catalog) Get(name string) (*CollectionMeta, error) {
	m, ok := c.collections[name]
	if !ok {
		return nil, fmt.Errorf("collection %q does not exist", name)
	}
	return m, nil
}

// Collections returns all collection names in no particular order.
func (c *Catalog) Collections() []string {
	names := make([]string, 0, len(c.collections))
	for name := range c.collections {
		names = append(names, name)
	}
	return names
}

// save persists the catalog to disk.
func (c *Catalog) save() error {
	cols := make([]*CollectionMeta, 0, len(c.collections))
	for _, m := range c.collections {
		cols = append(cols, m)
	}
	data, err := json.MarshalIndent(catalogFile{Collections: cols}, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(c.dir, "catalog.json")
	return os.WriteFile(path, data, 0644)
}
