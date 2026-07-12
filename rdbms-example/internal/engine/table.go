// Package engine wires together all data structures into a working table layer.
//
// Each Table instance owns:
//   - A B+ Tree    — primary-key index (int64 PK → uint32 rowID)
//   - A Bloom Filter — existence gate (avoid B+ Tree + disk hit for missing PKs)
//   - A map of Tries  — one per TEXT column (enables LIKE 'prefix%' searches)
//   - A Pager       — file-backed row storage
//
// On startup these are all rebuilt from the .rows file, making the file the
// single source of truth.
package engine

import (
	"fmt"
	"strconv"

	"rdbms/internal/catalog"
	"rdbms/internal/ds"
	"rdbms/internal/storage"
)

// Row is an alias for the storage layer's row type.
type Row = storage.Row

// Table is a single open table with all its in-memory indices.
type Table struct {
	schema *catalog.TableSchema
	bpt    *ds.BPTree
	bloom  *ds.BloomFilter
	tries  map[string]*ds.Trie // column name → Trie (TEXT columns only)
	pager  *storage.Pager
}

// openTable loads a table from disk and rebuilds all in-memory indices.
func openTable(schema *catalog.TableSchema, dataDir string) (*Table, error) {
	pager, err := storage.Open(dataDir, schema.Name)
	if err != nil {
		return nil, err
	}

	t := &Table{
		schema: schema,
		bpt:    ds.NewBPTree(),
		bloom:  ds.NewBloomFilter(1024, 0.01),
		tries:  make(map[string]*ds.Trie),
		pager:  pager,
	}

	// Initialise a Trie for every TEXT column.
	for _, col := range schema.Columns {
		if col.Type == catalog.ColText {
			t.tries[col.Name] = ds.NewTrie()
		}
	}

	// Rebuild indices from the rows file.
	rows, err := pager.ScanAll()
	if err != nil {
		return nil, fmt.Errorf("index rebuild: %w", err)
	}
	pkCol := schema.PKColumn().Name
	for _, row := range rows {
		ridF, ok := row["_rid"]
		if !ok {
			continue
		}
		rid := uint32(ridF.(float64))
		pkF, ok := row[pkCol]
		if !ok {
			continue
		}
		pk := int64(pkF.(float64))
		t.bpt.Insert(pk, rid)
		t.bloom.Add(pk)

		// Populate Trie indices.
		for colName, tr := range t.tries {
			if v, ok := row[colName]; ok {
				if s, ok := v.(string); ok {
					tr.Insert(s, rid)
				}
			}
		}
	}
	return t, nil
}

// close flushes and closes the pager.
func (t *Table) close() error { return t.pager.Close() }

