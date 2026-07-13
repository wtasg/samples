// Package storage manages on-disk persistence of documents.
//
// Documents are stored as newline-delimited JSON in a plain text file
// (<dataDir>/<collection>.docs). Each document includes an "_id" field.
// Soft-deleted documents have "_deleted": true.
//
// An in-memory offset index (_id → byte offset) is built by scanning the
// file at open time, giving O(1) random-access reads.
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

// Doc is a single document: a map of field name → value.
type Doc = map[string]any

const deletedField = "_deleted"
const idField = "_id"

// Store manages the document file for a single collection.
type Store struct {
	path    string           // path to <collection>.docs file
	file    *os.File         // open file handle
	offsets map[string]int64 // docID → byte offset in file
	mu      sync.Mutex
}

// Open opens (or creates) the document file for the given collection.
func Open(dataDir, collectionName string) (*Store, error) {
	path := filepath.Join(dataDir, collectionName+".docs")
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("store open %s: %w", path, err)
	}
	s := &Store{
		path:    path,
		file:    f,
		offsets: make(map[string]int64),
	}
	if err := s.scan(); err != nil {
		f.Close()
		return nil, err
	}
	return s, nil
}

// scan rebuilds the offset index from the document file.
func (s *Store) scan() error {
	if _, err := s.file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	scanner := bufio.NewScanner(s.file)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)

	var offset int64
	for scanner.Scan() {
		line := scanner.Bytes()
		lineLen := int64(len(line)) + 1 // +1 for '\n'

		var doc Doc
		if err := json.Unmarshal(line, &doc); err != nil {
			offset += lineLen
			continue
		}

		id, ok := doc[idField]
		if !ok {
			offset += lineLen
			continue
		}
		docID := fmt.Sprintf("%v", id)

		// Only track non-deleted documents.
		if del, _ := doc[deletedField].(bool); !del {
			s.offsets[docID] = offset
		} else {
			delete(s.offsets, docID)
		}

		offset += lineLen
	}
	return scanner.Err()
}

// Write appends a new document to the file and returns its ID.
func (s *Store) Write(doc Doc) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	docID, ok := doc[idField]
	if !ok {
		return fmt.Errorf("document missing _id field")
	}
	id := fmt.Sprintf("%v", docID)

	d := make(Doc, len(doc)+1)
	for k, v := range doc {
		d[k] = v
	}
	d[deletedField] = false

	data, err := json.Marshal(d)
	if err != nil {
		return err
	}

	offset, err := s.file.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}

	if _, err := s.file.Write(append(data, '\n')); err != nil {
		return err
	}
	if err := s.file.Sync(); err != nil {
		return err
	}

	s.offsets[id] = offset
	return nil
}

// Read fetches a document by its _id.
func (s *Store) Read(docID string) (Doc, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	offset, ok := s.offsets[docID]
	if !ok {
		return nil, fmt.Errorf("document %q not found", docID)
	}

	if _, err := s.file.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(s.file)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	if !scanner.Scan() {
		return nil, fmt.Errorf("document %q: empty read", docID)
	}

	var doc Doc
	if err := json.Unmarshal(scanner.Bytes(), &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// SoftDelete marks a document as deleted by appending a tombstone record.
func (s *Store) SoftDelete(docID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.offsets[docID]; !ok {
		return fmt.Errorf("document %q not found", docID)
	}

	// Append a tombstone record.
	tomb := Doc{idField: docID, deletedField: true}
	data, err := json.Marshal(tomb)
	if err != nil {
		return err
	}

	if _, err := s.file.Seek(0, io.SeekEnd); err != nil {
		return err
	}
	if _, err := s.file.Write(append(data, '\n')); err != nil {
		return err
	}
	if err := s.file.Sync(); err != nil {
		return err
	}

	delete(s.offsets, docID)
	return nil
}

// Update overwrites a document by appending a new version and marking the
// old one as superseded.
func (s *Store) Update(docID string, doc Doc) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.offsets[docID]; !ok {
		return fmt.Errorf("document %q not found", docID)
	}

	d := make(Doc, len(doc)+2)
	for k, v := range doc {
		d[k] = v
	}
	d[idField] = docID
	d[deletedField] = false

	data, err := json.Marshal(d)
	if err != nil {
		return err
	}

	offset, err := s.file.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	if _, err := s.file.Write(append(data, '\n')); err != nil {
		return err
	}
	if err := s.file.Sync(); err != nil {
		return err
	}

	s.offsets[docID] = offset
	return nil
}

// ScanAll returns all non-deleted documents in file order.
func (s *Store) ScanAll() ([]Doc, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(s.file)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)

	// Collect the latest version of each document.
	latest := make(map[string]Doc)
	for scanner.Scan() {
		var doc Doc
		if err := json.Unmarshal(scanner.Bytes(), &doc); err != nil {
			continue
		}
		id, ok := doc[idField]
		if !ok {
			continue
		}
		docID := fmt.Sprintf("%v", id)
		if del, _ := doc[deletedField].(bool); del {
			delete(latest, docID)
		} else {
			latest[docID] = doc
		}
	}

	docs := make([]Doc, 0, len(latest))
	for _, doc := range latest {
		docs = append(docs, doc)
	}
	return docs, scanner.Err()
}

// DocIDs returns the IDs of all non-deleted documents.
func (s *Store) DocIDs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	ids := make([]string, 0, len(s.offsets))
	for id := range s.offsets {
		ids = append(ids, id)
	}
	return ids
}

// Close flushes and closes the underlying file.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.file.Sync(); err != nil {
		return err
	}
	return s.file.Close()
}

// Drop removes the document file from disk.
func Drop(dataDir, collectionName string) error {
	return os.Remove(filepath.Join(dataDir, collectionName+".docs"))
}
