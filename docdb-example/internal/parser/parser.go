// Package parser implements a hand-written parser for the DocDB query language.
//
// Supported commands:
//
//	db.createCollection("name")
//	db.dropCollection("name")
//	db.collection.insert({...})
//	db.collection.find({filter}).sort({field: 1/-1})
//	db.collection.update({filter}, {$set: {...}})
//	db.collection.delete({filter})
//
// Filter operators:
//
//	{"field": "value"}               — exact equality ($eq)
//	{"field": {"$gt": N}}            — greater than
//	{"field": {"$gte": N}}           — greater than or equal
//	{"field": {"$lt": N}}            — less than
//	{"field": {"$lte": N}}           — less than or equal
//	{"field": {"$ne": "value"}}      — not equal
//	{"field": {"$prefix": "pre"}}    — prefix search
//	{"field": {"$contains": "sub"}}  — substring search
package parser

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ── Statement types ──────────────────────────────────────────────────────────

// Statement is a parsed DocDB command.
type Statement interface {
	stmtMarker()
}

// CreateCollectionStmt represents db.createCollection("name").
type CreateCollectionStmt struct {
	Name string
}

func (*CreateCollectionStmt) stmtMarker() {}

// DropCollectionStmt represents db.dropCollection("name").
type DropCollectionStmt struct {
	Name string
}

func (*DropCollectionStmt) stmtMarker() {}

// InsertStmt represents db.collection.insert({...}).
type InsertStmt struct {
	Collection string
	Document   map[string]any
}

func (*InsertStmt) stmtMarker() {}

// FindStmt represents db.collection.find({filter}).sort({...}).
type FindStmt struct {
	Collection string
	Filter     map[string]any
	SortField  string
	SortOrder  int // 1 = ascending, -1 = descending
}

func (*FindStmt) stmtMarker() {}

// UpdateStmt represents db.collection.update({filter}, {$set: {...}}).
type UpdateStmt struct {
	Collection string
	Filter     map[string]any
	Update     map[string]any
}

func (*UpdateStmt) stmtMarker() {}

// DeleteStmt represents db.collection.delete({filter}).
type DeleteStmt struct {
	Collection string
	Filter     map[string]any
}

func (*DeleteStmt) stmtMarker() {}

// ── Parser ───────────────────────────────────────────────────────────────────

// Parse parses a DocDB command string and returns a Statement.
func Parse(input string) (Statement, error) {
	input = strings.TrimSpace(input)
	input = strings.TrimSuffix(input, ";")
	input = strings.TrimSpace(input)

	if input == "" {
		return nil, nil
	}

	if !strings.HasPrefix(input, "db.") {
		return nil, fmt.Errorf("commands must start with 'db.' (got %q)", input)
	}

	rest := input[3:]

	// db.createCollection("name")
	if strings.HasPrefix(rest, "createCollection(") {
		return parseCreateCollection(rest)
	}

	// db.dropCollection("name")
	if strings.HasPrefix(rest, "dropCollection(") {
		return parseDropCollection(rest)
	}

	// db.collection.method(args)
	dotIdx := strings.Index(rest, ".")
	if dotIdx < 0 {
		return nil, fmt.Errorf("expected db.collection.method(...), got %q", input)
	}

	collection := rest[:dotIdx]
	methodAndArgs := rest[dotIdx+1:]

	if strings.HasPrefix(methodAndArgs, "insert(") {
		return parseInsert(collection, methodAndArgs)
	}
	if strings.HasPrefix(methodAndArgs, "find(") {
		return parseFind(collection, methodAndArgs)
	}
	if strings.HasPrefix(methodAndArgs, "update(") {
		return parseUpdate(collection, methodAndArgs)
	}
	if strings.HasPrefix(methodAndArgs, "delete(") {
		return parseDelete(collection, methodAndArgs)
	}

	return nil, fmt.Errorf("unknown method in %q", input)
}

// parseCreateCollection handles db.createCollection("name").
func parseCreateCollection(s string) (*CreateCollectionStmt, error) {
	arg, err := extractSingleArg(s, "createCollection")
	if err != nil {
		return nil, err
	}
	name, err := unquote(arg)
	if err != nil {
		return nil, fmt.Errorf("createCollection: %w", err)
	}
	return &CreateCollectionStmt{Name: name}, nil
}

// parseDropCollection handles db.dropCollection("name").
func parseDropCollection(s string) (*DropCollectionStmt, error) {
	arg, err := extractSingleArg(s, "dropCollection")
	if err != nil {
		return nil, err
	}
	name, err := unquote(arg)
	if err != nil {
		return nil, fmt.Errorf("dropCollection: %w", err)
	}
	return &DropCollectionStmt{Name: name}, nil
}

// parseInsert handles db.collection.insert({...}).
func parseInsert(collection, s string) (*InsertStmt, error) {
	arg, err := extractSingleArg(s, "insert")
	if err != nil {
		return nil, err
	}
	doc, err := parseJSON(arg)
	if err != nil {
		return nil, fmt.Errorf("insert: invalid JSON: %w", err)
	}
	return &InsertStmt{Collection: collection, Document: doc}, nil
}

