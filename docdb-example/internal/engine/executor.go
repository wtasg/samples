// executor.go — Query executor: dispatches parsed commands to collection
// operations.
//
// Data-structure dispatch summary:
//
//	GET by _id              → Hash Map O(1)
//	FIND by field=value     → Inverted Index → Hash Map → documents
//	FIND with $prefix       → Inverted Index prefix search → documents
//	FIND with $contains     → Inverted Index contains search → documents
//	FIND with $gt/$lt/...   → Full scan with typed predicate
//	SORT                    → Skip List ordered insertion → traversal
package engine

import (
	"fmt"
	"os"
	"path/filepath"

	"docdb/internal/catalog"
	"docdb/internal/parser"
	"docdb/internal/storage"
)

// Executor coordinates the catalog, open collections, and query execution.
type Executor struct {
	cat         *catalog.Catalog
	collections map[string]*Collection
	dataDir     string
}

// NewExecutor creates an executor rooted at dataDir.
func NewExecutor(dataDir string) (*Executor, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}
	cat, err := catalog.Load(dataDir)
	if err != nil {
		return nil, err
	}
	ex := &Executor{
		cat:         cat,
		collections: make(map[string]*Collection),
		dataDir:     dataDir,
	}
	// Pre-open all known collections.
	for _, name := range cat.Collections() {
		meta, _ := cat.Get(name)
		col, err := openCollection(meta, dataDir)
		if err != nil {
			return nil, fmt.Errorf("open collection %q: %w", name, err)
		}
		ex.collections[name] = col
	}
	return ex, nil
}

// Close flushes all open collections.
func (ex *Executor) Close() {
	for _, col := range ex.collections {
		col.close()
	}
}

// CollectionNames returns the names of all open collections.
func (ex *Executor) CollectionNames() []string {
	return ex.cat.Collections()
}

// Result is the output of a query.
type Result struct {
	Docs    []Doc
	Message string
}

// Execute runs a parsed Statement and returns a Result.
func (ex *Executor) Execute(stmt parser.Statement) (*Result, error) {
	switch s := stmt.(type) {
	case *parser.CreateCollectionStmt:
		return ex.execCreate(s)
	case *parser.DropCollectionStmt:
		return ex.execDrop(s)
	case *parser.InsertStmt:
		return ex.execInsert(s)
	case *parser.FindStmt:
		return ex.execFind(s)
	case *parser.UpdateStmt:
		return ex.execUpdate(s)
	case *parser.DeleteStmt:
		return ex.execDelete(s)
	}
	return nil, fmt.Errorf("unsupported statement type")
}

// ── CREATE COLLECTION ────────────────────────────────────────────────────────

func (ex *Executor) execCreate(s *parser.CreateCollectionStmt) (*Result, error) {
	if err := ex.cat.CreateCollection(s.Name); err != nil {
		return nil, err
	}

	// Ensure the data file exists.
	path := filepath.Join(ex.dataDir, s.Name+".docs")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	f.Close()

	meta, _ := ex.cat.Get(s.Name)
	col, err := openCollection(meta, ex.dataDir)
	if err != nil {
		return nil, err
	}
	ex.collections[s.Name] = col
	return &Result{Message: fmt.Sprintf("Collection %q created.", s.Name)}, nil
}

// ── DROP COLLECTION ──────────────────────────────────────────────────────────

func (ex *Executor) execDrop(s *parser.DropCollectionStmt) (*Result, error) {
	col, ok := ex.collections[s.Name]
	if !ok {
		return nil, fmt.Errorf("collection %q does not exist", s.Name)
	}
	col.close()
	delete(ex.collections, s.Name)

	if err := ex.cat.DropCollection(s.Name); err != nil {
		return nil, err
	}
	if err := storage.Drop(ex.dataDir, s.Name); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return &Result{Message: fmt.Sprintf("Collection %q dropped.", s.Name)}, nil
}

// ── INSERT ───────────────────────────────────────────────────────────────────

func (ex *Executor) execInsert(s *parser.InsertStmt) (*Result, error) {
	col, err := ex.getCollection(s.Collection)
	if err != nil {
		return nil, err
	}

	id, err := col.Insert(s.Document)
	if err != nil {
		return nil, err
	}
	return &Result{Message: fmt.Sprintf("Document inserted (id=%s).", id)}, nil
}

// ── FIND ─────────────────────────────────────────────────────────────────────

func (ex *Executor) execFind(s *parser.FindStmt) (*Result, error) {
	col, err := ex.getCollection(s.Collection)
	if err != nil {
		return nil, err
	}

	docs, err := col.Find(s.Filter)
	if err != nil {
		return nil, err
	}

	// Apply sort if requested.
	if s.SortField != "" {
		docs = SortDocs(docs, s.SortField, s.SortOrder)
	}

	return &Result{Docs: docs}, nil
}

// ── UPDATE ───────────────────────────────────────────────────────────────────

func (ex *Executor) execUpdate(s *parser.UpdateStmt) (*Result, error) {
	col, err := ex.getCollection(s.Collection)
	if err != nil {
		return nil, err
	}

	n, err := col.Update(s.Filter, s.Update)
	if err != nil {
		return nil, err
	}
	return &Result{Message: fmt.Sprintf("%d document(s) updated.", n)}, nil
}

// ── DELETE ───────────────────────────────────────────────────────────────────

func (ex *Executor) execDelete(s *parser.DeleteStmt) (*Result, error) {
	col, err := ex.getCollection(s.Collection)
	if err != nil {
		return nil, err
	}

	n, err := col.Delete(s.Filter)
	if err != nil {
		return nil, err
	}
	return &Result{Message: fmt.Sprintf("%d document(s) deleted.", n)}, nil
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func (ex *Executor) getCollection(name string) (*Collection, error) {
	col, ok := ex.collections[name]
	if !ok {
		return nil, fmt.Errorf("collection %q does not exist", name)
	}
	return col, nil
}
