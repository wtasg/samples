// inverted.go — Inverted Index for secondary-key queries on document fields.
//
// An Inverted Index maps field-value pairs to posting lists (sets of document
// IDs). This is the data structure that enables queries like:
//
//	db.products.find({"category": "electronics"})
//
// without requiring a full collection scan.
//
// Inverted indexes are foundational in:
//   - Elasticsearch / Apache Lucene — full-text search
//   - MongoDB — secondary indexes on arbitrary fields
//   - Apache Solr — faceted search and filtering
//   - Google Search — web page indexing
//
// Structure:
//
//	field "category":
//	  "electronics" → [docID-1, docID-5, docID-12]
//	  "clothing"    → [docID-2, docID-8]
//	  "books"       → [docID-3, docID-6, docID-7]
//
//	field "tags":
//	  "sale"        → [docID-1, docID-3]
//	  "new"         → [docID-5, docID-12]
//
// Each field has its own term→posting-list map. Posting lists store document
// IDs as strings to match the document database's _id field type.
package ds

import "strings"

// PostingList is a set of document IDs that contain a particular term.
type PostingList struct {
	DocIDs []string
}

// Add appends a document ID if not already present.
func (pl *PostingList) Add(docID string) {
	for _, id := range pl.DocIDs {
		if id == docID {
			return
		}
	}
	pl.DocIDs = append(pl.DocIDs, docID)
}

// Remove removes a document ID from the posting list.
func (pl *PostingList) Remove(docID string) bool {
	for i, id := range pl.DocIDs {
		if id == docID {
			pl.DocIDs = append(pl.DocIDs[:i], pl.DocIDs[i+1:]...)
			return true
		}
	}
	return false
}

// Contains reports whether docID is in the posting list.
func (pl *PostingList) Contains(docID string) bool {
	for _, id := range pl.DocIDs {
		if id == docID {
			return true
		}
	}
	return false
}

// fieldIndex is the per-field index mapping term values to posting lists.
type fieldIndex struct {
	terms map[string]*PostingList
}

// InvertedIndex maps (field, value) → posting list of document IDs.
type InvertedIndex struct {
	fields map[string]*fieldIndex
}

// NewInvertedIndex returns an empty inverted index.
func NewInvertedIndex() *InvertedIndex {
	return &InvertedIndex{fields: make(map[string]*fieldIndex)}
}

// Add indexes a document ID under (field, value).
func (idx *InvertedIndex) Add(field, value, docID string) {
	fi := idx.getOrCreateField(field)
	pl, ok := fi.terms[value]
	if !ok {
		pl = &PostingList{}
		fi.terms[value] = pl
	}
	pl.Add(docID)
}

// Search returns all document IDs where field equals value exactly.
func (idx *InvertedIndex) Search(field, value string) []string {
	fi, ok := idx.fields[field]
	if !ok {
		return nil
	}
	pl, ok := fi.terms[value]
	if !ok {
		return nil
	}
	result := make([]string, len(pl.DocIDs))
	copy(result, pl.DocIDs)
	return result
}

// PrefixSearch returns all document IDs where the field value starts with prefix.
func (idx *InvertedIndex) PrefixSearch(field, prefix string) []string {
	fi, ok := idx.fields[field]
	if !ok {
		return nil
	}

	seen := make(map[string]bool)
	var result []string
	for term, pl := range fi.terms {
		if strings.HasPrefix(term, prefix) {
			for _, id := range pl.DocIDs {
				if !seen[id] {
					seen[id] = true
					result = append(result, id)
				}
			}
		}
	}
	return result
}

// ContainsSearch returns all document IDs where the field value contains substr.
func (idx *InvertedIndex) ContainsSearch(field, substr string) []string {
	fi, ok := idx.fields[field]
	if !ok {
		return nil
	}

	seen := make(map[string]bool)
	var result []string
	for term, pl := range fi.terms {
		if strings.Contains(term, substr) {
			for _, id := range pl.DocIDs {
				if !seen[id] {
					seen[id] = true
					result = append(result, id)
				}
			}
		}
	}
	return result
}

// Delete removes a document ID from the posting list for (field, value).
func (idx *InvertedIndex) Delete(field, value, docID string) bool {
	fi, ok := idx.fields[field]
	if !ok {
		return false
	}
	pl, ok := fi.terms[value]
	if !ok {
		return false
	}
	removed := pl.Remove(docID)
	if len(pl.DocIDs) == 0 {
		delete(fi.terms, value)
	}
	return removed
}

// DeleteDoc removes a document ID from ALL posting lists across ALL fields.
// Used when deleting a document entirely.
func (idx *InvertedIndex) DeleteDoc(docID string) {
	for _, fi := range idx.fields {
		for term, pl := range fi.terms {
			pl.Remove(docID)
			if len(pl.DocIDs) == 0 {
				delete(fi.terms, term)
			}
		}
	}
}

// FieldTerms returns all indexed terms for a given field (for debugging).
func (idx *InvertedIndex) FieldTerms(field string) []string {
	fi, ok := idx.fields[field]
	if !ok {
		return nil
	}
	terms := make([]string, 0, len(fi.terms))
	for t := range fi.terms {
		terms = append(terms, t)
	}
	return terms
}

// FieldCount returns the number of indexed fields.
func (idx *InvertedIndex) FieldCount() int { return len(idx.fields) }

// getOrCreateField returns or creates the fieldIndex for the named field.
func (idx *InvertedIndex) getOrCreateField(field string) *fieldIndex {
	fi, ok := idx.fields[field]
	if !ok {
		fi = &fieldIndex{terms: make(map[string]*PostingList)}
		idx.fields[field] = fi
	}
	return fi
}
