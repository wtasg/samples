// Package storage manages on-disk persistence of table rows.
//
// Design: rows are stored as newline-delimited JSON in a plain text file
// (<dataDir>/<table>.rows).  Each row includes a "_rid" (row ID) field.
// Soft-deleted rows have "_deleted": true.
//
// An in-memory offset index (rowID → byte offset) is built by scanning the
// file at open time.  This gives O(1) random-access reads.
//
// The B+ Tree, Bloom Filter, and Trie stored in the engine layer use rowIDs
// as values; they are rebuilt from the rows file at startup.
package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// Row is a single database row: a map of column name → value.
type Row = map[string]any

// Pager manages the rows file for a single table.
type Pager struct {
	path    string           // path to <table>.rows file
	file    *os.File         // open file handle
	offsets map[uint32]int64 // rowID → byte offset in file
	nextRID uint32           // next row ID to assign
	mu      sync.Mutex
}

const deletedField = "_deleted"
const ridField = "_rid"

// Open opens (or creates) the rows file for the given table.
func Open(dataDir, tableName string) (*Pager, error) {
	path := filepath.Join(dataDir, tableName+".rows")
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("pager open %s: %w", path, err)
	}
	p := &Pager{
		path:    path,
		file:    f,
		offsets: make(map[uint32]int64),
	}
	if err := p.scan(); err != nil {
		f.Close()
		return nil, err
	}
	return p, nil
}

// scan rebuilds the offset index from the rows file.
func (p *Pager) scan() error {
	if _, err := p.file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	scanner := bufio.NewScanner(p.file)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)

	var offset int64
	for scanner.Scan() {
		line := scanner.Bytes()
		lineLen := int64(len(line)) + 1 // +1 for '\n'

		var row Row
		if err := json.Unmarshal(line, &row); err != nil {
			offset += lineLen
			continue
		}

		ridF, ok := row[ridField]
		if !ok {
			offset += lineLen
			continue
		}
		rid := uint32(ridF.(float64)) // JSON numbers are float64
		p.offsets[rid] = offset
		if rid >= p.nextRID {
			p.nextRID = rid + 1
		}
		offset += lineLen
	}
	return scanner.Err()
}

// Write appends a new row to the file and returns its assigned row ID.
// The row must NOT contain _rid or _deleted (those are injected here).
func (p *Pager) Write(row Row) (uint32, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	rid := p.nextRID
	p.nextRID++

	r := make(Row, len(row)+2)
	for k, v := range row {
		r[k] = v
	}
	r[ridField] = rid
	r[deletedField] = false

	data, err := json.Marshal(r)
	if err != nil {
		return 0, err
	}

	// Seek to end.
	offset, err := p.file.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}

	if _, err := p.file.Write(append(data, '\n')); err != nil {
		return 0, err
	}
	if err := p.file.Sync(); err != nil {
		return 0, err
	}

	p.offsets[rid] = offset
	return rid, nil
}

// Read fetches a row by its row ID.
func (p *Pager) Read(rid uint32) (Row, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	offset, ok := p.offsets[rid]
	if !ok {
		return nil, fmt.Errorf("row %d not found", rid)
	}

	if _, err := p.file.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(p.file)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	if !scanner.Scan() {
		return nil, fmt.Errorf("row %d: empty read", rid)
	}

	var row Row
	if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
		return nil, err
	}
	return row, nil
}

// SoftDelete marks a row as deleted in-place (overwrites the JSON line).
// The line length stays the same; the deleted flag is flipped to true.
func (p *Pager) SoftDelete(rid uint32) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	row, err := p.readLocked(rid)
	if err != nil {
		return err
	}
	row[deletedField] = true

	return p.rewriteLocked(rid, row)
}

// Update rewrites a row in place (value fields change, but _rid stays).
func (p *Pager) Update(rid uint32, updates Row) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	row, err := p.readLocked(rid)
	if err != nil {
		return err
	}
	for k, v := range updates {
		row[k] = v
	}
	return p.rewriteLocked(rid, row)
}

// readLocked reads a row without acquiring p.mu (caller holds it).
func (p *Pager) readLocked(rid uint32) (Row, error) {
	offset, ok := p.offsets[rid]
	if !ok {
		return nil, fmt.Errorf("row %d not found", rid)
	}
	if _, err := p.file.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(p.file)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	if !scanner.Scan() {
		return nil, fmt.Errorf("row %d: empty read", rid)
	}
	var row Row
	return row, json.Unmarshal(scanner.Bytes(), &row)
}

// rewriteLocked overwrites the row's JSON line in place.
// Padding is added if the new JSON is shorter; this maintains byte offsets.
// NOTE: JSON can shrink only if we remove fields, which we don't.  For safety
// we append a new version and tombstone the old one when the row grows.
func (p *Pager) rewriteLocked(rid uint32, row Row) error {
	offset := p.offsets[rid]

	// Measure original line length.
	if _, err := p.file.Seek(offset, io.SeekStart); err != nil {
		return err
	}
	scanner := bufio.NewScanner(p.file)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	scanner.Scan()
	origLen := len(scanner.Bytes())

	data, err := json.Marshal(row)
	if err != nil {
		return err
	}

	if len(data) <= origLen {
		// Pad with spaces to keep line length stable.
		padded := make([]byte, origLen)
		copy(padded, data)
		for i := len(data); i < origLen; i++ {
			padded[i] = ' '
		}
		if _, err := p.file.Seek(offset, io.SeekStart); err != nil {
			return err
		}
		_, err = p.file.Write(padded)
		return err
	}

	// Row grew: append new version, tombstone old.
	row[deletedField] = false
	data, _ = json.Marshal(row)
	newOffset, err := p.file.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	if _, err := p.file.Write(append(data, '\n')); err != nil {
		return err
	}

	// Tombstone the old entry.
	oldRow, _ := p.readLocked(rid) // re-read to get current
	oldRow[deletedField] = true
	tombData, _ := json.Marshal(oldRow)
	padded := make([]byte, origLen)
	copy(padded, tombData)
	for i := len(tombData); i < origLen; i++ {
		padded[i] = ' '
	}
	if _, err := p.file.Seek(offset, io.SeekStart); err != nil {
		return err
	}
	p.file.Write(padded) //nolint

	p.offsets[rid] = newOffset
	return p.file.Sync()
}

// ScanAll returns all non-deleted rows in file order.
func (p *Pager) ScanAll() ([]Row, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, err := p.file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(p.file)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)

	var rows []Row
	for scanner.Scan() {
		var row Row
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			continue
		}
		if del, _ := row[deletedField].(bool); del {
			continue
		}
		rows = append(rows, row)
	}
	return rows, scanner.Err()
}

// AllRowIDs returns the rowIDs of all non-deleted rows (for index rebuilding).
func (p *Pager) AllRowIDs() []uint32 {
	p.mu.Lock()
	defer p.mu.Unlock()

	var ids []uint32
	for rid := range p.offsets {
		ids = append(ids, rid)
	}
	return ids
}

// Close flushes and closes the underlying file.
func (p *Pager) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.file.Sync(); err != nil {
		return err
	}
	return p.file.Close()
}

// IsDeleted reports whether a given row is soft-deleted.
func (p *Pager) IsDeleted(rid uint32) bool {
	row, err := p.readLocked(rid)
	if err != nil {
		return true
	}
	del, _ := row[deletedField].(bool)
	return del
}

// Drop removes the rows file from disk.
func Drop(dataDir, tableName string) error {
	return os.Remove(filepath.Join(dataDir, tableName+".rows"))
}
