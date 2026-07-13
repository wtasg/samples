# Inverted Index

## What Is It?

An **Inverted Index** is an index data structure storing a mapping from content
(words, terms, or field values) to its locations in a database document or set
of documents (posting lists).

```
"category": "electronics" ──▶ ["doc1", "doc2", "doc4"]
"category": "clothing"    ──▶ ["doc3"]
```

### Key Properties
- **Posting Lists** — lists of document IDs containing the value.
- **Fast Search** — turns collection scans into O(1) term lookups.
- **Prefix & Substring support** — enables wildcard lookups.

## Complexity

| Operation | Time |
|---|---|
| Search | O(1) (term lookup) |
| Insert | O(1) (append to list) |
| Delete | O(d) (where d = document fields to unindex) |

## Significance in Databases

Inverted Indexes are the engine behind full-text search:
- **Elasticsearch / Lucene** — core indexing strategy.
- **MongoDB** — secondary indexes on document fields.

## How It Is Used Here

In DocDB, the Inverted Index (`internal/ds/inverted.go`) handles secondary field
lookups. When a query filters by a field value (e.g. `db.products.find({"category": "electronics"})`),
the engine fetches document IDs from the Inverted Index and reads them from the Hash Map,
avoiding a slow full-collection scan. It also supports prefix search.