// pkFromRow extracts the primary key (int64) from a raw value string.
func pkFromValue(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// coerceValue converts a raw string value to the appropriate Go type per column type.
func coerceValue(raw string, colType catalog.ColType) (any, error) {
	switch colType {
	case catalog.ColInt:
		v, err := strconv.ParseInt(raw, 10, 64)
		return v, err
	case catalog.ColFloat:
		v, err := strconv.ParseFloat(raw, 64)
		return v, err
	case catalog.ColBool:
		v, err := strconv.ParseBool(raw)
		return v, err
	case catalog.ColText:
		return raw, nil
	}
	return raw, nil
}

// buildRow converts a list of raw value strings (INSERT VALUES) into a Row.
func (t *Table) buildRow(values []string) (Row, error) {
	if len(values) != len(t.schema.Columns) {
		return nil, fmt.Errorf("expected %d values, got %d", len(t.schema.Columns), len(values))
	}
	row := make(Row, len(values))
	for i, col := range t.schema.Columns {
		v, err := coerceValue(values[i], col.Type)
		if err != nil {
			return nil, fmt.Errorf("column %q: %w", col.Name, err)
		}
		row[col.Name] = v
	}
	return row, nil
}

// ── CRUD operations ───────────────────────────────────────────────────────────

// Insert adds a new row. Returns an error if the primary key already exists.
func (t *Table) Insert(values []string) error {
	row, err := t.buildRow(values)
	if err != nil {
		return err
	}

	pkCol := t.schema.PKColumn()
	pkRaw, ok := row[pkCol.Name]
	if !ok {
		return fmt.Errorf("missing primary key column %q", pkCol.Name)
	}
	pk := int64(pkRaw.(int64))

	// Bloom check first — fast path for duplicate detection.
	if t.bloom.MightContain(pk) {
		if _, exists := t.bpt.Search(pk); exists {
			return fmt.Errorf("duplicate primary key %d", pk)
		}
	}

	rid, err := t.pager.Write(row)
	if err != nil {
		return err
	}

	t.bpt.Insert(pk, rid)
	t.bloom.Add(pk)

	// Update Trie indices.
	for colName, tr := range t.tries {
		if v, ok := row[colName]; ok {
			if s, ok := v.(string); ok {
				tr.Insert(s, rid)
			}
		}
	}
	return nil
}

// GetByPK retrieves a single row by primary key.
// Query path: Bloom Filter → B+ Tree → Pager.
func (t *Table) GetByPK(pk int64) (Row, error) {
	// ① Bloom Filter: O(k) — definitely-absent keys never reach disk.
	if !t.bloom.MightContain(pk) {
		return nil, fmt.Errorf("row with PK %d not found", pk)
	}
	// ② B+ Tree: O(log n) — get rowID.
	rid, ok := t.bpt.Search(pk)
	if !ok {
		return nil, fmt.Errorf("row with PK %d not found", pk)
	}
	// ③ Pager: O(1) seek — read the row from disk.
	return t.pager.Read(rid)
}

// Scan returns all rows satisfying pred (nil pred = return all).
func (t *Table) Scan(pred func(Row) bool) ([]Row, error) {
	rows, err := t.pager.ScanAll()
	if err != nil {
		return nil, err
	}
	if pred == nil {
		return rows, nil
	}
	var result []Row
	for _, row := range rows {
		if pred(row) {
			result = append(result, row)
		}
	}
	return result, nil
}

// RangeScan returns rows whose PK is in [lo, hi] using the B+ Tree leaf list.
func (t *Table) RangeScan(lo, hi int64) ([]Row, error) {
	entries := t.bpt.RangeScan(lo, hi)
	rows := make([]Row, 0, len(entries))
	for _, e := range entries {
		row, err := t.pager.Read(e.Val)
		if err != nil {
			continue
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// PrefixScan returns rows where column col starts with prefix.
// Uses the Trie index if available, otherwise falls back to a full scan.
func (t *Table) PrefixScan(col, prefix string) ([]Row, error) {
	tr, hasTrie := t.tries[col]
	if hasTrie {
		// ① Trie: O(m + k) — find all rowIDs with col value having this prefix.
		rowIDs := tr.PrefixSearch(prefix)
		rows := make([]Row, 0, len(rowIDs))
		seen := make(map[uint32]bool)
		for _, rid := range rowIDs {
			if seen[rid] {
				continue
			}
			seen[rid] = true
			row, err := t.pager.Read(rid)
			if err != nil {
				continue
			}
			rows = append(rows, row)
		}
		return rows, nil
	}
	// Fallback: full scan with string prefix check.
	return t.Scan(func(row Row) bool {
		v, ok := row[col]
		if !ok {
			return false
		}
		s, ok := v.(string)
		return ok && ds.HasPrefix(s, prefix)
	})
}

// SubstringScan returns rows where column col contains pattern.
// Uses Rabin-Karp rolling hash over each row's column value.
func (t *Table) SubstringScan(col, pattern string) ([]Row, error) {
	return t.Scan(func(row Row) bool {
		v, ok := row[col]
		if !ok {
			return false
		}
		s, ok := v.(string)
		return ok && ds.Contains(s, pattern)
	})
}

// SuffixScan returns rows where column col ends with suffix.
func (t *Table) SuffixScan(col, suffix string) ([]Row, error) {
	return t.Scan(func(row Row) bool {
		v, ok := row[col]
		if !ok {
			return false
		}
		s, ok := v.(string)
		return ok && ds.HasSuffix(s, suffix)
	})
}

// Update applies assignments to all rows matching pred.
// If the WHERE is on the PK, uses the B+ Tree for O(log n) lookup.
func (t *Table) Update(pred func(Row) bool, assignments map[string]string, schema *catalog.TableSchema) (int, error) {
	rows, err := t.pager.ScanAll()
	if err != nil {
		return 0, err
	}

	count := 0
	pkCol := t.schema.PKColumn().Name
	for _, row := range rows {
		if pred != nil && !pred(row) {
			continue
		}

		rid := uint32(row["_rid"].(float64))
		updates := make(Row)
		for col, rawVal := range assignments {
			colIdx := t.schema.ColumnIndex(col)
			if colIdx < 0 {
				return count, fmt.Errorf("unknown column %q", col)
			}
			v, err := coerceValue(rawVal, t.schema.Columns[colIdx].Type)
			if err != nil {
				return count, err
			}
			updates[col] = v
		}

		// Update Trie indices: remove old value, add new.
		for colName, tr := range t.tries {
			if newVal, changing := updates[colName]; changing {
				if oldVal, ok := row[colName]; ok {
					if s, ok := oldVal.(string); ok {
						tr.Delete(s, rid)
					}
				}
				if s, ok := newVal.(string); ok {
					tr.Insert(s, rid)
				}
			}
		}

		if err := t.pager.Update(rid, updates); err != nil {
			return count, err
		}

		// If PK changed (unusual but valid), update B+ Tree and Bloom.
		if newPKRaw, ok := updates[pkCol]; ok {
			oldPK := int64(row[pkCol].(float64))
			newPK := int64(newPKRaw.(int64))
			if oldPK != newPK {
				t.bpt.Delete(oldPK)
				t.bpt.Insert(newPK, rid)
				t.bloom.Add(newPK)
			}
		}
		count++
	}
	return count, nil
}

// Delete removes all rows matching pred.
func (t *Table) Delete(pred func(Row) bool) (int, error) {
	rows, err := t.pager.ScanAll()
	if err != nil {
		return 0, err
	}

	count := 0
	pkCol := t.schema.PKColumn().Name
	for _, row := range rows {
		if pred != nil && !pred(row) {
			continue
		}

		rid := uint32(row["_rid"].(float64))
		pk := int64(row[pkCol].(float64))

		// Remove from Trie indices.
		for colName, tr := range t.tries {
			if v, ok := row[colName]; ok {
				if s, ok := v.(string); ok {
					tr.Delete(s, rid)
				}
			}
		}

		t.bpt.Delete(pk)
		// Note: we do NOT remove from Bloom Filter (no deletion support).
		// False positives after deletion are fine — they hit the B+ Tree
		// and get a definitive "not found" answer there.

		if err := t.pager.SoftDelete(rid); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// OrderByInt returns rows sorted by an INT column using a Red-Black Tree.
// The RBTree's in-order traversal gives ascending order; InOrderDesc gives DESC.
func OrderByInt(rows []Row, col string, desc bool) []Row {
	rbt := ds.NewRBTree()
	for _, row := range rows {
		v, ok := row[col]
		if !ok {
			continue
		}
		var key int64
		switch v := v.(type) {
		case int64:
			key = v
		case float64:
			key = int64(v)
		default:
			continue
		}
		rbt.Insert(key, row)
	}

	var sorted []any
	if desc {
		sorted = rbt.InOrderDesc()
	} else {
		sorted = rbt.InOrder()
	}

	result := make([]Row, 0, len(sorted))
	for _, v := range sorted {
		result = append(result, v.(Row))
	}
	return result
}