// parseFind handles db.collection.find({filter}).sort({field: order}).
func parseFind(collection, s string) (*FindStmt, error) {
	stmt := &FindStmt{Collection: collection}

	// Extract the find(...) part and optional .sort(...)
	// Find the matching closing paren for find(
	argStart := len("find(")
	depth := 1
	i := argStart
	for i < len(s) && depth > 0 {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
		}
		if depth > 0 {
			i++
		}
	}
	if depth != 0 {
		return nil, fmt.Errorf("find: unmatched parenthesis")
	}

	filterStr := strings.TrimSpace(s[argStart:i])
	remaining := ""
	if i+1 < len(s) {
		remaining = s[i+1:]
	}

	// Parse filter (empty filter = match all).
	if filterStr == "" || filterStr == "{}" {
		stmt.Filter = nil
	} else {
		filter, err := parseJSON(filterStr)
		if err != nil {
			return nil, fmt.Errorf("find: invalid filter JSON: %w", err)
		}
		stmt.Filter = filter
	}

	// Parse optional .sort({field: order}).
	remaining = strings.TrimSpace(remaining)
	if strings.HasPrefix(remaining, ".sort(") {
		sortArg, err := extractSingleArg(remaining[1:], "sort")
		if err != nil {
			return nil, fmt.Errorf("sort: %w", err)
		}
		sortMap, err := parseJSON(sortArg)
		if err != nil {
			return nil, fmt.Errorf("sort: invalid JSON: %w", err)
		}
		for field, order := range sortMap {
			stmt.SortField = field
			switch v := order.(type) {
			case float64:
				stmt.SortOrder = int(v)
			case int:
				stmt.SortOrder = v
			default:
				stmt.SortOrder = 1
			}
			break // only one sort field supported
		}
	}

	return stmt, nil
}

// parseUpdate handles db.collection.update({filter}, {$set: {...}}).
func parseUpdate(collection, s string) (*UpdateStmt, error) {
	args, err := extractTwoArgs(s, "update")
	if err != nil {
		return nil, err
	}

	filter, err := parseJSON(args[0])
	if err != nil {
		return nil, fmt.Errorf("update: invalid filter JSON: %w", err)
	}

	update, err := parseJSON(args[1])
	if err != nil {
		return nil, fmt.Errorf("update: invalid update JSON: %w", err)
	}

	return &UpdateStmt{Collection: collection, Filter: filter, Update: update}, nil
}

// parseDelete handles db.collection.delete({filter}).
func parseDelete(collection, s string) (*DeleteStmt, error) {
	arg, err := extractSingleArg(s, "delete")
	if err != nil {
		return nil, err
	}
	filter, err := parseJSON(arg)
	if err != nil {
		return nil, fmt.Errorf("delete: invalid filter JSON: %w", err)
	}
	return &DeleteStmt{Collection: collection, Filter: filter}, nil
}

// ── Helpers ──────────────────────────────────────────────────────────────────

// extractSingleArg extracts the argument from "method(arg)".
func extractSingleArg(s, method string) (string, error) {
	prefix := method + "("
	if !strings.HasPrefix(s, prefix) {
		return "", fmt.Errorf("expected %s(...)", method)
	}

	// Find matching close paren.
	rest := s[len(prefix):]
	depth := 1
	i := 0
	for i < len(rest) && depth > 0 {
		switch rest[i] {
		case '(':
			depth++
		case ')':
			depth--
		}
		if depth > 0 {
			i++
		}
	}
	if depth != 0 {
		return "", fmt.Errorf("%s: unmatched parenthesis", method)
	}

	return strings.TrimSpace(rest[:i]), nil
}

// extractTwoArgs extracts two comma-separated args from "method(arg1, arg2)".
func extractTwoArgs(s, method string) ([2]string, error) {
	prefix := method + "("
	if !strings.HasPrefix(s, prefix) {
		return [2]string{}, fmt.Errorf("expected %s(..., ...)", method)
	}

	rest := s[len(prefix):]

	// Find the matching close paren.
	depth := 1
	end := 0
	for end < len(rest) && depth > 0 {
		switch rest[end] {
		case '(', '{', '[':
			depth++
		case ')', '}', ']':
			depth--
		}
		if depth > 0 {
			end++
		}
	}
	if depth != 0 {
		return [2]string{}, fmt.Errorf("%s: unmatched parenthesis", method)
	}

	inner := rest[:end]

	// Split on comma at depth 0.
	commaIdx := -1
	depth = 0
	for i := 0; i < len(inner); i++ {
		switch inner[i] {
		case '{', '[', '(':
			depth++
		case '}', ']', ')':
			depth--
		case ',':
			if depth == 0 {
				commaIdx = i
				break
			}
		}
		if commaIdx >= 0 {
			break
		}
	}
	if commaIdx < 0 {
		return [2]string{}, fmt.Errorf("%s: expected two arguments separated by comma", method)
	}

	return [2]string{
		strings.TrimSpace(inner[:commaIdx]),
		strings.TrimSpace(inner[commaIdx+1:]),
	}, nil
}

// parseJSON parses a JSON object string into a map.
func parseJSON(s string) (map[string]any, error) {
	var result map[string]any
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// unquote removes surrounding double quotes from a string.
func unquote(s string) (string, error) {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1], nil
	}
	// Also support single quotes.
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		return s[1 : len(s)-1], nil
	}
	return "", fmt.Errorf("expected quoted string, got %q", s)
}
