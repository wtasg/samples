// Package docdb is the DocDB Go client library.
//
// It provides a clean, idiomatic Go API over the DocDB Connect-RPC service.
// The library is wire-compatible with standard gRPC clients; it uses the
// Connect protocol by default (HTTP/1.1 or HTTP/2, JSON or binary).
package docdb

import (
	"encoding/json"
	"fmt"
)

// Doc is a document: a map of field name → value.
// Values are typed according to the document structure:
//
//	int64   → int64
//	float64 → float64
//	string  → string
//	bool    → bool
//	array   → []any
//	object  → map[string]any
type Doc map[string]any

// M is an alias for a map, representing query filters or update operations.
type M map[string]any

// Int returns the field value as int64, or 0 if absent or wrong type.
func (d Doc) Int(field string) int64 {
	switch v := d[field].(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	}
	return 0
}

// Float returns the field value as float64, or 0.0 if absent or wrong type.
func (d Doc) Float(field string) float64 {
	switch v := d[field].(type) {
	case float64:
		return v
	case int64:
		return float64(v)
	case int:
		return float64(v)
	}
	return 0.0
}

// Text returns the field value as string.
func (d Doc) Text(field string) string {
	if d[field] == nil {
		return ""
	}
	if s, ok := d[field].(string); ok {
		return s
	}
	return fmt.Sprintf("%v", d[field])
}

// Bool returns the field value as bool.
func (d Doc) Bool(field string) bool {
	if v, ok := d[field].(bool); ok {
		return v
	}
	return false
}

// IsNull returns true if the field value is nil.
func (d Doc) IsNull(field string) bool {
	return d[field] == nil
}

// Array returns the field value as an array of any type.
func (d Doc) Array(field string) []any {
	v, ok := d[field].([]any)
	if !ok {
		// If it's stored as a JSON string, try to parse it
		if s, ok := d[field].(string); ok {
			var arr []any
			if err := json.Unmarshal([]byte(s), &arr); err == nil {
				return arr
			}
		}
		return nil
	}
	return v
}

// Result is the output of a query.
type Result struct {
	Docs    []Doc
	Message string // populated for write operations (insert, update, delete)
}

// CollectionInfo contains collection metadata.
type CollectionInfo struct {
	Name     string
	DocCount int64
	Size     int64
}
