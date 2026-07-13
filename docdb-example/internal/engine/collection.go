// Package engine wires together all data structures into a working collection
// layer.
//
// Each Collection instance owns:
//   - A Hash Map      — O(1) document ID lookup (_id → document)
//   - An LSM Tree     — persistent key-value storage (with Skip List memtable)
//   - An Inverted Index — secondary field queries (field-value → posting lists)
//   - A Bloom Filter  — per-SST existence gate (inside the LSM Tree)
//
// On startup, the Hash Map and Inverted Index are rebuilt from the storage
// file, making the file the single source of truth.
package engine

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"docdb/internal/catalog"
	"docdb/internal/ds"
	"docdb/internal/storage"
)

// Doc is an alias for the storage layer's document type.
type Doc = storage.Doc

// Collection is a single open collection with all its in-memory indices.
type Collection struct {
	meta     *catalog.CollectionMeta
	hashMap  *ds.HashMap        // _id → serialized document (fast O(1) lookup)
	inverted *ds.InvertedIndex  // field-value → document IDs
	lsm      *ds.LSMTree        // persistent key-value storage (with Skip List memtable)
	store    *storage.Store     // persistent storage
}

// openCollection loads a collection from disk and rebuilds all in-memory
// indices.
func openCollection(meta *catalog.CollectionMeta, dataDir string) (*Collection, error) {
	st, err := storage.Open(dataDir, meta.Name)
	if err != nil {
		return nil, err
	}

	c := &Collection{
		meta:     meta,
		hashMap:  ds.NewHashMap(),
		inverted: ds.NewInvertedIndex(),
		lsm:      ds.NewLSMTree(100),
		store:    st,
	}

	// Rebuild indices from the storage file.
	docs, err := st.ScanAll()
	if err != nil {
		return nil, fmt.Errorf("index rebuild: %w", err)
	}
	for _, doc := range docs {
		id, ok := doc["_id"]
		if !ok {
			continue
		}
		docID := fmt.Sprintf("%v", id)

		data, _ := json.Marshal(doc)
		c.hashMap.Put(docID, data)
		c.lsm.Put(docID, data)

		// Populate inverted index.
		c.indexDoc(docID, doc)
	}

	return c, nil
}

// close closes the underlying store.
func (c *Collection) close() error { return c.store.Close() }

// ── CRUD operations ──────────────────────────────────────────────────────────

// Insert adds a new document. Generates an _id if not provided.
func (c *Collection) Insert(doc Doc) (string, error) {
	// Generate _id if missing.
	id, ok := doc["_id"]
	if !ok {
		doc["_id"] = generateID()
		id = doc["_id"]
	}
	docID := fmt.Sprintf("%v", id)

	// Check for duplicate _id using Hash Map — O(1).
	if c.hashMap.Has(docID) {
		return "", fmt.Errorf("duplicate document _id %q", docID)
	}

	// Write to persistent storage.
	if err := c.store.Write(doc); err != nil {
		return "", err
	}

	// Update Hash Map index.
	data, _ := json.Marshal(doc)
	c.hashMap.Put(docID, data)

	// Update LSM Tree.
	c.lsm.Put(docID, data)

	// Update Inverted Index.
	c.indexDoc(docID, doc)

	return docID, nil
}

