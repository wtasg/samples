package docdb

import (
	"context"
	"encoding/json"
	"fmt"
)

// CollectionQuery is a fluent builder for queries against a single collection.
// Chain methods and call Find/Insert/Update/Delete to execute.
//
// Example:
//
//	res, err := c.Collection("users").
//	    Filter(docdb.M{"age": docdb.M{"$gt": 18}}).
//	    Sort("name", 1).
//	    Find(ctx)
type CollectionQuery struct {
	client         *Client
	collectionName string
	filter         M
	sortField      string
	sortOrder      int
}

// Filter sets the query filter.
func (q *CollectionQuery) Filter(filter M) *CollectionQuery {
	q2 := *q
	q2.filter = filter
	return &q2
}

// Sort sets the sorting configuration. order can be 1 (ascending) or -1 (descending).
func (q *CollectionQuery) Sort(field string, order int) *CollectionQuery {
	q2 := *q
	q2.sortField = field
	q2.sortOrder = order
	return &q2
}

// Find executes a find query and returns all matching documents.
// Uses the streaming Query RPC internally.
func (q *CollectionQuery) Find(ctx context.Context) (*Result, error) {
	cmd, err := q.buildFind()
	if err != nil {
		return nil, err
	}
	return q.client.Query(ctx, cmd)
}

// FindStream executes a find query and streams documents to the callback.
func (q *CollectionQuery) FindStream(ctx context.Context, fn func(doc Doc) error) error {
	cmd, err := q.buildFind()
	if err != nil {
		return err
	}
	return q.client.QueryStream(ctx, cmd, fn)
}

// Insert inserts a document into the collection.
func (q *CollectionQuery) Insert(ctx context.Context, doc Doc) error {
	cmd, err := q.buildInsert(doc)
	if err != nil {
		return err
	}
	_, err = q.client.Execute(ctx, cmd)
	return err
}

// Update modifies documents matching the query filter.
func (q *CollectionQuery) Update(ctx context.Context, update M) (int, error) {
	cmd, err := q.buildUpdate(update)
	if err != nil {
		return 0, err
	}
	res, err := q.client.Execute(ctx, cmd)
	if err != nil {
		return 0, err
	}
	var n int
	fmt.Sscanf(res.Message, "%d", &n)
	return n, nil
}

// Delete removes documents matching the query filter.
func (q *CollectionQuery) Delete(ctx context.Context) (int, error) {
	cmd, err := q.buildDelete()
	if err != nil {
		return 0, err
	}
	res, err := q.client.Execute(ctx, cmd)
	if err != nil {
		return 0, err
	}
	var n int
	fmt.Sscanf(res.Message, "%d", &n)
	return n, nil
}

// ── Query builders ──

func (q *CollectionQuery) buildFind() (string, error) {
	filterJSON := "{}"
	if q.filter != nil {
		b, err := json.Marshal(q.filter)
		if err != nil {
			return "", fmt.Errorf("serialize filter: %w", err)
		}
		filterJSON = string(b)
	}

	cmd := fmt.Sprintf("db.%s.find(%s)", q.collectionName, filterJSON)
	if q.sortField != "" {
		cmd += fmt.Sprintf(".sort({%q: %d})", q.sortField, q.sortOrder)
	}
	return cmd, nil
}

func (q *CollectionQuery) buildInsert(doc Doc) (string, error) {
	b, err := json.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("serialize doc: %w", err)
	}
	return fmt.Sprintf("db.%s.insert(%s)", q.collectionName, string(b)), nil
}

func (q *CollectionQuery) buildUpdate(update M) (string, error) {
	filterJSON := "{}"
	if q.filter != nil {
		b, err := json.Marshal(q.filter)
		if err != nil {
			return "", fmt.Errorf("serialize filter: %w", err)
		}
		filterJSON = string(b)
	}

	updateJSON, err := json.Marshal(update)
	if err != nil {
		return "", fmt.Errorf("serialize update: %w", err)
	}

	return fmt.Sprintf("db.%s.update(%s, %s)", q.collectionName, filterJSON, string(updateJSON)), nil
}

func (q *CollectionQuery) buildDelete() (string, error) {
	filterJSON := "{}"
	if q.filter != nil {
		b, err := json.Marshal(q.filter)
		if err != nil {
			return "", fmt.Errorf("serialize filter: %w", err)
		}
		filterJSON = string(b)
	}

	return fmt.Sprintf("db.%s.delete(%s)", q.collectionName, filterJSON), nil
}