// Get retrieves a document by _id. Uses the Hash Map for O(1) lookup.
func (c *Collection) Get(docID string) (Doc, error) {
	// ① Hash Map: O(1)
	data, ok := c.hashMap.Get(docID)
	if !ok {
		return nil, fmt.Errorf("document %q not found", docID)
	}

	var doc Doc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// Find returns all documents matching the filter.
// Dispatches to the optimal data-structure path based on the filter.
func (c *Collection) Find(filter map[string]any) ([]Doc, error) {
	if filter == nil || len(filter) == 0 {
		// No filter: return all documents.
		return c.scanAll()
	}

	// Check for _id filter — O(1) Hash Map lookup.
	if idVal, ok := filter["_id"]; ok {
		docID := fmt.Sprintf("%v", idVal)
		doc, err := c.Get(docID)
		if err != nil {
			return []Doc{}, nil
		}
		return []Doc{doc}, nil
	}

	// Check for equality filters — Inverted Index lookup.
	// If any filter field has a simple value (not an operator map), use the
	// inverted index.
	for field, val := range filter {
		switch v := val.(type) {
		case string:
			// Exact equality via Inverted Index.
			return c.findByInvertedIndex(field, v)
		case map[string]any:
			// Operator filter — check for $prefix, $contains, or comparison.
			return c.findWithOperator(field, v)
		default:
			// Numeric or bool equality — full scan with predicate.
			return c.scanWithPredicate(func(doc Doc) bool {
				return fmt.Sprintf("%v", doc[field]) == fmt.Sprintf("%v", val)
			})
		}
	}

	return c.scanAll()
}

// Update modifies documents matching the filter.
func (c *Collection) Update(filter map[string]any, update map[string]any) (int, error) {
	docs, err := c.Find(filter)
	if err != nil {
		return 0, err
	}

	// Extract $set operations.
	sets, ok := update["$set"].(map[string]any)
	if !ok {
		// Treat the whole update as a $set.
		sets = update
	}

	count := 0
	for _, doc := range docs {
		docID := fmt.Sprintf("%v", doc["_id"])

		// Remove old inverted index entries.
		c.unindexDoc(docID, doc)

		// Apply updates.
		for k, v := range sets {
			doc[k] = v
		}

		// Write updated document.
		if err := c.store.Update(docID, doc); err != nil {
			return count, err
		}

		// Update Hash Map.
		data, _ := json.Marshal(doc)
		c.hashMap.Put(docID, data)

		// Update LSM Tree.
		c.lsm.Put(docID, data)

		// Re-index.
		c.indexDoc(docID, doc)

		count++
	}

	return count, nil
}

// Delete removes documents matching the filter.
func (c *Collection) Delete(filter map[string]any) (int, error) {
	docs, err := c.Find(filter)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, doc := range docs {
		docID := fmt.Sprintf("%v", doc["_id"])

		// Remove from inverted index.
		c.unindexDoc(docID, doc)

		// Remove from Hash Map.
		c.hashMap.Delete(docID)

		// Delete from LSM Tree.
		c.lsm.Delete(docID)

		// Soft delete in storage.
		if err := c.store.SoftDelete(docID); err != nil {
			return count, err
		}

		count++
	}

	return count, nil
}

// ── Query helpers ────────────────────────────────────────────────────────────

// findByInvertedIndex returns documents where field equals value.
func (c *Collection) findByInvertedIndex(field, value string) ([]Doc, error) {
	docIDs := c.inverted.Search(field, value)
	return c.getDocsByIDs(docIDs)
}

// findWithOperator handles operator-based filters like $gt, $prefix, etc.
func (c *Collection) findWithOperator(field string, ops map[string]any) ([]Doc, error) {
	for op, val := range ops {
		switch op {
		case "$eq":
			return c.findByInvertedIndex(field, fmt.Sprintf("%v", val))

		case "$ne":
			target := fmt.Sprintf("%v", val)
			return c.scanWithPredicate(func(doc Doc) bool {
				return fmt.Sprintf("%v", doc[field]) != target
			})

		case "$gt":
			return c.scanWithPredicate(numericPred(field, ">", val))

		case "$gte":
			return c.scanWithPredicate(numericPred(field, ">=", val))

		case "$lt":
			return c.scanWithPredicate(numericPred(field, "<", val))

		case "$lte":
			return c.scanWithPredicate(numericPred(field, "<=", val))

		case "$prefix":
			prefix := fmt.Sprintf("%v", val)
			docIDs := c.inverted.PrefixSearch(field, prefix)
			return c.getDocsByIDs(docIDs)

		case "$contains":
			substr := fmt.Sprintf("%v", val)
			docIDs := c.inverted.ContainsSearch(field, substr)
			return c.getDocsByIDs(docIDs)
		}
	}

	return nil, fmt.Errorf("unsupported operator in filter")
}

// scanAll returns all documents in the collection.
func (c *Collection) scanAll() ([]Doc, error) {
	entries := c.hashMap.Entries()
	docs := make([]Doc, 0, len(entries))
	for _, e := range entries {
		var doc Doc
		if err := json.Unmarshal(e.Val, &doc); err != nil {
			continue
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

// scanWithPredicate returns documents matching pred via full scan.
func (c *Collection) scanWithPredicate(pred func(Doc) bool) ([]Doc, error) {
	all, err := c.scanAll()
	if err != nil {
		return nil, err
	}
	var result []Doc
	for _, doc := range all {
		if pred(doc) {
			result = append(result, doc)
		}
	}
	return result, nil
}

// getDocsByIDs retrieves documents from the Hash Map by their IDs.
func (c *Collection) getDocsByIDs(docIDs []string) ([]Doc, error) {
	docs := make([]Doc, 0, len(docIDs))
	for _, id := range docIDs {
		doc, err := c.Get(id)
		if err != nil {
			continue
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

// ── Index management ─────────────────────────────────────────────────────────

// indexDoc adds a document's fields to the inverted index.
func (c *Collection) indexDoc(docID string, doc Doc) {
	for field, val := range doc {
		if field == "_id" || field == "_deleted" {
			continue
		}
		// Index string values directly.
		switch v := val.(type) {
		case string:
			c.inverted.Add(field, v, docID)
		default:
			// Index non-string values as their string representation.
			c.inverted.Add(field, fmt.Sprintf("%v", v), docID)
		}
	}
}

// unindexDoc removes a document's fields from the inverted index.
func (c *Collection) unindexDoc(docID string, doc Doc) {
	for field, val := range doc {
		if field == "_id" || field == "_deleted" {
			continue
		}
		switch v := val.(type) {
		case string:
			c.inverted.Delete(field, v, docID)
		default:
			c.inverted.Delete(field, fmt.Sprintf("%v", v), docID)
		}
	}
}

// ── Sort helpers ─────────────────────────────────────────────────────────────

// SortDocs sorts documents by a field. Uses Skip List for efficient sorted
// insertion.
func SortDocs(docs []Doc, field string, order int) []Doc {
	sl := ds.NewSkipList()
	for i, doc := range docs {
		// Build a sort key that preserves order.
		key := sortKey(doc, field, i)
		data, _ := json.Marshal(doc)
		sl.Insert(key, data)
	}

	entries := sl.InOrder()
	if order < 0 {
		// Reverse for descending.
		for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
			entries[i], entries[j] = entries[j], entries[i]
		}
	}

	result := make([]Doc, 0, len(entries))
	for _, e := range entries {
		var doc Doc
		if err := json.Unmarshal(e.Val, &doc); err != nil {
			continue
		}
		result = append(result, doc)
	}
	return result
}

// sortKey generates a string sort key from a document field value.
// Numeric values are zero-padded for correct lexicographic ordering.
func sortKey(doc Doc, field string, tiebreaker int) string {
	val := doc[field]
	var key string
	switch v := val.(type) {
	case float64:
		key = fmt.Sprintf("%020.6f", v)
	case string:
		key = v
	default:
		key = fmt.Sprintf("%v", v)
	}
	// Append tiebreaker for stable sort.
	return fmt.Sprintf("%s|%010d", key, tiebreaker)
}

// ── Predicates ───────────────────────────────────────────────────────────────

func numericPred(field, op string, val any) func(Doc) bool {
	target := toFloat64(val)
	return func(doc Doc) bool {
		v := toFloat64(doc[field])
		switch op {
		case ">":
			return v > target
		case ">=":
			return v >= target
		case "<":
			return v < target
		case "<=":
			return v <= target
		}
		return false
	}
}

func toFloat64(v any) float64 {
	switch v := v.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}

// ── ID generation ────────────────────────────────────────────────────────────

// generateID creates a random 12-byte hex document ID (similar to MongoDB
// ObjectId but simpler).
func generateID() string {
	b := make([]byte, 12)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// isStringValue returns true if the filter value is a simple string (not an
// operator map). Used for inverted index dispatch.
func isStringValue(v any) bool {
	_, ok := v.(string)
	return ok
}

// containsSubstring is a helper for LIKE '%substr%' style queries.
func containsSubstring(text, substr string) bool {
	return strings.Contains(text, substr)
}
